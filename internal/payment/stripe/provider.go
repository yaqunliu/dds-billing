package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

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

	if ctx == nil {
		ctx = context.Background()
	}

	switch req.PaymentType {
	case payment.PaymentTypeWxpay:
		// 微信支付走 PaymentIntent 直连方式，next_action 里返回 weixin:// 链接
		// 前端用它生成二维码，微信扫码可直接拉起支付（避免 Checkout 页二次扫码）
		pi, err := p.client.CreateWeChatPayIntent(ctx, req.OrderNo, amountCents, "cny")
		if err != nil {
			return nil, fmt.Errorf("stripe create wechat payment intent: %w", err)
		}
		payURL := ""
		if pi.NextAction != nil && pi.NextAction.WeChatPayDisplayQRCode != nil {
			payURL = pi.NextAction.WeChatPayDisplayQRCode.Data
		}
		if payURL == "" {
			return nil, fmt.Errorf("stripe wechat pay: empty qr code data, status=%s", pi.Status)
		}
		return &payment.CreatePaymentResponse{
			TradeNo:   pi.ID, // pi_xxx
			PayURL:    payURL,
			QRCodeURL: "",
		}, nil

	case payment.PaymentTypeAlipay:
		// 支付宝保持 Checkout Session 流程不变
		successURL := p.cfg.SuccessURL + "?order_no=" + req.OrderNo + "&status=success"
		cancelURL := p.cfg.CancelURL + "?order_no=" + req.OrderNo + "&status=cancel"
		session, err := p.client.CreateCheckoutSession(ctx, req.OrderNo, amountCents, "cny", "alipay", successURL, cancelURL)
		if err != nil {
			return nil, fmt.Errorf("stripe create checkout session: %w", err)
		}
		return &payment.CreatePaymentResponse{
			TradeNo:   session.ID, // cs_xxx
			PayURL:    session.URL,
			QRCodeURL: "",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported payment type: %s", req.PaymentType)
	}
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

	jsonData, err := json.Marshal(event.Data.Object)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}

	switch event.Type {
	case "checkout.session.completed":
		// 支付宝 Checkout Session 完成
		var session gostripe.CheckoutSession
		if err := json.Unmarshal(jsonData, &session); err != nil {
			return nil, fmt.Errorf("parse checkout session from event: %w", err)
		}

		orderNo := session.ClientReferenceID
		if orderNo == "" {
			if v, ok := session.Metadata["order_no"]; ok {
				orderNo = v
			}
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

		payNo := ""
		if session.PaymentIntent != nil {
			payNo = session.PaymentIntent.ID
		}

		return &payment.PaymentNotification{
			OrderNo:     orderNo,
			TradeNo:     session.ID,
			PayNo:       payNo,
			Amount:      amountYuan,
			PaymentType: payType,
			PaidAt:      fmt.Sprintf("%d", event.Created),
		}, nil

	case "payment_intent.succeeded":
		// 微信支付 PaymentIntent 直连流程
		var pi gostripe.PaymentIntent
		if err := json.Unmarshal(jsonData, &pi); err != nil {
			return nil, fmt.Errorf("parse payment intent from event: %w", err)
		}

		orderNo := ""
		if v, ok := pi.Metadata["order_no"]; ok {
			orderNo = v
		}

		amountYuan := fmt.Sprintf("%.2f", float64(pi.Amount)/100)

		var payType payment.PaymentType
		if len(pi.PaymentMethodTypes) > 0 {
			switch pi.PaymentMethodTypes[0] {
			case "wechat_pay":
				payType = payment.PaymentTypeWxpay
			case "alipay":
				payType = payment.PaymentTypeAlipay
			}
		}

		return &payment.PaymentNotification{
			OrderNo:     orderNo,
			TradeNo:     pi.ID,
			PayNo:       pi.ID,
			Amount:      amountYuan,
			PaymentType: payType,
			PaidAt:      fmt.Sprintf("%d", event.Created),
		}, nil

	default:
		return nil, fmt.Errorf("%w: %s", payment.ErrEventIgnored, event.Type)
	}
}

func (p *Provider) QueryOrder(ctx context.Context, tradeNo string) (*payment.PaymentNotification, error) {
	// tradeNo 是创建支付时返回的渠道单号：
	//   - cs_xxx → Checkout Session（支付宝）
	//   - pi_xxx → PaymentIntent（微信支付）
	if ctx == nil {
		ctx = context.Background()
	}

	if strings.HasPrefix(tradeNo, "pi_") {
		pi, err := p.client.RetrievePaymentIntent(ctx, tradeNo)
		if err != nil {
			return nil, fmt.Errorf("stripe retrieve payment intent: %w", err)
		}
		if pi.Status != gostripe.PaymentIntentStatusSucceeded {
			return nil, fmt.Errorf("stripe payment intent not succeeded: status=%s", pi.Status)
		}

		orderNo := ""
		if v, ok := pi.Metadata["order_no"]; ok {
			orderNo = v
		}
		amountYuan := fmt.Sprintf("%.2f", float64(pi.Amount)/100)
		var payType payment.PaymentType
		if len(pi.PaymentMethodTypes) > 0 {
			switch pi.PaymentMethodTypes[0] {
			case "wechat_pay":
				payType = payment.PaymentTypeWxpay
			case "alipay":
				payType = payment.PaymentTypeAlipay
			}
		}
		return &payment.PaymentNotification{
			OrderNo:     orderNo,
			TradeNo:     pi.ID,
			PayNo:       pi.ID,
			Amount:      amountYuan,
			PaymentType: payType,
		}, nil
	}

	session, err := p.client.RetrieveSession(ctx, tradeNo)
	if err != nil {
		return nil, fmt.Errorf("stripe retrieve session: %w", err)
	}

	// 只有 payment_status=paid 才算支付成功；其他状态（unpaid / no_payment_required）让调用方按未支付处理
	if session.PaymentStatus != gostripe.CheckoutSessionPaymentStatusPaid {
		return nil, fmt.Errorf("stripe session not paid: status=%s", session.PaymentStatus)
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

	payNo := ""
	if session.PaymentIntent != nil {
		payNo = session.PaymentIntent.ID
	}

	return &payment.PaymentNotification{
		OrderNo:     session.ClientReferenceID,
		TradeNo:     session.ID,
		PayNo:       payNo,
		Amount:      amountYuan,
		PaymentType: payType,
	}, nil
}
