package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/930r91na/Subasta-grpc/pkg/auction"
	"google.golang.org/grpc"
)

// AuctionServer implements the gRPC service
type AuctionServer struct {
	pb.UnimplementedAuctionServiceServer
	mu              sync.RWMutex
	users           map[string]string
	products        map[string]*pb.ProductInfo
	bids            map[string]*pb.BidInfo
	purchasedItems  map[string][]*pb.PurchasedItem // user -> purchased items
	highestBidders  map[string]string               // product -> buyer with highest bid
}

// NewAuctionServer creates a new auction server instance
func NewAuctionServer() *AuctionServer {
	server := &AuctionServer{
		users:          make(map[string]string),
		products:       make(map[string]*pb.ProductInfo),
		bids:           make(map[string]*pb.BidInfo),
		purchasedItems: make(map[string][]*pb.PurchasedItem),
		highestBidders: make(map[string]string),
	}

	// Start background goroutine to check for expired auctions
	go server.checkExpiredAuctions()

	return server
}

// RegisterUser registers a new user in the system
func (s *AuctionServer) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := req.GetName()

	if _, exists := s.users[name]; !exists {
		log.Printf("Adding new user: %s", name)
		s.users[name] = name
		return &pb.RegisterUserResponse{
			Success: true,
			Message: fmt.Sprintf("User %s registered successfully", name),
		}, nil
	}

	return &pb.RegisterUserResponse{
		Success: false,
		Message: fmt.Sprintf("User %s already exists", name),
	}, nil
}

// AddProduct adds a product for sale
func (s *AuctionServer) AddProduct(ctx context.Context, req *pb.AddProductRequest) (*pb.AddProductResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product := req.GetProduct()

	if _, exists := s.products[product]; !exists {
		// Calculate auction end time
		duration := req.GetAuctionDurationSeconds()
		if duration <= 0 {
			duration = 3600 // Default to 1 hour if not specified
		}
		endTime := time.Now().Unix() + int64(duration)

		log.Printf("Adding new product: %s (auction ends in %d seconds)", product, duration)
		s.products[product] = &pb.ProductInfo{
			Seller:          req.GetSeller(),
			Product:         product,
			InitialPrice:    req.GetInitialPrice(),
			CurrentPrice:    req.GetInitialPrice(),
			AuctionEndTime:  endTime,
			IsActive:        true,
		}
		return &pb.AddProductResponse{
			Success: true,
			Message: fmt.Sprintf("Product %s added successfully. Auction ends at %s", product, time.Unix(endTime, 0).Format(time.RFC3339)),
		}, nil
	}

	return &pb.AddProductResponse{
		Success: false,
		Message: fmt.Sprintf("Product %s already exists", product),
	}, nil
}

// PlaceBid places a bid on a product
func (s *AuctionServer) PlaceBid(ctx context.Context, req *pb.PlaceBidRequest) (*pb.PlaceBidResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product := req.GetProduct()
	buyer := req.GetBuyer()
	amount := req.GetAmount()

	productInfo, exists := s.products[product]
	if !exists {
		return &pb.PlaceBidResponse{
			Success: false,
			Message: fmt.Sprintf("Product %s does not exist", product),
		}, nil
	}

	// Check if auction is still active
	if !productInfo.IsActive {
		return &pb.PlaceBidResponse{
			Success:      false,
			Message:      "Auction has ended",
			CurrentPrice: productInfo.CurrentPrice,
		}, nil
	}

	// Check if auction time has expired
	if time.Now().Unix() > productInfo.AuctionEndTime {
		return &pb.PlaceBidResponse{
			Success:      false,
			Message:      "Auction time has expired",
			CurrentPrice: productInfo.CurrentPrice,
		}, nil
	}

	// Check if bid is higher than current price (updatePrice logic)
	if amount > productInfo.CurrentPrice {
		productInfo.CurrentPrice = amount

		// Store the bid
		key := product + buyer
		s.bids[key] = &pb.BidInfo{
			Buyer:   buyer,
			Product: product,
			Amount:  amount,
		}

		// Track highest bidder
		s.highestBidders[product] = buyer

		log.Printf("Bid accepted: %s offers %.2f for %s", buyer, amount, product)
		return &pb.PlaceBidResponse{
			Success:      true,
			Message:      fmt.Sprintf("Bid accepted for %.2f", amount),
			CurrentPrice: productInfo.CurrentPrice,
		}, nil
	}

	return &pb.PlaceBidResponse{
		Success:      false,
		Message:      fmt.Sprintf("Bid must be higher than %.2f", productInfo.CurrentPrice),
		CurrentPrice: productInfo.CurrentPrice,
	}, nil
}

// GetCatalog returns all active products in the catalog
func (s *AuctionServer) GetCatalog(ctx context.Context, req *pb.GetCatalogRequest) (*pb.GetCatalogResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	products := make([]*pb.ProductInfo, 0, len(s.products))
	for _, prod := range s.products {
		// Only include active auctions
		if prod.IsActive {
			products = append(products, prod)
		}
	}

	log.Printf("Sending catalog with %d active products", len(products))
	return &pb.GetCatalogResponse{
		Products: products,
	}, nil
}

// GetProduct returns information about a specific product
func (s *AuctionServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product := req.GetProduct()
	if prod, exists := s.products[product]; exists {
		return &pb.GetProductResponse{
			Found:   true,
			Product: prod,
		}, nil
	}

	return &pb.GetProductResponse{
		Found: false,
	}, nil
}

// GetPurchasedItems returns all items purchased by a user
func (s *AuctionServer) GetPurchasedItems(ctx context.Context, req *pb.GetPurchasedItemsRequest) (*pb.GetPurchasedItemsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buyer := req.GetBuyer()
	items := s.purchasedItems[buyer]

	if items == nil {
		items = []*pb.PurchasedItem{}
	}

	log.Printf("Sending %d purchased items for user %s", len(items), buyer)
	return &pb.GetPurchasedItemsResponse{
		Items: items,
	}, nil
}

// checkExpiredAuctions runs in the background to check for and process expired auctions
func (s *AuctionServer) checkExpiredAuctions() {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for range ticker.C {
		s.processExpiredAuctions()
	}
}

// processExpiredAuctions finds expired auctions and transfers items to winners
func (s *AuctionServer) processExpiredAuctions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()

	for productName, productInfo := range s.products {
		// Skip if already inactive or not yet expired
		if !productInfo.IsActive || now <= productInfo.AuctionEndTime {
			continue
		}

		// Auction has expired
		log.Printf("Auction expired for product: %s", productName)

		// Mark as inactive
		productInfo.IsActive = false

		// Check if there's a winner (highest bidder)
		if winner, hasWinner := s.highestBidders[productName]; hasWinner {
			// Create purchased item
			purchasedItem := &pb.PurchasedItem{
				Product:       productName,
				Seller:        productInfo.Seller,
				PurchasePrice: productInfo.CurrentPrice,
				PurchaseTime:  now,
			}

			// Add to winner's purchased items
			s.purchasedItems[winner] = append(s.purchasedItems[winner], purchasedItem)

			log.Printf("Product %s sold to %s for %.2f", productName, winner, productInfo.CurrentPrice)
		} else {
			log.Printf("Product %s auction ended with no bids", productName)
		}
	}
}

func main() {
	// Create a TCP listener
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register the auction service
	pb.RegisterAuctionServiceServer(grpcServer, NewAuctionServer())

	log.Println("Auction server started on port 50051...")

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
