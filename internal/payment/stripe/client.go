package stripe

import (
	"context"
	"fmt"

	"dds-billing/internal/config"

	gostripe "github.com/stripe/stripe-go/v82"
)

// Client Stripe API 客户端封装
type Client struct {
	cfg    config.StripeConfig
	stripe *gostripe.Client
}

func NewClient(cfg config.StripeConfig) *Client {
	sc := gostripe.NewClient(cfg.SecretKey)
	return &Client{cfg: cfg, stripe: sc}
}

// CreatePaymentIntent 创建并立即 confirm 一个 PaymentIntent（微信/支付宝统一走此路径）。
// 返回后由调用方从 NextAction 中读取对应的支付链接：
//   - wechat_pay: NextAction.WeChatPayDisplayQRCode.Data（weixin:// 链接，前端生成二维码）
//   - alipay:    NextAction.AlipayHandleRedirect.URL（Stripe 托管跳转链接，避免 Checkout 强制收集邮箱）
//
// returnURL 仅 alipay 需要：支付完成后浏览器跳回的地址。
func (c *Client) CreatePaymentIntent(ctx context.Context, orderNo string, amountCents int64, currency, methodType, returnURL string) (*gostripe.PaymentIntent, error) {
	params := &gostripe.PaymentIntentCreateParams{
		Amount:             gostripe.Int64(amountCents),
		Currency:           gostripe.String(currency),
		PaymentMethodTypes: []*string{gostripe.String(methodType)},
		PaymentMethodData: &gostripe.PaymentIntentCreatePaymentMethodDataParams{
			Type: gostripe.String(methodType),
		},
		Confirm: gostripe.Bool(true),
		Metadata: map[string]string{
			"order_no": orderNo,
			"project":  "dds-billing",
		},
	}

	switch methodType {
	case "wechat_pay":
		params.PaymentMethodData.WeChatPay = &gostripe.PaymentMethodWeChatPayParams{}
		params.PaymentMethodOptions = &gostripe.PaymentIntentCreatePaymentMethodOptionsParams{
			WeChatPay: &gostripe.PaymentIntentCreatePaymentMethodOptionsWeChatPayParams{
				Client: gostripe.String("web"),
			},
		}
	case "alipay":
		params.PaymentMethodData.Alipay = &gostripe.PaymentMethodAlipayParams{}
		if returnURL != "" {
			params.ReturnURL = gostripe.String(returnURL)
		}
	}

	return c.stripe.V1PaymentIntents.Create(ctx, params)
}

// RetrievePaymentIntent 查询 PaymentIntent 状态
func (c *Client) RetrievePaymentIntent(ctx context.Context, intentID string) (*gostripe.PaymentIntent, error) {
	return c.stripe.V1PaymentIntents.Retrieve(ctx, intentID, nil)
}

// VerifyWebhook 验证 webhook 签名并解析事件
func VerifyWebhook(payload []byte, sigHeader string, webhookSecret string) (*gostripe.Event, error) {
	event, err := gostripe.ConstructEvent(payload, sigHeader, webhookSecret, gostripe.WithIgnoreAPIVersionMismatch())
	if err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}
	return &event, nil
}
