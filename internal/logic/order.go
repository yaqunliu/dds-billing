package logic

import (
	"fmt"
	"log"
	"time"

	"dds-billing/internal/config"
	"dds-billing/internal/model"
	"dds-billing/internal/payment"
	"dds-billing/internal/repo"
	"dds-billing/internal/sub2api"
)

type OrderLogic struct {
	cfg       *config.Config
	orderRepo *repo.OrderRepo
	sub2api   *sub2api.Client
}

func NewOrderLogic(cfg *config.Config, orderRepo *repo.OrderRepo, sub2apiClient *sub2api.Client) *OrderLogic {
	return &OrderLogic{
		cfg:       cfg,
		orderRepo: orderRepo,
		sub2api:   sub2apiClient,
	}
}

type CreateOrderRequest struct {
	Token       string  `json:"token"`
	Amount      float64 `json:"amount"`
	PaymentType string  `json:"payment_type"` // wxpay / alipay
}

type CreateOrderResponse struct {
	OrderNo   string  `json:"order_no"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	QRCodeURL string  `json:"qr_code_url"`
	PayURL    string  `json:"pay_url"`
	ExpiresAt string  `json:"expires_at"`
}

func (l *OrderLogic) CreateOrder(req CreateOrderRequest) (*CreateOrderResponse, error) {
	// 1. Verify user token
	user, err := l.sub2api.VerifyUser(req.Token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// 2. Validate amount
	if req.Amount < l.cfg.Billing.MinAmount || req.Amount > l.cfg.Billing.MaxAmount {
		return nil, fmt.Errorf("amount must be between %.2f and %.2f", l.cfg.Billing.MinAmount, l.cfg.Billing.MaxAmount)
	}

	// 3. Validate payment type
	payType := payment.PaymentType(req.PaymentType)
	if payType != payment.PaymentTypeWxpay && payType != payment.PaymentTypeAlipay {
		return nil, fmt.Errorf("unsupported payment type: %s", req.PaymentType)
	}

	// 4. Get active payment provider
	provider := payment.GetActive(l.cfg)

	// 5. Generate order number
	orderNo := fmt.Sprintf("ORD%s%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%10000)
	expiresAt := time.Now().Add(time.Duration(l.cfg.Billing.OrderTimeoutMinutes) * time.Minute)

	// 6. Create payment via provider
	amountStr := fmt.Sprintf("%.2f", req.Amount)
	payResp, err := provider.CreatePayment(nil, payment.CreatePaymentRequest{
		OrderNo:     orderNo,
		Amount:      amountStr,
		Subject:     "VIP会员",
		PaymentType: payType,
	})
	if err != nil {
		log.Printf("[order] payment provider error: %v", err)
		return nil, fmt.Errorf("暂时无法充值，请联系客服人员")
	}

	// 7. Save order to database
	order := &model.Order{
		OrderNo:     orderNo,
		UserID:      user.ID,
		UserEmail:   user.Email,
		Amount:      req.Amount,
		Status:      model.OrderStatusPending,
		PaymentType: req.PaymentType,
		Provider:    provider.Name(),
		TradeNo:     payResp.TradeNo,
		QRCodeURL:   payResp.QRCodeURL,
		PayURL:      payResp.PayURL,
		ExpiresAt:   expiresAt,
	}
	if err := l.orderRepo.Create(order); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}

	log.Printf("[order] created: no=%s, user=%d, amount=%.2f, provider=%s", orderNo, user.ID, req.Amount, provider.Name())

	return &CreateOrderResponse{
		OrderNo:   orderNo,
		Amount:    req.Amount,
		Status:    string(model.OrderStatusPending),
		QRCodeURL: payResp.QRCodeURL,
		PayURL:    payResp.PayURL,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}
