package main

import (
	"fmt"
	"log"

	"dds-billing/internal/config"
	"dds-billing/internal/handler"
	"dds-billing/internal/logic"
	"dds-billing/internal/model"
	"dds-billing/internal/payment"
	"dds-billing/internal/payment/easypay"
	stripepay "dds-billing/internal/payment/stripe"
	"dds-billing/internal/repo"
	"dds-billing/internal/sub2api"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Load config
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect database
	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&model.Order{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Init repositories
	orderRepo := repo.NewOrderRepo(db)

	// Init Sub2API client
	sub2apiClient := sub2api.NewClient(cfg.Sub2API.BaseURL, cfg.Sub2API.AdminAPIKey)

	// Register payment providers
	if cfg.Easypay.PID != "" {
		payment.Register("easypay", easypay.NewProvider(cfg.Easypay))
	}
	if cfg.Stripe.SecretKey != "" {
		payment.Register("stripe", stripepay.NewProvider(cfg.Stripe))
	}

	// Init business logic
	orderLogic := logic.NewOrderLogic(cfg, orderRepo, sub2apiClient)
	rechargeLogic := logic.NewRechargeLogic(orderRepo, sub2apiClient)

	// Setup router
	r := handler.SetupRouter(cfg, orderRepo, orderLogic, rechargeLogic)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
