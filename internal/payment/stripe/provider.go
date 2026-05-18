package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"dds-billing/internal/config"
	"dds-billing/internal/payment"

	gostripe "github.com/stripe/stripe-go/v82"
)

type Provider struct {
	client *Client
	cfg    config.StripeConfig
}

func NewProvider(cfg config.StripeConfig) *Provider {
	return &Provider{
		client: NewClient(cfg),
		cfg:    cfg,
	}
}

func (p *Provider) Name() string {
	return "stripe"
}

func (p *Provider) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.PaymentTypeWxpay, payment.PaymentTypeAlipay}
}

// stripeMethodType 将业务 PaymentType 映射为 Stripe 的 payment_method_types 值
func stripeMethodType(t payment.PaymentType) string {
	switch t {
	case payment.PaymentTypeWxpay:
		return "wechat_pay"
	case payment.PaymentTypeAlipay:
		return "alipay"
	}
	return ""
}

// paymentTypeFromStripe 将 Stripe 的 payment_method_types 值映射为业务 PaymentType
func paymentTypeFromStripe(s string) payment.PaymentType {
	switch s {
	case "wechat_pay":
		return payment.PaymentTypeWxpay
	case "alipay":
		return payment.PaymentTypeAlipay
	}
	return ""
}

// nextActionURL 从 PaymentIntent.NextAction 中读取对应支付方式的跳转/二维码链接
func nextActionURL(pi *gostripe.PaymentIntent, methodType string) string {
	if pi == nil || pi.NextAction == nil {
		return ""
	}
	switch methodType {
	case "wechat_pay":
		if pi.NextAction.WeChatPayDisplayQRCode != nil {
			return pi.NextAction.WeChatPayDisplayQRCode.Data
		}
	case "alipay":
		if pi.NextAction.AlipayHandleRedirect != nil {
			return pi.NextAction.AlipayHandleRedirect.URL
		}
	}
	return ""
}

// paymentIntentToNotification 把 PaymentIntent 转换为通知结构（webhook 与查询共用）
func paymentIntentToNotification(pi *gostripe.PaymentIntent) *payment.PaymentNotification {
	orderNo := ""
	if v, ok := pi.Metadata["order_no"]; ok {
		orderNo = v
	}
	amountYuan := fmt.Sprintf("%.2f", float64(pi.Amount)/100)
	var payType payment.PaymentType
	if len(pi.PaymentMethodTypes) > 0 {
		payType = paymentTypeFromStripe(pi.PaymentMethodTypes[0])
	}
	return &payment.PaymentNotification{
		OrderNo:     orderNo,
		TradeNo:     pi.ID,
		PayNo:       pi.ID,
		Amount:      amountYuan,
		PaymentType: payType,
	}
}

func (p *Provider) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	// 将金额从元转为分（Stripe 使用最小货币单位）
	amountFloat, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	amountCents := int64(math.Round(amountFloat * 100))

	if ctx == nil {
		ctx = context.Background()
	}

	methodType := stripeMethodType(req.PaymentType)
	if methodType == "" {
		return nil, fmt.Errorf("unsupported payment type: %s", req.PaymentType)
	}

	// alipay 必须带 return_url；wechat_pay 不需要
	returnURL := ""
	if methodType == "alipay" {
		returnURL = p.cfg.SuccessURL + "?order_no=" + req.OrderNo + "&status=success"
	}

	pi, err := p.client.CreatePaymentIntent(ctx, req.OrderNo, amountCents, "cny", methodType, returnURL)
	if err != nil {
		return nil, fmt.Errorf("stripe create payment intent: %w", err)
	}

	payURL := nextActionURL(pi, methodType)
	if payURL == "" {
		return nil, fmt.Errorf("stripe %s: empty pay url, status=%s", methodType, pi.Status)
	}
	return &payment.CreatePaymentResponse{
		TradeNo:   pi.ID, // pi_xxx
		PayURL:    payURL,
		QRCodeURL: "",
	}, nil
}

func (p *Provider) VerifyNotification(ctx context.Context, body []byte, params map[string]string) (*payment.PaymentNotification, error) {
	sigHeader := ""
	if params != nil {
		sigHeader = params["Stripe-Signature"]
	}

	event, err := VerifyWebhook(body, sigHeader, p.cfg.WebhookSecret)
	if err != nil {
		return nil, err
	}

	if event.Type != "payment_intent.succeeded" {
		return nil, fmt.Errorf("%w: %s", payment.ErrEventIgnored, event.Type)
	}

	jsonData, err := json.Marshal(event.Data.Object)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}

	var pi gostripe.PaymentIntent
	if err := json.Unmarshal(jsonData, &pi); err != nil {
		return nil, fmt.Errorf("parse payment intent from event: %w", err)
	}

	n := paymentIntentToNotification(&pi)
	n.PaidAt = fmt.Sprintf("%d", event.Created)
	return n, nil
}

func (p *Provider) QueryOrder(ctx context.Context, tradeNo string) (*payment.PaymentNotification, error) {
	// tradeNo 为创建支付时返回的 PaymentIntent ID（pi_xxx）
	if ctx == nil {
		ctx = context.Background()
	}

	pi, err := p.client.RetrievePaymentIntent(ctx, tradeNo)
	if err != nil {
		return nil, fmt.Errorf("stripe retrieve payment intent: %w", err)
	}
	if pi.Status != gostripe.PaymentIntentStatusSucceeded {
		return nil, fmt.Errorf("stripe payment intent not succeeded: status=%s", pi.Status)
	}
	return paymentIntentToNotification(pi), nil
}
