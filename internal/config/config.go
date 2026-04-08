package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Payment  PaymentConfig  `yaml:"payment"`
	Easypay  EasypayConfig  `yaml:"easypay"`
	Zpay     ZpayConfig     `yaml:"zpay"`
	Stripe   StripeConfig   `yaml:"stripe"`
	Sub2API  Sub2APIConfig  `yaml:"sub2api"`
	Billing  BillingConfig  `yaml:"billing"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type PaymentConfig struct {
	Provider     string   `yaml:"provider"`
	EnabledTypes []string `yaml:"enabled_types"`
}

type EasypayConfig struct {
	PID       string `yaml:"pid"`
	PKey      string `yaml:"pkey"`
	APIBase   string `yaml:"api_base"`
	NotifyURL string `yaml:"notify_url"`
}

type ZpayConfig struct {
	MchID     string `yaml:"mch_id"`
	SecretKey string `yaml:"secret_key"`
	NotifyURL string `yaml:"notify_url"`
}

type StripeConfig struct {
	SecretKey      string `yaml:"secret_key"`
	PublishableKey string `yaml:"publishable_key"`
	WebhookSecret  string `yaml:"webhook_secret"`
	NotifyURL      string `yaml:"notify_url"`
	SuccessURL     string `yaml:"success_url"`
	CancelURL      string `yaml:"cancel_url"`
}

type Sub2APIConfig struct {
	BaseURL     string `yaml:"base_url"`
	AdminAPIKey string `yaml:"admin_api_key"`
}

type BillingConfig struct {
	MinAmount           float64 `yaml:"min_amount"`
	MaxAmount           float64 `yaml:"max_amount"`
	OrderTimeoutMinutes int     `yaml:"order_timeout_minutes"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Expand environment variables in config
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 3000
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Billing.OrderTimeoutMinutes == 0 {
		cfg.Billing.OrderTimeoutMinutes = 5
	}

	return &cfg, nil
}
