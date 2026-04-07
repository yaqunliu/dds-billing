package handler

import (
	"net/http"

	"dds-billing/internal/logic"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderLogic *logic.OrderLogic
}

func NewOrderHandler(orderLogic *logic.OrderLogic) *OrderHandler {
	return &OrderHandler{orderLogic: orderLogic}
}

// Create POST /api/orders
func (h *OrderHandler) Create(c *gin.Context) {
	var req logic.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid request: " + err.Error()})
		return
	}

	if req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "token is required"})
		return
	}

	resp, err := h.orderLogic.CreateOrder(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}
