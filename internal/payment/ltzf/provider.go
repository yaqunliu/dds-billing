package ltzf

import (
	"context"
	"encoding/json"
	"fmt"

	"dds-billing/internal/config"
	"dds-billing/internal/payment"
)

type Provider struct {
	client *Client
	cfg    config.LtzfConfig
}

func NewProvider(cfg config.LtzfConfig) *Provider {
	return &Provider{
		client: NewClient(cfg.MchID, cfg.SecretKey),
		cfg:    cfg,
	}
}

func (p *Provider) Name() string {
	return "ltzf"
}

func (p *Provider) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.PaymentTypeWxpay, payment.PaymentTypeAlipay}
}

func (p *Provider) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = p.cfg.NotifyURL
	}

	payData, err := p.client.CreateNativePayment(
		string(req.PaymentType),
		req.OrderNo,
		req.Amount,
		req.Subject,
		notifyURL,
	)
	if err != nil {
		return nil, fmt.Errorf("ltzf create payment: %w", err)
	}

	return &payment.CreatePaymentResponse{
		TradeNo:   payData.OrderNo,
		PayURL:    payData.CodeURL,
		QRCodeURL: payData.QRCodeURL,
	}, nil
}

func (p *Provider) VerifyNotification(ctx context.Context, body []byte, params map[string]string) (*payment.PaymentNotification, error) {
	// Parse form body if params not provided
	formParams := params
	if len(formParams) == 0 {
		var err error
		formParams, err = ParseFormToMap(body)
		if err != nil {
			return nil, fmt.Errorf("parse notify body: %w", err)
		}
	}

	// Verify signature
	if !VerifySign(formParams, p.cfg.SecretKey) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Check code
	if formParams["code"] != "0" {
		return nil, fmt.Errorf("payment not successful, code=%s", formParams["code"])
	}

	// Map pay_channel to PaymentType
	var payType payment.PaymentType
	switch formParams["pay_channel"] {
	case "wxpay":
		payType = payment.PaymentTypeWxpay
	case "alipay":
		payType = payment.PaymentTypeAlipay
	default:
		payType = payment.PaymentType(formParams["pay_channel"])
	}

	return &payment.PaymentNotification{
		OrderNo:     formParams["out_trade_no"],
		TradeNo:     formParams["order_no"],
		PayNo:       formParams["pay_no"],
		Amount:      formParams["total_fee"],
		PaymentType: payType,
		PaidAt:      formParams["success_time"],
	}, nil
}

func (p *Provider) QueryOrder(ctx context.Context, orderNo string) (*payment.PaymentNotification, error) {
	resp, err := p.client.QueryOrder(orderNo)
	if err != nil {
		return nil, fmt.Errorf("ltzf query order: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("ltzf query failed: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	// Parse response data
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("parse query response: %w", err)
	}

	getString := func(key string) string {
		if v, ok := data[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	var payType payment.PaymentType
	switch getString("pay_channel") {
	case "wxpay":
		payType = payment.PaymentTypeWxpay
	case "alipay":
		payType = payment.PaymentTypeAlipay
	}

	return &payment.PaymentNotification{
		OrderNo:     getString("out_trade_no"),
		TradeNo:     getString("order_no"),
		PayNo:       getString("pay_no"),
		Amount:      getString("total_fee"),
		PaymentType: payType,
		PaidAt:      getString("success_time"),
	}, nil
}
