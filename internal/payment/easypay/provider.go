package easypay

import (
	"context"
	"fmt"

	"dds-billing/internal/config"
	"dds-billing/internal/payment"
)

type Provider struct {
	client *Client
	cfg    config.EasypayConfig
}

func NewProvider(cfg config.EasypayConfig) *Provider {
	return &Provider{
		client: NewClient(cfg.PID, cfg.PKey, cfg.APIBase),
		cfg:    cfg,
	}
}

func (p *Provider) Name() string {
	return "easypay"
}

func (p *Provider) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.PaymentTypeWxpay, payment.PaymentTypeAlipay}
}

func (p *Provider) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = p.cfg.NotifyURL
	}

	returnURL := p.cfg.ReturnURL
	if returnURL == "" {
		returnURL = notifyURL
	}

	result, err := p.client.CreateOrder(
		req.OrderNo,
		string(req.PaymentType),
		req.Subject,
		req.Amount,
		notifyURL,
		returnURL,
		"127.0.0.1", // clientIP 由后端传，回调时不校验
	)
	if err != nil {
		return nil, fmt.Errorf("easypay create payment: %w", err)
	}

	return &payment.CreatePaymentResponse{
		TradeNo:   result.TradeNo,
		PayURL:    result.PayURL,
		QRCodeURL: result.QRCode,
	}, nil
}

func (p *Provider) VerifyNotification(ctx context.Context, body []byte, params map[string]string) (*payment.PaymentNotification, error) {
	// 易支付回调是 GET 请求，参数通过 query string 传递
	// notify handler 会把 body 传过来，但易支付回调参数在 params map 中
	// 如果 params 为空，尝试从 body 解析（兼容 POST form）
	notifyParams := params
	if len(notifyParams) == 0 {
		var err error
		notifyParams, err = ParseQueryToMap(string(body))
		if err != nil {
			return nil, fmt.Errorf("parse notify params: %w", err)
		}
	}

	// 验签
	if !VerifySign(notifyParams, p.cfg.PKey) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// 检查交易状态
	if notifyParams["trade_status"] != "TRADE_SUCCESS" {
		return nil, fmt.Errorf("trade not successful: status=%s", notifyParams["trade_status"])
	}

	// 映射支付方式
	var payType payment.PaymentType
	switch notifyParams["type"] {
	case "wxpay":
		payType = payment.PaymentTypeWxpay
	case "alipay":
		payType = payment.PaymentTypeAlipay
	default:
		payType = payment.PaymentType(notifyParams["type"])
	}

	return &payment.PaymentNotification{
		OrderNo:     notifyParams["out_trade_no"],
		TradeNo:     notifyParams["trade_no"],
		PayNo:       notifyParams["trade_no"],
		Amount:      notifyParams["money"],
		PaymentType: payType,
	}, nil
}

func (p *Provider) QueryOrder(ctx context.Context, orderNo string) (*payment.PaymentNotification, error) {
	result, err := p.client.QueryOrder(orderNo)
	if err != nil {
		return nil, fmt.Errorf("easypay query order: %w", err)
	}

	var payType payment.PaymentType
	switch result.Type {
	case "wxpay":
		payType = payment.PaymentTypeWxpay
	case "alipay":
		payType = payment.PaymentTypeAlipay
	default:
		payType = payment.PaymentType(result.Type)
	}

	return &payment.PaymentNotification{
		OrderNo:     result.OutTradeNo,
		TradeNo:     result.TradeNo,
		PayNo:       result.TradeNo,
		Amount:      result.Money,
		PaymentType: payType,
		PaidAt:      result.Endtime,
	}, nil
}
