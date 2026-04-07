package ltzf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL         = "https://api.ltzf.cn"
	wxpayNativePath = "/api/wxpay/native"
	alipayNativePath = "/api/alipay/native"
	queryOrderPath  = "/api/wxpay/get_pay_order"
	refundPath      = "/api/wxpay/refund_order"
)

// Client 蓝兔支付 HTTP 客户端
type Client struct {
	mchID     string
	secretKey string
	httpClient *http.Client
}

func NewClient(mchID, secretKey string) *Client {
	return &Client{
		mchID:     mchID,
		secretKey: secretKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// APIResponse 蓝兔支付通用响应
type APIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// PayData 创建支付返回数据
type PayData struct {
	CodeURL   string `json:"code_url"`
	QRCodeURL string `json:"QRcode_url"`
	OrderNo   string `json:"order_no"`
}

// CreateNativePayment 创建扫码支付（微信或支付宝）
func (c *Client) CreateNativePayment(payType, outTradeNo, totalFee, body, notifyURL string) (*PayData, error) {
	var apiPath string
	switch payType {
	case "wxpay":
		apiPath = wxpayNativePath
	case "alipay":
		apiPath = alipayNativePath
	default:
		return nil, fmt.Errorf("unsupported pay type: %s", payType)
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	params := map[string]string{
		"mch_id":       c.mchID,
		"out_trade_no": outTradeNo,
		"total_fee":    totalFee,
		"body":         body,
		"timestamp":    timestamp,
		"notify_url":   notifyURL,
	}
	params["sign"] = Sign(params, c.secretKey)

	resp, err := c.postForm(baseURL+apiPath, params)
	if err != nil {
		return nil, fmt.Errorf("create payment request: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("create payment failed: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	var payData PayData
	if err := json.Unmarshal(resp.Data, &payData); err != nil {
		return nil, fmt.Errorf("parse payment response: %w", err)
	}
	return &payData, nil
}

// QueryOrder 查询订单状态
func (c *Client) QueryOrder(outTradeNo string) (*APIResponse, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	params := map[string]string{
		"mch_id":       c.mchID,
		"out_trade_no": outTradeNo,
		"timestamp":    timestamp,
	}
	params["sign"] = Sign(params, c.secretKey)

	return c.postForm(baseURL+queryOrderPath, params)
}

func (c *Client) postForm(apiURL string, params map[string]string) (*APIResponse, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(respBody))
	}
	return &apiResp, nil
}
