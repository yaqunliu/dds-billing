package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// fxRater 拉取并缓存实时汇率（USD 基准），用于把人民币金额换算成信用卡计价币种。
// 拉取失败或数据缺失时回退到 fallback（按 1 USD = fallback CNY 估算）。
type fxRater struct {
	apiURL   string        // USD 基准汇率接口，返回 {"result":"success","rates":{"CNY":7.1,...}}
	fallback float64       // 兜底：1 美元 = 多少人民币
	ttl      time.Duration // 缓存有效期
	client   *http.Client

	mu      sync.Mutex
	rates   map[string]float64 // 1 USD = rates[币种] 该币种金额
	expires time.Time
}

func newFxRater(apiURL string, fallback float64) *fxRater {
	if apiURL == "" {
		apiURL = "https://open.er-api.com/v6/latest/USD"
	}
	if fallback <= 0 {
		fallback = 7
	}
	return &fxRater{
		apiURL:   apiURL,
		fallback: fallback,
		ttl:      time.Hour,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

type erAPIResp struct {
	Result string             `json:"result"`
	Rates  map[string]float64 `json:"rates"`
}

// CNYPer 返回 1 个 currency 单位等于多少人民币。
// 例如 currency=usd 时返回「1 美元 = 多少人民币」。拉取失败时回退 fallback（按 USD 口径）。
func (f *fxRater) CNYPer(ctx context.Context, currency string) float64 {
	cur := strings.ToUpper(currency)
	if cur == "CNY" {
		return 1
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rates == nil || time.Now().After(f.expires) {
		if rates, err := f.fetch(ctx); err == nil && rates["CNY"] > 0 {
			f.rates = rates
			f.expires = time.Now().Add(f.ttl)
		}
		// 拉取失败时保留上一次的有效缓存（若有），否则走兜底
	}

	if f.rates != nil {
		cny := f.rates["CNY"]
		base := f.rates[cur]
		if cny > 0 && base > 0 {
			return cny / base
		}
	}
	return f.fallback
}

func (f *fxRater) fetch(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.apiURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fx api status %d", resp.StatusCode)
	}
	var out erAPIResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Rates) == 0 {
		return nil, fmt.Errorf("fx api empty rates")
	}
	return out.Rates, nil
}
