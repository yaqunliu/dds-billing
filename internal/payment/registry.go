package payment

import "dds-billing/internal/config"

var providers = map[string]PaymentProvider{}

func Register(name string, p PaymentProvider) {
	providers[name] = p
}

func Get(name string) (PaymentProvider, bool) {
	p, ok := providers[name]
	return p, ok
}

// GetActive 根据配置返回当前激活的渠道
func GetActive(cfg *config.Config) PaymentProvider {
	p, ok := Get(cfg.Payment.Provider)
	if !ok {
		panic("unknown payment provider: " + cfg.Payment.Provider)
	}
	return p
}
