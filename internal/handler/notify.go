package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"dds-billing/internal/logic"
	"dds-billing/internal/model"
	"dds-billing/internal/payment"
	"dds-billing/internal/repo"

	"github.com/gin-gonic/gin"
)

type NotifyHandler struct {
	orderRepo      *repo.OrderRepo
	rechargeLogic  *logic.RechargeLogic
}

func NewNotifyHandler(orderRepo *repo.OrderRepo, rechargeLogic *logic.RechargeLogic) *NotifyHandler {
	return &NotifyHandler{
		orderRepo:     orderRepo,
		rechargeLogic: rechargeLogic,
	}
}

// Handle 通用支付回调处理 POST /api/notify/:provider
func (h *NotifyHandler) Handle(c *gin.Context) {
	providerName := c.Param("provider")

	provider, ok := payment.Get(providerName)
	if !ok {
		log.Printf("[notify] unknown provider: %s", providerName)
		c.String(http.StatusBadRequest, "FAIL")
		return
	}

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[notify] read body error: %v", err)
		c.String(http.StatusBadRequest, "FAIL")
		return
	}

	// Build params map from headers and query string
	notifyParams := map[string]string{}
	// Stripe webhook signature
	if sig := c.GetHeader("Stripe-Signature"); sig != "" {
		notifyParams["Stripe-Signature"] = sig
	}
	// 易支付协议回调通过 GET query 传参
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			notifyParams[k] = v[0]
		}
	}

	// Verify signature and parse notification
	notification, err := provider.VerifyNotification(c.Request.Context(), body, notifyParams)
	if err != nil {
		// 不关心的事件类型，返回 200 避免重试
		if errors.Is(err, payment.ErrEventIgnored) {
			log.Printf("[notify] %s: %v", providerName, err)
			c.String(http.StatusOK, "SUCCESS")
			return
		}
		log.Printf("[notify] verify failed for %s: %v", providerName, err)
		c.String(http.StatusBadRequest, "FAIL")
		return
	}

	log.Printf("[notify] payment confirmed: order=%s, trade=%s, amount=%s, type=%s",
		notification.OrderNo, notification.TradeNo, notification.Amount, notification.PaymentType)

	// Find order
	order, err := h.orderRepo.GetByOrderNo(notification.OrderNo)
	if err != nil {
		log.Printf("[notify] order not found: %s, err: %v", notification.OrderNo, err)
		c.String(http.StatusOK, "SUCCESS")
		return
	}

	// Skip if already processed (except expired - those need reconciliation)
	if order.Status != model.OrderStatusPending && order.Status != model.OrderStatusExpired {
		log.Printf("[notify] order %s already in status %s, skip", notification.OrderNo, order.Status)
		c.String(http.StatusOK, "SUCCESS")
		return
	}

	// If order was marked expired but payment succeeded, log reconciliation
	if order.Status == model.OrderStatusExpired {
		log.Printf("[notify] reconciliation: order %s was expired but payment confirmed, updating to paid", notification.OrderNo)
	}

	// Update order to paid
	now := time.Now()
	err = h.orderRepo.UpdateStatus(notification.OrderNo, model.OrderStatusPaid, map[string]interface{}{
		"trade_no":    notification.TradeNo,
		"pay_no":      notification.PayNo,
		"paid_at":     now,
		"notify_data": string(body),
	})
	if err != nil {
		log.Printf("[notify] update order %s failed: %v", notification.OrderNo, err)
		c.String(http.StatusInternalServerError, "FAIL")
		return
	}

	// Call Sub2API to recharge (async, don't block callback response)
	go func() {
		if err := h.rechargeLogic.ProcessRecharge(notification.OrderNo); err != nil {
			log.Printf("[notify] recharge order %s failed: %v", notification.OrderNo, err)
		}
	}()

	c.String(http.StatusOK, "SUCCESS")
}
