package payment

import (
	"context"
	"errors"
)

// ErrEventIgnored 表示该回调事件不需要处理（非错误，应返回 200）
var ErrEventIgnored = errors.New("event ignored")

type PaymentType string

const (
	PaymentTypeWxpay  PaymentType = "wxpay"
	PaymentTypeAlipay PaymentType = "alipay"
)

// CreatePaymentRequest 创建支付请求
type CreatePaymentRequest struct {
	OrderNo     string      // 商户订单号
	Amount      string      // 支付金额（元）
	Subject     string      // 商品描述
	NotifyURL   string      // 回调通知地址
	PaymentType PaymentType // wxpay / alipay
}

// CreatePaymentResponse 创建支付响应
type CreatePaymentResponse struct {
	TradeNo   string // 渠道系统订单号
	PayURL    string // 支付链接
	QRCodeURL string // 二维码图片地址
}

// PaymentNotification 回调通知数据
type PaymentNotification struct {
	OrderNo     string // 商户订单号
	TradeNo     string // 渠道系统订单号
	PayNo       string // 支付宝/微信订单号
	Amount      string // 支付金额
	PaymentType PaymentType
	PaidAt      string // 支付时间
}

// PaymentProvider 支付渠道统一接口
type PaymentProvider interface {
	// Name 渠道名称
	Name() string

	// SupportedTypes 支持的支付方式
	SupportedTypes() []PaymentType

	// CreatePayment 创建支付订单，返回二维码/支付链接
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error)

	// VerifyNotification 验证回调签名，解析通知数据
	VerifyNotification(ctx context.Context, body []byte, params map[string]string) (*PaymentNotification, error)

	// QueryOrder 主动查询订单状态
	QueryOrder(ctx context.Context, orderNo string) (*PaymentNotification, error)
}
