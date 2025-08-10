package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config, uygulamamızın tüm yapılandırma değerlerini tutan struct'tır.
type Config struct {
	GRPCPort         string `mapstructure:"GRPC_PORT"`
	CouchbaseConnStr string `mapstructure:"COUCHBASE_CONN_STR"`
	CouchbaseUser    string `mapstructure:"COUCHBASE_USER"`
	CouchbasePass    string `mapstructure:"COUCHBASE_PASS"`
	CouchbaseBucket  string `mapstructure:"COUCHBASE_BUCKET"`
	RabbitMQURL      string `mapstructure:"RABBITMQ_URL"`
}

var AppConfig *Config

// LoadConfig, yapılandırmayı .env dosyasından veya ortam değişkenlerinden okur.
func LoadConfig() {
	viper.AddConfigPath(".")    // config dosyasının aranacağı yer (proje ana dizini)
	viper.SetConfigName(".env") // config dosyasının adı
	viper.SetConfigType("env")  // config dosyasının tipi

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Warning: .env file not found, reading from environment variables")
	}

	err := viper.Unmarshal(&AppConfig)
	if err != nil {
		log.Fatalf("Unable to decode config into struct, %v", err)
	}
}
