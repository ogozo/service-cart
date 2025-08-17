package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	gocbopentelemetry "github.com/couchbase/gocb-opentelemetry"
	"github.com/couchbase/gocb/v2"
	"github.com/ogozo/proto-definitions/gen/go/cart"
	"github.com/ogozo/service-cart/internal/broker"
	internalCart "github.com/ogozo/service-cart/internal/cart"
	"github.com/ogozo/service-cart/internal/config"
	"github.com/ogozo/service-cart/internal/logging"
	"github.com/ogozo/service-cart/internal/observability"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func startMetricsServer(l *zap.Logger, port string) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		l.Info("metrics server started", zap.String("port", port))
		if err := http.ListenAndServe(port, mux); err != nil {
			l.Fatal("failed to start metrics server", zap.Error(err))
		}
	}()
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var cfg config.CartConfig
	config.LoadConfig(&cfg)

	logging.Init(cfg.OtelServiceName)
	defer logging.Sync()

	logger := logging.FromContext(ctx)

	shutdown, err := observability.InitTracerProvider(ctx, cfg.OtelServiceName, cfg.OtelExporterEndpoint, logger)
	if err != nil {
		logger.Fatal("failed to initialize tracer provider", zap.Error(err))
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			logger.Fatal("failed to shutdown tracer provider", zap.Error(err))
		}
	}()

	startMetricsServer(logger, cfg.MetricsPort)

	consumer, err := broker.NewConsumer(cfg.RabbitMQURL)
	if err != nil {
		logger.Fatal("failed to create consumer", zap.Error(err))
	}
	defer consumer.Close()
	logger.Info("RabbitMQ consumer connected")

	tp := otel.GetTracerProvider()

	cluster, err := gocb.Connect(cfg.CouchbaseConnStr, gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: cfg.CouchbaseUser,
			Password: cfg.CouchbasePass,
		},
		Tracer: gocbopentelemetry.NewOpenTelemetryRequestTracer(tp),
	})
	if err != nil {
		logger.Fatal("could not connect to Couchbase", zap.Error(err))
	}

	bucket := cluster.Bucket(cfg.CouchbaseBucket)
	if err = bucket.WaitUntilReady(5*time.Second, nil); err != nil {
		logger.Fatal("could not get bucket", zap.Error(err), zap.String("bucket", cfg.CouchbaseBucket))
	}
	collection := bucket.DefaultCollection()
	logger.Info("Couchbase connection successful, with OTel instrumentation")

	cartRepo := internalCart.NewRepository(collection)
	cartService := internalCart.NewService(cartRepo)
	cartHandler := internalCart.NewHandler(cartService)

	if err := consumer.StartOrderConfirmedConsumer(cartService.HandleOrderConfirmedEvent); err != nil {
		logger.Fatal("failed to start consumer", zap.Error(err))
	}

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err), zap.String("port", cfg.GRPCPort))
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	cart.RegisterCartServiceServer(s, cartHandler)

	logger.Info("gRPC server listening", zap.String("address", lis.Addr().String()))
	if err := s.Serve(lis); err != nil {
		logger.Fatal("failed to serve gRPC", zap.Error(err))
	}
}
