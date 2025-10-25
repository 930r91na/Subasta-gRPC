package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	pb "github.com/930r91na/Subasta-grpc/auction"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var grpcClient pb.AuctionServiceClient

func main() {
	// Connect to gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	grpcClient = pb.NewAuctionServiceClient(conn)

	// Enable CORS for all routes
	http.HandleFunc("/auction.AuctionService/RegisterUser", corsMiddleware(handleRegisterUser))
	http.HandleFunc("/auction.AuctionService/GetCatalog", corsMiddleware(handleGetCatalog))
	http.HandleFunc("/auction.AuctionService/PlaceBid", corsMiddleware(handlePlaceBid))
	http.HandleFunc("/auction.AuctionService/AddProduct", corsMiddleware(handleAddProduct))
	http.HandleFunc("/auction.AuctionService/GetProduct", corsMiddleware(handleGetProduct))

	// Serve static files from Client directory
	fs := http.FileServer(http.Dir("../Client"))
	http.Handle("/", fs)

	log.Println("🚀 Web UI Server started on http://localhost:8080")
	log.Println("📂 Serving files from Client directory")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := grpcClient.RegisterUser(ctx, &pb.RegisterUserRequest{Name: req.Name})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.Success,
		"message": resp.Message,
	})
}

func handleGetCatalog(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := grpcClient.GetCatalog(ctx, &pb.GetCatalogRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	products := make([]map[string]interface{}, 0)
	for _, p := range resp.Products {
		products = append(products, map[string]interface{}{
			"seller":        p.Seller,
			"product":       p.Product,
			"initial_price": p.InitialPrice,
			"current_price": p.CurrentPrice,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"products": products,
	})
}

func handlePlaceBid(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Buyer   string  `json:"buyer"`
		Product string  `json:"product"`
		Amount  float64 `json:"amount"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := grpcClient.PlaceBid(ctx, &pb.PlaceBidRequest{
		Buyer:   req.Buyer,
		Product: req.Product,
		Amount:  float32(req.Amount),
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       resp.Success,
		"message":       resp.Message,
		"current_price": resp.CurrentPrice,
	})
}

func handleAddProduct(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Seller       string  `json:"seller"`
		Product      string  `json:"product"`
		InitialPrice float64 `json:"initial_price"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := grpcClient.AddProduct(ctx, &pb.AddProductRequest{
		Seller:       req.Seller,
		Product:      req.Product,
		InitialPrice: float32(req.InitialPrice),
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": resp.Success,
		"message": resp.Message,
	})
}

func handleGetProduct(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Product string `json:"product"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := grpcClient.GetProduct(ctx, &pb.GetProductRequest{Product: req.Product})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"found": resp.Found,
	}

	if resp.Found && resp.Product != nil {
		result["product"] = map[string]interface{}{
			"seller":        resp.Product.Seller,
			"product":       resp.Product.Product,
			"initial_price": resp.Product.InitialPrice,
			"current_price": resp.Product.CurrentPrice,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
