package easypay

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Sign 易支付协议签名算法
// 1. 排除 sign、sign_type 和空值参数
// 2. 按参数名 ASCII 字典序升序排序
// 3. 拼接为 URL key=value 格式
// 4. 末尾追加商户密钥
// 5. MD5 取小写
func Sign(params map[string]string, key string) string {
	var keys []string
	for k, v := range params {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	raw := strings.Join(parts, "&") + key

	hash := md5.Sum([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

// VerifySign 验证签名
func VerifySign(params map[string]string, key string) bool {
	sign, ok := params["sign"]
	if !ok || sign == "" {
		return false
	}
	expected := Sign(params, key)
	return sign == expected
}

// ParseQueryToMap 将 URL query string 解析为 map
func ParseQueryToMap(raw string) (map[string]string, error) {
	values, err := url.ParseQuery(raw)
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
