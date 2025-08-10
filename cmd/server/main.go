package main

import (
	"log"
	"net"
	"time"

	"github.com/couchbase/gocb/v2"
	pb "github.com/ogozo/proto-definitions/gen/go/cart"
	"github.com/ogozo/service-cart/internal/broker"
	"github.com/ogozo/service-cart/internal/cart"
	"github.com/ogozo/service-cart/internal/config"
	"google.golang.org/grpc"
)

func main() {
	// 1. Yapılandırmayı yükle
	config.LoadConfig()
	cfg := config.AppConfig

	// Consumer'ı başlat
	consumer, err := broker.NewConsumer(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()
	log.Println("RabbitMQ consumer connected.")

	// 2. Couchbase bağlantısını yapılandırmadan alarak kur
	cluster, err := gocb.Connect(cfg.CouchbaseConnStr, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: cfg.CouchbaseUser,
			Password: cfg.CouchbasePass,
		},
	})
	if err != nil {
		log.Fatalf("Could not connect to Couchbase: %v", err)
	}

	bucket := cluster.Bucket(cfg.CouchbaseBucket)
	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		log.Fatalf("Could not get bucket %s: %v", cfg.CouchbaseBucket, err)
	}
	collection := bucket.DefaultCollection()
	log.Println("Couchbase connection successful for cart service.")

	// 3. Bağımlılıkları enjekte et
	cartRepo := cart.NewRepository(collection)
	cartService := cart.NewService(cartRepo)
	cartHandler := cart.NewHandler(cartService)

	if err := consumer.StartOrderConfirmedConsumer(cartService.HandleOrderConfirmedEvent); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	// 4. gRPC sunucusunu yapılandırmadan aldığı port ile başlat
	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", cfg.GRPCPort, err)
	}
	s := grpc.NewServer()
	pb.RegisterCartServiceServer(s, cartHandler)

	log.Printf("Cart gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
