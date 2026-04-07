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

func (p *Provider) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	// 将金额从元转为分（Stripe 使用最小货币单位）
	amountFloat, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	amountCents := int64(math.Round(amountFloat * 100))

	// 映射支付方式到 Stripe payment_method_type
	var stripeMethodType string
	switch req.PaymentType {
	case payment.PaymentTypeWxpay:
		stripeMethodType = "wechat_pay"
	case payment.PaymentTypeAlipay:
		stripeMethodType = "alipay"
	default:
		return nil, fmt.Errorf("unsupported payment type: %s", req.PaymentType)
	}

	// Stripe 用 success_url/cancel_url 重定向用户，webhook 异步通知后端
	successURL := p.cfg.SuccessURL + "?order_no=" + req.OrderNo + "&status=success"
	cancelURL := p.cfg.CancelURL + "?order_no=" + req.OrderNo + "&status=cancel"

	if ctx == nil {
		ctx = context.Background()
	}

	session, err := p.client.CreateCheckoutSession(ctx, req.OrderNo, amountCents, "cny", stripeMethodType, successURL, cancelURL)
	if err != nil {
		return nil, fmt.Errorf("stripe create checkout session: %w", err)
	}

	return &payment.CreatePaymentResponse{
		TradeNo:   session.ID,
		PayURL:    session.URL,
		QRCodeURL: "", // Stripe 不直接返回二维码图片，前端从 PayURL 生成
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

	// 只处理 checkout.session.completed 事件，其他事件忽略
	if event.Type != "checkout.session.completed" {
		return nil, fmt.Errorf("%w: %s", payment.ErrEventIgnored, event.Type)
	}

	// 从 event.Data.Object 解析 session 数据
	var session gostripe.CheckoutSession
	jsonData, err := json.Marshal(event.Data.Object)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}
	if err := json.Unmarshal(jsonData, &session); err != nil {
		return nil, fmt.Errorf("parse checkout session from event: %w", err)
	}

	orderNo := session.ClientReferenceID
	if orderNo == "" {
		if v, ok := session.Metadata["order_no"]; ok {
			orderNo = v
		}
	}

	// 金额从分转回元
	amountYuan := fmt.Sprintf("%.2f", float64(session.AmountTotal)/100)

	var payType payment.PaymentType
	if len(session.PaymentMethodTypes) > 0 {
		switch session.PaymentMethodTypes[0] {
		case "wechat_pay":
			payType = payment.PaymentTypeWxpay
		case "alipay":
			payType = payment.PaymentTypeAlipay
		}
	}

	return &payment.PaymentNotification{
		OrderNo:     orderNo,
		TradeNo:     session.ID,
		PayNo:       string(session.PaymentIntent.ID),
		Amount:      amountYuan,
		PaymentType: payType,
		PaidAt:      fmt.Sprintf("%d", event.Created),
	}, nil
}

func (p *Provider) QueryOrder(ctx context.Context, orderNo string) (*payment.PaymentNotification, error) {
	// orderNo 在 Stripe 中对应的是 session ID（存在 trade_no 字段中）
	// 这里 orderNo 传入的实际上是 trade_no (session ID)
	if ctx == nil {
		ctx = context.Background()
	}

	session, err := p.client.RetrieveSession(ctx, orderNo)
	if err != nil {
		return nil, fmt.Errorf("stripe retrieve session: %w", err)
	}

	amountYuan := fmt.Sprintf("%.2f", float64(session.AmountTotal)/100)

	var payType payment.PaymentType
	if len(session.PaymentMethodTypes) > 0 {
		switch session.PaymentMethodTypes[0] {
		case "wechat_pay":
			payType = payment.PaymentTypeWxpay
		case "alipay":
			payType = payment.PaymentTypeAlipay
		}
	}

	return &payment.PaymentNotification{
		OrderNo:     session.ClientReferenceID,
		TradeNo:     session.ID,
		PayNo:       string(session.PaymentIntent.ID),
		Amount:      amountYuan,
		PaymentType: payType,
	}, nil
}
