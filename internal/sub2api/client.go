package sub2api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	adminKey   string
	httpClient *http.Client
}

func NewClient(baseURL, adminKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		adminKey:   adminKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// UserInfo Sub2API 用户信息
type UserInfo struct {
	ID       int64   `json:"id"`
	Email    string  `json:"email"`
	Username string  `json:"username"`
	Balance  float64 `json:"balance"`
}

type apiResponse struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// isSuccess 判断 code 是否表示成功（数字 0 或字符串 "0"）
func (r *apiResponse) isSuccess() bool {
	s := string(r.Code)
	return s == "0" || s == `"0"`
}

// codeString 返回 code 的字符串表示
func (r *apiResponse) codeString() string {
	s := string(r.Code)
	// 去掉 JSON 字符串的引号
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// VerifyUser 用 token 调 Sub2API 验证用户身份
func (c *Client) VerifyUser(token string) (*UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/v1/auth/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sub2api verify user: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse sub2api response: %w, body: %s", err, string(body))
	}
	if !apiResp.isSuccess() {
		return nil, fmt.Errorf("sub2api auth failed: code=%s, msg=%s", apiResp.codeString(), apiResp.Message)
	}

	var user UserInfo
	if err := json.Unmarshal(apiResp.Data, &user); err != nil {
		return nil, fmt.Errorf("parse user info: %w", err)
	}
	return &user, nil
}

// RechargeRequest 充值请求
type RechargeRequest struct {
	Code   string  `json:"code"`
	Type   string  `json:"type"`
	Value  float64 `json:"value"`
	UserID int64   `json:"user_id"`
	Notes  string  `json:"notes"`
}

// Recharge 调用 Sub2API 创建充值码并兑换，幂等
func (c *Client) Recharge(orderNo string, userID int64, amount float64) error {
	reqBody := RechargeRequest{
		Code:   orderNo,
		Type:   "balance",
		Value:  amount,
		UserID: userID,
		Notes:  fmt.Sprintf("dds-billing recharge order:%s", orderNo),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/admin/redeem-codes/create-and-redeem", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.adminKey)
	req.Header.Set("Idempotency-Key", fmt.Sprintf("dds-billing:recharge:%s", orderNo))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sub2api recharge request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("parse recharge response: %w, body: %s", err, string(body))
	}
	if !apiResp.isSuccess() {
		return fmt.Errorf("sub2api recharge failed: code=%s, msg=%s", apiResp.codeString(), apiResp.Message)
	}

	return nil
}
