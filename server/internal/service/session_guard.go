// 会话归属断言（P0-1）。顾客端 /sessions/:sid/* 原先只认 session_id 存在、不认归属，
// 任意顾客 JWT 可枚举 sid 读他人整桌订单/账单、改人数、抢结账锁、取消他人结账、冒名呼叫。
// 顾客 JWT 的 SID 是 shop_id（16.2）而非会话 id，故「消费 SID 不能修复」——正确修法是成员断言。
package service

import (
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pkg/respond"
)

// assertSessionMember 会话归属断言：该顾客必须是会话成员。
// 「非成员」与会话不存在返回【同一个】ErrSessionInvalid，避免
// 「403=存在但不属于我 / 20008=不存在」构成 sid 枚举预言机。
func assertSessionMember(db *gorm.DB, shopID, sessionID, customerID int64) *respond.BizErr {
	var cnt int64
	if err := db.Model(&model.TableSessionMember{}).
		Where("shop_id = ? AND session_id = ? AND customer_id = ?", shopID, sessionID, customerID).
		Count(&cnt).Error; err != nil {
		slog.Error("assert session member failed", "err", err, "session_id", sessionID, "customer_id", customerID)
		return respond.ErrInternal
	}
	if cnt == 0 {
		return respond.ErrSessionInvalid
	}
	return nil
}

// addSessionMember 登记会话参与人（幂等，重复加入静默忽略）。
func addSessionMember(tx *gorm.DB, shopID, sessionID, customerID int64) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}, {Name: "customer_id"}},
		DoNothing: true,
	}).Create(&model.TableSessionMember{ShopID: shopID, SessionID: sessionID, CustomerID: customerID}).Error
}
