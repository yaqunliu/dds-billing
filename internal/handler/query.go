package handler

import (
	"fmt"
	"net/http"

	"dds-billing/internal/repo"

	"github.com/gin-gonic/gin"
)

type QueryHandler struct {
	orderRepo *repo.OrderRepo
}

func NewQueryHandler(orderRepo *repo.OrderRepo) *QueryHandler {
	return &QueryHandler{orderRepo: orderRepo}
}

// Query GET /api/orders/:order_no
func (h *QueryHandler) Query(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "order_no is required"})
		return
	}

	order, err := h.orderRepo.GetByOrderNo(orderNo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order_no":     order.OrderNo,
			"status":       order.Status,
			"amount":       order.Amount,
			"payment_type": order.PaymentType,
			"qr_code_url":  order.QRCodeURL,
			"expires_at":   order.ExpiresAt,
			"paid_at":      order.PaidAt,
			"completed_at": order.CompletedAt,
			"created_at":   order.CreatedAt,
		},
	})
}

// List GET /api/orders
func (h *QueryHandler) List(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "token is required"})
		return
	}

	// For now, use user_id from query param (frontend passes it)
	// In production, should verify token and extract user_id
	userID := c.GetInt64("user_id")
	if userID == 0 {
		// Try from query param
		var uid int64
		if _, err := fmt.Sscanf(c.Query("user_id"), "%d", &uid); err == nil {
			userID = uid
		}
	}
	if userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "user_id is required"})
		return
	}

	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	orders, total, err := h.orderRepo.ListByUserID(userID, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "query failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  orders,
			"total": total,
		},
	})
}
