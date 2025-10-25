package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/930r91na/Subasta-grpc/auction" // Update this import path
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuctionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Example 1: Register users
	fmt.Println("=== Registering Users ===")
	users := []string{"John", "Mary", "Peter"}
	for _, name := range users {
		resp, err := client.RegisterUser(ctx, &pb.RegisterUserRequest{
			Name: name,
		})
		if err != nil {
			log.Printf("Error registering user: %v", err)
			continue
		}
		fmt.Printf("User %s: %s (Success: %v)\n", name, resp.Message, resp.Success)
	}

	// Example 2: Add products for sale
	fmt.Println("\n=== Adding Products ===")
	products := []struct {
		seller       string
		product      string
		initialPrice float32
	}{
		{"John", "Laptop", 500.0},
		{"Mary", "Phone", 300.0},
		{"Peter", "Tablet", 200.0},
	}

	for _, p := range products {
		resp, err := client.AddProduct(ctx, &pb.AddProductRequest{
			Seller:       p.seller,
			Product:      p.product,
			InitialPrice: p.initialPrice,
		})
		if err != nil {
			log.Printf("Error adding product: %v", err)
			continue
		}
		fmt.Printf("Product %s: %s (Success: %v)\n", p.product, resp.Message, resp.Success)
	}

	// Example 3: View catalog
	fmt.Println("\n=== Product Catalog ===")
	catalogResp, err := client.GetCatalog(ctx, &pb.GetCatalogRequest{})
	if err != nil {
		log.Fatalf("Error getting catalog: %v", err)
	}

	for _, prod := range catalogResp.Products {
		fmt.Printf("- %s (Seller: %s, Initial Price: $%.2f, Current Price: $%.2f)\n",
			prod.Product, prod.Seller, prod.InitialPrice, prod.CurrentPrice)
	}

	// Example 4: Place bids
	fmt.Println("\n=== Placing Bids ===")
	bids := []struct {
		buyer   string
		product string
		amount  float32
	}{
		{"Mary", "Laptop", 550.0},
		{"Peter", "Laptop", 600.0},
		{"John", "Phone", 350.0},
		{"Peter", "Laptop", 580.0}, // This should fail (lower than current)
	}

	for _, b := range bids {
		resp, err := client.PlaceBid(ctx, &pb.PlaceBidRequest{
			Buyer:   b.buyer,
			Product: b.product,
			Amount:  b.amount,
		})
		if err != nil {
			log.Printf("Error placing bid: %v", err)
			continue
		}
		fmt.Printf("%s bids $%.2f for %s: %s (Success: %v, Current Price: $%.2f)\n",
			b.buyer, b.amount, b.product, resp.Message, resp.Success, resp.CurrentPrice)
	}

	// Example 5: View updated catalog
	fmt.Println("\n=== Updated Catalog ===")
	catalogResp, err = client.GetCatalog(ctx, &pb.GetCatalogRequest{})
	if err != nil {
		log.Fatalf("Error getting catalog: %v", err)
	}

	for _, prod := range catalogResp.Products {
		fmt.Printf("- %s (Current Price: $%.2f)\n", prod.Product, prod.CurrentPrice)
	}

	// Example 6: Get specific product
	fmt.Println("\n=== Specific Product Information ===")
	prodResp, err := client.GetProduct(ctx, &pb.GetProductRequest{
		Product: "Laptop",
	})
	if err != nil {
		log.Fatalf("Error getting product: %v", err)
	}

	if prodResp.Found {
		p := prodResp.Product
		fmt.Printf("Product: %s\n", p.Product)
		fmt.Printf("Seller: %s\n", p.Seller)
		fmt.Printf("Initial Price: $%.2f\n", p.InitialPrice)
		fmt.Printf("Current Price: $%.2f\n", p.CurrentPrice)
	} else {
		fmt.Println("Product not found")
	}
}
