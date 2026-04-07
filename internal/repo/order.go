package repo

import (
	"dds-billing/internal/model"

	"gorm.io/gorm"
)

type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) Create(order *model.Order) error {
	return r.db.Create(order).Error
}

func (r *OrderRepo) GetByOrderNo(orderNo string) (*model.Order, error) {
	var order model.Order
	err := r.db.Where("order_no = ?", orderNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepo) Update(order *model.Order) error {
	return r.db.Save(order).Error
}

func (r *OrderRepo) ListByUserID(userID int64, offset, limit int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	query := r.db.Where("user_id = ?", userID)
	if err := query.Model(&model.Order{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *OrderRepo) UpdateStatus(orderNo string, status model.OrderStatus, updates map[string]interface{}) error {
	updates["status"] = status
	return r.db.Model(&model.Order{}).Where("order_no = ?", orderNo).Updates(updates).Error
}
