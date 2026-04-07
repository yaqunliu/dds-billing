package model

import "time"

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusPaid       OrderStatus = "paid"
	OrderStatusRecharging OrderStatus = "recharging"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusFailed     OrderStatus = "failed"
	OrderStatusExpired    OrderStatus = "expired"
)

type Order struct {
	ID           int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNo      string      `gorm:"type:varchar(64);not null;uniqueIndex" json:"order_no"`
	UserID       int64       `gorm:"not null;index" json:"user_id"`
	UserEmail    string      `gorm:"type:varchar(255)" json:"user_email"`
	Amount       float64     `gorm:"type:decimal(10,2);not null" json:"amount"`
	Status       OrderStatus `gorm:"type:varchar(20);not null;default:pending;index" json:"status"`
	PaymentType  string      `gorm:"type:varchar(20);not null" json:"payment_type"`
	Provider     string      `gorm:"type:varchar(32);not null" json:"provider"`
	TradeNo      string      `gorm:"type:varchar(128)" json:"trade_no"`
	PayNo        string      `gorm:"type:varchar(128)" json:"pay_no"`
	QRCodeURL    string      `gorm:"type:text" json:"qr_code_url"`
	PayURL       string      `gorm:"type:text" json:"pay_url"`
	NotifyData   string      `gorm:"type:text" json:"notify_data,omitempty"`
	RechargeCode *string     `gorm:"type:varchar(64);uniqueIndex" json:"recharge_code,omitempty"`
	FailedReason string      `gorm:"type:varchar(512)" json:"failed_reason,omitempty"`
	ExpiresAt    time.Time   `gorm:"not null" json:"expires_at"`
	PaidAt       *time.Time  `json:"paid_at,omitempty"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
	CreatedAt    time.Time   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Order) TableName() string {
	return "orders"
}
