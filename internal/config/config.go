package config

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type CartConfig struct {
	GRPCPort             string `mapstructure:"GRPC_PORT"`
	CouchbaseConnStr     string `mapstructure:"COUCHBASE_CONN_STR"`
	CouchbaseUser        string `mapstructure:"COUCHBASE_USER"`
	CouchbasePass        string `mapstructure:"COUCHBASE_PASS"`
	CouchbaseBucket      string `mapstructure:"COUCHBASE_BUCKET"`
	RabbitMQURL          string `mapstructure:"RABBITMQ_URL"`
	OtelExporterEndpoint string `mapstructure:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OtelServiceName      string `mapstructure:"OTEL_SERVICE_NAME"`
	MetricsPort          string `mapstructure:"METRICS_PORT"`
}

func LoadConfig(cfg any) {
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		tempLogger, _ := zap.NewProduction()
		defer tempLogger.Sync()
		tempLogger.Warn(".env file not found, reading from environment variables")
	}

	err := viper.Unmarshal(&cfg)
	if err != nil {
		tempLogger, _ := zap.NewProduction()
		defer tempLogger.Sync()
		tempLogger.Fatal("Unable to decode config into struct", zap.Error(err))
	}
}
