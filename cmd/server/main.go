package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/couchbase/gocb/v2"
	pb "github.com/ogozo/proto-definitions/gen/go/cart"
	"github.com/ogozo/service-cart/internal/broker"
	"github.com/ogozo/service-cart/internal/cart"
	"github.com/ogozo/service-cart/internal/config"
	"github.com/ogozo/service-cart/internal/observability"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	config.LoadConfig()
	cfg := config.AppConfig

	shutdown, err := observability.InitTracerProvider(ctx, cfg.OtelServiceName, cfg.OtelExporterEndpoint)
	if err != nil {
		log.Fatalf("failed to initialize tracer provider: %v", err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatalf("failed to shutdown tracer provider: %v", err)
		}
	}()

	consumer, err := broker.NewConsumer(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()
	log.Println("RabbitMQ consumer connected.")

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

	cartRepo := cart.NewRepository(collection)
	cartService := cart.NewService(cartRepo)
	cartHandler := cart.NewHandler(cartService)

	if err := consumer.StartOrderConfirmedConsumer(cartService.HandleOrderConfirmedEvent); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", cfg.GRPCPort, err)
	}
	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	pb.RegisterCartServiceServer(s, cartHandler)

	log.Printf("Cart gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
