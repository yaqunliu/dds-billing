package handler

import (
	"dds-billing/internal/config"
	"dds-billing/internal/logic"
	"dds-billing/internal/middleware"
	"dds-billing/internal/repo"

	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config, orderRepo *repo.OrderRepo, orderLogic *logic.OrderLogic, rechargeLogic *logic.RechargeLogic) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(middleware.CORS())

	// Health check
	r.GET("/health", HealthCheck)

	api := r.Group("/api")
	{
		// Config endpoint
		api.GET("/config", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"code": 0,
				"data": gin.H{
					"enabled_types": cfg.Payment.EnabledTypes,
					"min_amount":    cfg.Billing.MinAmount,
					"max_amount":    cfg.Billing.MaxAmount,
				},
			})
		})

		// Order routes
		orderHandler := NewOrderHandler(orderLogic)
		api.POST("/orders", orderHandler.Create)

		queryHandler := NewQueryHandler(orderRepo, orderLogic)
		api.GET("/orders/:order_no", queryHandler.Query)
		api.GET("/orders", queryHandler.List)

		// Payment notify callback (POST for Stripe, GET for easypay protocol)
		notifyHandler := NewNotifyHandler(orderRepo, rechargeLogic)
		api.POST("/notify/:provider", notifyHandler.Handle)
		api.GET("/notify/:provider", notifyHandler.Handle)
	}

	return r
}
