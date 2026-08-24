// 会话与三态结账（设计 3.4/3.7/3.8，阶段 2.3）。
// 后付/前台结：订单提交即 unsettled → 结账生成 bill → 支付批量置位 paid → 清台 done。
// 防并发：locked_for_bill 条件更新（原子）+ uniq_session_active_bill 部分唯一索引双重防线。
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pay"
	"github.com/yodian/server/internal/pkg/respond"
	"github.com/yodian/server/internal/ws"
)

type BillService struct {
	db     *gorm.DB
	rdb    *redis.Client
	shopID int64
	hub    *ws.Hub // 2.4：结账入账后推后厨
}

func NewBillService(db *gorm.DB, rdb *redis.Client, shopID int64, hub *ws.Hub) *BillService {
	return &BillService{db: db, rdb: rdb, shopID: shopID, hub: hub}
}

// push 事件广播（hub 未挂接时静默跳过）
func (s *BillService) push(event string, payload any) {
	if s.hub != nil {
		s.hub.Broadcast(event, payload)
	}
}

// CreateBill 生成结账单（服务端聚合未结账子单 + postpay/frontend 首单按人项一次性计入）。
// 并发防御：先原子锁定会话（locked_for_bill false→true），未抢到锁视为「结账中」。
func (s *BillService) CreateBill(ctx context.Context, sessionID int64) (*model.Bill, *respond.BizErr) {
	var sess model.TableSession
	if err := s.db.First(&sess, "shop_id = ? AND id = ? AND status = 'active'", s.shopID, sessionID).Error; err != nil {
		return nil, respond.ErrSessionInvalid
	}
	if sess.LockedForBill {
		return nil, respond.ErrSessionLocked
	}

	var orders []model.Order
	if err := s.db.Where("session_id = ? AND status = 'unsettled'", sess.ID).Find(&orders).Error; err != nil {
		slog.Error("query unsettled orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	if len(orders) == 0 {
		return nil, respond.NewBiz(20009, "暂无未结账订单", 200)
	}
	var total model.Money
	for _, o := range orders {
		total += o.TotalAmount
	}

	// postpay/frontend 首次结账：按人项一次性计入（3.8），折叠进账单总额
	if (sess.PayMode == "postpay" || sess.PayMode == "frontend") && !s.hasPriorBill(ctx, sess.ID) {
		charges, bizErr := s.loadPerHead(ctx)
		if bizErr != nil {
			return nil, bizErr
		}
		for _, c := range charges {
			total += c.Price * model.Money(sess.Pax)
		}
	}

	outTradeNo := genOutTradeNo()
	bill := model.Bill{
		ShopID: s.shopID, SessionID: sess.ID,
		TotalAmount: total, PayableAmount: total,
		Status: "pending", OutTradeNo: &outTradeNo,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 原子锁会话：并发结账只有一个能成功（防双账单/结账期间加菜）
		res := tx.Model(&model.TableSession{}).
			Where("id = ? AND locked_for_bill = false AND status = 'active'", sess.ID).
			Update("locked_for_bill", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return respond.ErrSessionLocked
		}
		// 2. 归档已支付旧账单（paid→closed），释放 uniq_session_active_bill，允许追加新账单
		if err := tx.Model(&model.Bill{}).Where("session_id = ? AND status = 'paid'", sess.ID).
			Update("status", "closed").Error; err != nil {
			return err
		}
		// 3. 建单；uniq_session_active_bill 兜底并发
		return tx.Create(&bill).Error
	})
	if err != nil {
		var be *respond.BizErr
		if errors.As(err, &be) {
			return nil, be
		}
		if isUniqueViolation(err) {
			return nil, respond.NewBiz(20010, "该会话已有结账单，请勿重复结账", 200)
		}
		slog.Error("create bill tx failed", "err", err)
		return nil, respond.ErrInternal
	}

	slog.Info("bill created", "bill_id", bill.ID, "session_id", sess.ID,
		"payable_cents", int64(bill.PayableAmount), "unsettled_orders", len(orders))
	return &bill, nil
}

// HandleBillPaidLocked 账单支付成功（由 OrderService.HandlePaidSuccess 在持有 pay:lock 时委托）：
// 幂等 + 金额校验 + 批量置位。约定：锁由顶层支付管线持有，本方法不再加锁。
func (s *BillService) HandleBillPaidLocked(ctx context.Context, res pay.NotifyResult) error {
	var bill model.Bill
	if err := s.db.First(&bill, "out_trade_no = ?", res.OutTradeNo).Error; err != nil {
		return fmt.Errorf("query bill: %w", err)
	}
	if bill.Status == "paid" || bill.Status == "closed" {
		slog.Info("idempotent: bill already settled", "bill_id", bill.ID)
		return nil
	}
	if res.PaidAmount != bill.PayableAmount {
		slog.Error("bill amount mismatch, mark manual", "bill_id", bill.ID,
			"expect_cents", int64(bill.PayableAmount), "got_cents", int64(res.PaidAmount))
		s.db.Model(&bill).Update("status", "manual")
		return respond.ErrBillAmountMismatch
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Bill{}).Where("id = ?", bill.ID).
			Updates(map[string]any{"status": "paid", "paid_at": time.Now(), "pay_channel": string(res.Channel)}).Error; err != nil {
			return err
		}
		// 批量：会话全部未结账子单 → paid，挂 bill_id（一次结账覆盖全部）
		if err := tx.Model(&model.Order{}).Where("session_id = ? AND status = 'unsettled'", bill.SessionID).
			Updates(map[string]any{"status": "paid", "paid_at": time.Now(), "bill_id": bill.ID}).Error; err != nil {
			return err
		}
		pm := model.Payment{
			BillID: &bill.ID, OutTradeNo: res.OutTradeNo, Channel: string(res.Channel),
			ChannelTradeNo: nullableString(res.ChannelTradeNo),
			Amount: res.PaidAmount, Status: "success", CallbackRaw: res.RawPayload,
		}
		if err := tx.Create(&pm).Error; err != nil {
			return err
		}
		// 设计 3.7：支付成功 → 子单批量已支付 + 会话置 settled（桌台可清台，不再接受新单）
		return tx.Model(&model.TableSession{}).Where("id = ?", bill.SessionID).
			Updates(map[string]any{"status": "settled", "locked_for_bill": false}).Error
	})
	if err != nil {
		if isUniqueViolation(err) {
			slog.Error("duplicate bill success payment, mark manual", "bill_id", bill.ID)
			s.db.Model(&bill).Update("status", "manual")
			return respond.ErrDuplicatePay
		}
		return fmt.Errorf("bill paid tx: %w", err)
	}

	s.push("order.paid", map[string]any{
		"session_id": bill.SessionID, "bill_id": bill.ID,
		"channel": res.Channel, "amount_cents": int64(res.PaidAmount),
	})
	slog.Info("bill paid", "bill_id", bill.ID, "session_id", bill.SessionID, "channel", res.Channel)
	return nil
}

// BillPay 前台收银入账 / 代客结账收款（7.4 POST /admin/sessions/:sid/bill/pay）。
// channel: cash / pos / scan。与 CreateBill 同链路聚合金额（含 postpay/frontend 首单按人项），
// 一步完成 建单+入账+子单批量置已支付+会话 settled，钱在前台收，不走支付网关回调（3.7 frontend 态）。
func (s *BillService) BillPay(ctx context.Context, sessionID, operatorID int64, channel string) (*model.Bill, *respond.BizErr) {
	var sess model.TableSession
	if err := s.db.First(&sess, "shop_id = ? AND id = ? AND status = 'active'", s.shopID, sessionID).Error; err != nil {
		return nil, respond.ErrSessionInvalid
	}
	if sess.LockedForBill {
		return nil, respond.ErrSessionLocked
	}
	if channel != "cash" && channel != "pos" && channel != "scan" {
		return nil, respond.ErrBadRequest.WithMsg("入账渠道仅支持 cash/pos/scan")
	}

	var orders []model.Order
	if err := s.db.Where("session_id = ? AND status = 'unsettled'", sess.ID).Find(&orders).Error; err != nil {
		slog.Error("query unsettled orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	if len(orders) == 0 {
		return nil, respond.NewBiz(20009, "暂无未结账订单", 200)
	}
	var total model.Money
	for _, o := range orders {
		total += o.TotalAmount
	}
	if (sess.PayMode == "postpay" || sess.PayMode == "frontend") && !s.hasPriorBill(ctx, sess.ID) {
		charges, bizErr := s.loadPerHead(ctx)
		if bizErr != nil {
			return nil, bizErr
		}
		for _, c := range charges {
			total += c.Price * model.Money(sess.Pax)
		}
	}

	outTradeNo := genNo("CP") // 前台收银支付单号（cash/pos/scan）
	now := time.Now()
	bill := model.Bill{
		ShopID: s.shopID, SessionID: sess.ID,
		TotalAmount: total, PayableAmount: total,
		Status: "paid", PayChannel: &channel, OutTradeNo: &outTradeNo,
		OperatorID: &operatorID,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 原子锁会话（并发前台结账/线上结账只有一个能成功）
		res := tx.Model(&model.TableSession{}).
			Where("id = ? AND locked_for_bill = false AND status = 'active'", sess.ID).
			Update("locked_for_bill", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return respond.ErrSessionLocked
		}
		if err := tx.Model(&model.Bill{}).Where("session_id = ? AND status = 'paid'", sess.ID).
			Update("status", "closed").Error; err != nil {
			return err
		}
		if err := tx.Create(&bill).Error; err != nil {
			return err
		}
		// 子单批量入账 + 记 payments（cash/pos/scan 无渠道回调原文）
		if err := tx.Model(&model.Order{}).Where("session_id = ? AND status = 'unsettled'", sess.ID).
			Updates(map[string]any{"status": "paid", "paid_at": now, "bill_id": bill.ID}).Error; err != nil {
			return err
		}
		pm := model.Payment{
			BillID: &bill.ID, OutTradeNo: outTradeNo,
			Channel: channel, Amount: total, Status: "success",
		}
		if err := tx.Create(&pm).Error; err != nil {
			return err
		}
		return tx.Model(&model.TableSession{}).Where("id = ?", sess.ID).
			Updates(map[string]any{"status": "settled", "locked_for_bill": false}).Error
	})
	if err != nil {
		var be *respond.BizErr
		if errors.As(err, &be) {
			return nil, be
		}
		if isUniqueViolation(err) {
			return nil, respond.NewBiz(20010, "该会话已有结账单，请勿重复结账", 200)
		}
		slog.Error("bill pay tx failed", "err", err)
		return nil, respond.ErrInternal
	}

	// 结账小票出单 + 后厨/收银推送
	s.db.Create(&model.PrintTask{BillID: &bill.ID, PrinterSN: "MOCK", Status: "pending"})
	s.push("order.paid", map[string]any{
		"session_id": sess.ID, "bill_id": bill.ID, "channel": channel,
		"amount_cents": int64(total), "operator": operatorID,
	})
	slog.Info("bill paid at counter", "bill_id", bill.ID, "session_id", sess.ID, "channel", channel, "operator", operatorID)
	return &bill, nil
}

// CloseSession 清台 / 店长核销。mode: close=清台（未结账被拒）/ write_off=核销（强制关台，需 owner）。
func (s *BillService) CloseSession(ctx context.Context, sessionID, operatorID int64, mode string) *respond.BizErr {
	var sess model.TableSession
	if err := s.db.First(&sess, "shop_id = ? AND id = ?", s.shopID, sessionID).Error; err != nil {
		return respond.ErrSessionInvalid
	}
	// 可清台状态：active（后付未结账）或 settled（账单已付，设计 3.7）；closed 不可再关
	if sess.Status != "active" && sess.Status != "settled" {
		return respond.NewBiz(20006, "会话已关闭", 200)
	}

	// 清台强制校验：存在未结账子单或待支付账单 → 拒绝（设计 3.7 清台前置条件）
	// 核销（write_off）是店长强制关台兜底，不受锁单/未结账约束
	if mode == "close" {
		if sess.LockedForBill {
			return respond.NewBiz(20004, "会话结账中，暂不能清台", 200)
		}
		var unsettled, pendingBill int64
		s.db.Model(&model.Order{}).Where("session_id = ? AND status = 'unsettled'", sess.ID).Count(&unsettled)
		s.db.Model(&model.Bill{}).Where("session_id = ? AND status = 'pending'", sess.ID).Count(&pendingBill)
		if unsettled > 0 || pendingBill > 0 {
			return respond.NewBiz(20011, "存在未结账订单或待支付账单，禁止清台", 200)
		}
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 订单终态
		if mode == "write_off" {
			// 核销：未结账子单 → written_off，待支付账单 → written_off，prepay 未支付单 → cancelled
			if err := tx.Model(&model.Order{}).Where("session_id = ? AND status = 'unsettled'", sess.ID).
				Update("status", "written_off").Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Order{}).Where("session_id = ? AND status = 'pending'", sess.ID).
				Update("status", "cancelled").Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Bill{}).Where("session_id = ? AND status = 'pending'", sess.ID).
				Update("status", "written_off").Error; err != nil {
				return err
			}
		}
		// paid 子单 → done，已支付账单 → closed
		if err := tx.Model(&model.Order{}).Where("session_id = ? AND status = 'paid'", sess.ID).
			Updates(map[string]any{"status": "done", "finished_at": time.Now()}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Bill{}).Where("session_id = ? AND status = 'paid'", sess.ID).
			Update("status", "closed").Error; err != nil {
			return err
		}
		// 会话 + 桌台派生状态
		if err := tx.Model(&model.TableSession{}).Where("id = ?", sess.ID).
			Updates(map[string]any{"status": "closed", "closed_at": time.Now(), "locked_for_bill": false}).Error; err != nil {
			return err
		}
		return tx.Model(&model.Table{}).Where("id = ?", sess.TableID).Update("status", "empty").Error
	})
	if err != nil {
		slog.Error("close session tx failed", "err", err)
		return respond.ErrInternal
	}

	slog.Info("session closed", "session_id", sess.ID, "mode", mode, "operator", operatorID)
	return nil
}

func (s *BillService) hasPriorBill(ctx context.Context, sessionID int64) bool {
	var n int64
	s.db.Model(&model.Bill{}).Where("session_id = ?", sessionID).Count(&n)
	return n > 0
}

func (s *BillService) loadPerHead(ctx context.Context) ([]model.PerHeadCharge, *respond.BizErr) {
	var out []model.PerHeadCharge
	if err := s.db.Where("shop_id = ? AND is_active = true", s.shopID).Order("sort").Find(&out).Error; err != nil {
		return nil, respond.ErrInternal
	}
	return out, nil
}
