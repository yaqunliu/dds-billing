package easypay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client 易支付协议 HTTP 客户端
type Client struct {
	pid     string
	key     string
	apiBase string // 例如 https://api.payqixiang.cn
	http    *http.Client
}

func NewClient(pid, key, apiBase string) *Client {
	// 去掉末尾斜杠
	apiBase = strings.TrimRight(apiBase, "/")
	return &Client{
		pid:     pid,
		key:     key,
		apiBase: apiBase,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateOrderResponse mapi.php 响应
type CreateOrderResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	TradeNo string `json:"trade_no"`
	PayURL  string `json:"payurl"`
	QRCode  string `json:"qrcode"`
}

// QueryOrderResponse 查询订单响应
type QueryOrderResponse struct {
	Code        int    `json:"code"`
	Msg         string `json:"msg"`
	TradeNo     string `json:"trade_no"`
	OutTradeNo  string `json:"out_trade_no"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Money       string `json:"money"`
	Status      int    `json:"status"` // 1为支付成功，0为未支付
	Addtime     string `json:"addtime"`
	Endtime     string `json:"endtime"`
}

// CreateOrder 统一下单（mapi.php）
func (c *Client) CreateOrder(outTradeNo, payType, name, money, notifyURL, returnURL, clientIP string) (*CreateOrderResponse, error) {
	params := map[string]string{
		"pid":          c.pid,
		"type":         payType,
		"out_trade_no": outTradeNo,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         name,
		"money":        money,
		"clientip":     clientIP,
		"device":       "pc",
	}
	params["sign"] = Sign(params, c.key)
	params["sign_type"] = "MD5"

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	req, err := http.NewRequest(http.MethodPost, c.apiBase+"/mapi.php", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("easypay request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result CreateOrderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse easypay response: %w, body: %s", err, string(body))
	}
	if result.Code != 1 {
		return nil, fmt.Errorf("easypay create order failed: code=%d, msg=%s", result.Code, result.Msg)
	}

	return &result, nil
}

// QueryOrder 查询单笔订单
func (c *Client) QueryOrder(outTradeNo string) (*QueryOrderResponse, error) {
	u := fmt.Sprintf("%s/api.php?act=order&pid=%s&key=%s&out_trade_no=%s",
		c.apiBase, c.pid, c.key, url.QueryEscape(outTradeNo))

	resp, err := c.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("easypay query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result QueryOrderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse query response: %w, body: %s", err, string(body))
	}
	if result.Code != 1 {
		return nil, fmt.Errorf("easypay query failed: code=%d, msg=%s", result.Code, result.Msg)
	}

	return &result, nil
}
