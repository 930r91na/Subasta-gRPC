package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/930r91na/Subasta-grpc/pkg/auction"
	"google.golang.org/grpc"
)

// AuctionServer implements the gRPC service
type AuctionServer struct {
	pb.UnimplementedAuctionServiceServer
	mu       sync.RWMutex
	users    map[string]string
	products map[string]*pb.ProductInfo
	bids     map[string]*pb.BidInfo
}

// NewAuctionServer creates a new auction server instance
func NewAuctionServer() *AuctionServer {
	return &AuctionServer{
		users:    make(map[string]string),
		products: make(map[string]*pb.ProductInfo),
		bids:     make(map[string]*pb.BidInfo),
	}
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
		log.Printf("Adding new product: %s", product)
		s.products[product] = &pb.ProductInfo{
			Seller:       req.GetSeller(),
			Product:      product,
			InitialPrice: req.GetInitialPrice(),
			CurrentPrice: req.GetInitialPrice(),
		}
		return &pb.AddProductResponse{
			Success: true,
			Message: fmt.Sprintf("Product %s added successfully", product),
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

// GetCatalog returns all products in the catalog
func (s *AuctionServer) GetCatalog(ctx context.Context, req *pb.GetCatalogRequest) (*pb.GetCatalogResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	products := make([]*pb.ProductInfo, 0, len(s.products))
	for _, prod := range s.products {
		products = append(products, prod)
	}

	log.Printf("Sending catalog with %d products", len(products))
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
