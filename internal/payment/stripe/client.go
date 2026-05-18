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

// CreateCheckoutSession 创建 Checkout Session
// amountCents 为最小货币单位（分）
func (c *Client) CreateCheckoutSession(ctx context.Context, orderNo string, amountCents int64, currency string, paymentMethodType string, successURL, cancelURL string) (*gostripe.CheckoutSession, error) {
	params := &gostripe.CheckoutSessionCreateParams{
		Mode: gostripe.String(string(gostripe.CheckoutSessionModePayment)),
		LineItems: []*gostripe.CheckoutSessionCreateLineItemParams{
			{
				PriceData: &gostripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency: gostripe.String(currency),
					ProductData: &gostripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name: gostripe.String("账户充值"),
					},
					UnitAmount: gostripe.Int64(amountCents),
				},
				Quantity: gostripe.Int64(1),
			},
		},
		PaymentMethodTypes: []*string{gostripe.String(paymentMethodType)},
		SuccessURL:         gostripe.String(successURL),
		CancelURL:          gostripe.String(cancelURL),
		ClientReferenceID:  gostripe.String(orderNo),
		Metadata: map[string]string{
			"order_no": orderNo,
			"project":  "dds-billing",
		},
	}

	// wechat_pay 需要指定 client 类型
	if paymentMethodType == "wechat_pay" {
		params.PaymentMethodOptions = &gostripe.CheckoutSessionCreatePaymentMethodOptionsParams{
			WeChatPay: &gostripe.CheckoutSessionCreatePaymentMethodOptionsWeChatPayParams{
				Client: gostripe.String("web"),
			},
		}
	}

	return c.stripe.V1CheckoutSessions.Create(ctx, params)
}

// RetrieveSession 查询 Session 状态
func (c *Client) RetrieveSession(ctx context.Context, sessionID string) (*gostripe.CheckoutSession, error) {
	return c.stripe.V1CheckoutSessions.Retrieve(ctx, sessionID, nil)
}

// CreateWeChatPayIntent 创建并立即 confirm 一个微信支付的 PaymentIntent
// 返回的 PaymentIntent.NextAction.WeChatPayDisplayQRCode.Data 是 weixin://wxpay/bizpayurl 链接，
// 前端用它生成二维码后，微信扫码可直接拉起支付。
func (c *Client) CreateWeChatPayIntent(ctx context.Context, orderNo string, amountCents int64, currency string) (*gostripe.PaymentIntent, error) {
	params := &gostripe.PaymentIntentCreateParams{
		Amount:             gostripe.Int64(amountCents),
		Currency:           gostripe.String(currency),
		PaymentMethodTypes: []*string{gostripe.String("wechat_pay")},
		PaymentMethodData: &gostripe.PaymentIntentCreatePaymentMethodDataParams{
			Type:      gostripe.String("wechat_pay"),
			WeChatPay: &gostripe.PaymentMethodWeChatPayParams{},
		},
		PaymentMethodOptions: &gostripe.PaymentIntentCreatePaymentMethodOptionsParams{
			WeChatPay: &gostripe.PaymentIntentCreatePaymentMethodOptionsWeChatPayParams{
				Client: gostripe.String("web"),
			},
		},
		Confirm: gostripe.Bool(true),
		Metadata: map[string]string{
			"order_no": orderNo,
			"project":  "dds-billing",
		},
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
