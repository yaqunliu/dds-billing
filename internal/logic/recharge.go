package logic

import (
	"fmt"
	"log"
	"time"

	"dds-billing/internal/model"
	"dds-billing/internal/repo"
	"dds-billing/internal/sub2api"
)

type RechargeLogic struct {
	orderRepo *repo.OrderRepo
	sub2api   *sub2api.Client
}

func NewRechargeLogic(orderRepo *repo.OrderRepo, sub2apiClient *sub2api.Client) *RechargeLogic {
	return &RechargeLogic{
		orderRepo: orderRepo,
		sub2api:   sub2apiClient,
	}
}

// ProcessRecharge 支付成功后调用 Sub2API 给用户充值
// 状态流转：paid → recharging → completed / failed
func (l *RechargeLogic) ProcessRecharge(orderNo string) error {
	order, err := l.orderRepo.GetByOrderNo(orderNo)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	// Only process paid orders
	if order.Status != model.OrderStatusPaid {
		log.Printf("[recharge] order %s status is %s, skip", orderNo, order.Status)
		return nil
	}

	// Mark as recharging
	if err := l.orderRepo.UpdateStatus(orderNo, model.OrderStatusRecharging, map[string]interface{}{}); err != nil {
		return fmt.Errorf("update status to recharging: %w", err)
	}

	// Call Sub2API to recharge
	err = l.sub2api.Recharge(orderNo, order.UserID, order.Amount)
	if err != nil {
		// Recharge failed
		log.Printf("[recharge] order %s recharge failed: %v", orderNo, err)
		l.orderRepo.UpdateStatus(orderNo, model.OrderStatusFailed, map[string]interface{}{
			"failed_reason": err.Error(),
		})
		return fmt.Errorf("recharge: %w", err)
	}

	// Recharge success
	now := time.Now()
	if err := l.orderRepo.UpdateStatus(orderNo, model.OrderStatusCompleted, map[string]interface{}{
		"recharge_code": orderNo,
		"completed_at":  now,
	}); err != nil {
		return fmt.Errorf("update status to completed: %w", err)
	}

	log.Printf("[recharge] order %s completed, user=%d, amount=%.2f", orderNo, order.UserID, order.Amount)
	return nil
}
