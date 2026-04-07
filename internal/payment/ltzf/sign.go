package ltzf

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Sign 按蓝兔支付签名算法生成签名
// 1. 筛选：只取非空参数（不含 sign 本身）
// 2. 排序：按参数名 ASCII 字典序升序
// 3. 拼接：key1=value1&key2=value2&...&key=商户密钥
// 4. 加密：MD5(拼接字符串).toUpperCase()
func Sign(params map[string]string, secretKey string) string {
	// Filter and collect keys
	var keys []string
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build query string
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	parts = append(parts, "key="+secretKey)
	raw := strings.Join(parts, "&")

	// MD5 uppercase
	hash := md5.Sum([]byte(raw))
	return strings.ToUpper(fmt.Sprintf("%x", hash))
}

// VerifySign 验证签名
func VerifySign(params map[string]string, secretKey string) bool {
	sign, ok := params["sign"]
	if !ok || sign == "" {
		return false
	}
	expected := Sign(params, secretKey)
	return sign == expected
}

// ParseFormToMap 将 URL-encoded form body 解析为 map
func ParseFormToMap(body []byte) (map[string]string, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result, nil
}
