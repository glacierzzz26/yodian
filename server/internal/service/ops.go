// 运营支撑：沽清开关 + 六口径对账（设计 5.3 / 7.4 /admin/recon）。
package service

import (
	"context"
	"log/slog"

	"gorm.io/gorm"

	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pkg/respond"
)

type OpsService struct {
	db     *gorm.DB
	shopID int64
}

func NewOpsService(db *gorm.DB, shopID int64) *OpsService {
	return &OpsService{db: db, shopID: shopID}
}

// SetSoldOut 标记沽清/恢复（7.4 POST /admin/dishes/:id/soldout）
func (s *OpsService) SetSoldOut(ctx context.Context, dishID int64, soldOut bool) *respond.BizErr {
	res := s.db.Model(&model.Dish{}).Where("shop_id = ? AND id = ?", s.shopID, dishID).
		Update("is_sold_out", soldOut)
	if res.Error != nil {
		slog.Error("set soldout failed", "err", res.Error)
		return respond.ErrInternal
	}
	if res.RowsAffected == 0 {
		return respond.NewBiz(20016, "菜品不存在", 200)
	}
	return nil
}

// ReconSnapshot 六口径对账快照（金额单位：分，与 API 其它金额一致）。
// 口径（5.3）：payments 成功流水按渠道（微信/支付宝/现金/POS/扫）= 六口径合计；
// 应收 = 订单直付（prepay 单笔支付）+ 已支付账单（后付/前台结，含按人项）；
// 挂账/核销单独列示、不混入资金流。diff = 应收合计 - 支付实收，0 为平。
type ReconSnapshot struct {
	OrderReceivable int64            `json:"order_receivable"` // 订单直付应收（prepay，分）
	BillReceivable  int64            `json:"bill_receivable"`  // 已支付账单应收（含按人项，分）
	TotalReceivable int64            `json:"total_receivable"` // 应收合计 = 订单 + 账单
	PaymentPaid     int64            `json:"payment_paid"`     // 六口径合计 = 成功支付（分）
	PaymentByChan   map[string]int64 `json:"payment_by_channel"`
	Refunded        int64            `json:"refunded"`      // 退款总额（分）
	NetReceipt      int64            `json:"net_receipt"`   // 实收净额 = 支付 - 退款
	UnsettledAmount int64            `json:"unsettled_amount"` // 挂账额：已出餐未收款（分，单独列示）
	UnsettledOrders int64            `json:"unsettled_orders"`
	PendingBills    int64            `json:"pending_bills"`
	WrittenOffAmount int64           `json:"written_off_amount"` // 核销额：订单侧损耗（分）
	ManualOrders    int64            `json:"manual_orders"`
	OrderCount      int64            `json:"order_count"`
	OrderByStatus   map[string]int64 `json:"order_by_status"`
	Diff            int64            `json:"diff"` // total_receivable - payment_paid，0 为平
}

// Recon 生成六口径对账快照。注：DB 金额列以元存储（numeric），SQL 内 ×100 折分。
func (s *OpsService) Recon(ctx context.Context) (*ReconSnapshot, *respond.BizErr) {
	snap := &ReconSnapshot{
		OrderByStatus: map[string]int64{},
		PaymentByChan: map[string]int64{},
	}

	// ① 订单状态拆分（展示 + 挂账/核销单独列示）
	var rows []struct {
		Status string
		Sum    int64
		Count  int64
	}
	if err := s.db.Raw(`SELECT status, (COALESCE(SUM(total_amount),0)*100)::bigint AS sum,
			COUNT(*)::bigint AS count FROM orders WHERE shop_id = ? GROUP BY status`,
		s.shopID).Scan(&rows).Error; err != nil {
		slog.Error("recon orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	for _, r := range rows {
		snap.OrderByStatus[r.Status] = r.Sum
		snap.OrderCount += r.Count
		switch r.Status {
		case "unsettled":
			snap.UnsettledAmount += r.Sum
			snap.UnsettledOrders += r.Count
		case "written_off":
			snap.WrittenOffAmount += r.Sum
		case "manual":
			snap.ManualOrders += r.Count
		}
	}

	// ② 订单直付应收：prepay 单笔支付（未挂账单的已支付订单）
	if err := s.db.Raw(`SELECT (COALESCE(SUM(total_amount),0)*100)::bigint FROM orders
			WHERE shop_id = ? AND bill_id IS NULL
			  AND status IN ('paid','preparing','served','done','partially_refunded')`, s.shopID).
		Scan(&snap.OrderReceivable).Error; err != nil {
		slog.Error("recon direct orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	// ③ 已支付账单应收（后付/前台结，含按人项；closed = 已支付后的清台归档，同为已收）
	if err := s.db.Raw(`SELECT (COALESCE(SUM(payable_amount),0)*100)::bigint FROM bills
			WHERE shop_id = ? AND status IN ('paid','closed')`, s.shopID).Scan(&snap.BillReceivable).Error; err != nil {
		slog.Error("recon paid bills failed", "err", err)
		return nil, respond.ErrInternal
	}
	// ④ 六口径资金流：payments 成功流水按渠道（微信/支付宝/现金/POS/扫）
	var chans []struct {
		Channel string
		Sum     int64
	}
	if err := s.db.Raw(`SELECT channel, (COALESCE(SUM(amount),0)*100)::bigint AS sum
			FROM payments WHERE status = 'success' GROUP BY channel`).Scan(&chans).Error; err != nil {
		slog.Error("recon payments failed", "err", err)
		return nil, respond.ErrInternal
	}
	for _, r := range chans {
		snap.PaymentByChan[r.Channel] = r.Sum
		snap.PaymentPaid += r.Sum
	}
	// ⑤ 退款总额
	if err := s.db.Raw(`SELECT (COALESCE(SUM(amount),0)*100)::bigint
			FROM refunds WHERE status IN ('success','pending')`).Scan(&snap.Refunded).Error; err != nil {
		slog.Error("recon refunds failed", "err", err)
		return nil, respond.ErrInternal
	}
	// ⑥ 待支付账单（挂账提醒）
	if err := s.db.Raw(`SELECT COUNT(*)::bigint FROM bills
			WHERE shop_id = ? AND status = 'pending'`, s.shopID).Scan(&snap.PendingBills).Error; err != nil {
		slog.Error("recon pending bills failed", "err", err)
		return nil, respond.ErrInternal
	}

	snap.TotalReceivable = snap.OrderReceivable + snap.BillReceivable
	snap.NetReceipt = snap.PaymentPaid - snap.Refunded
	snap.Diff = snap.TotalReceivable - snap.PaymentPaid // 5.3：应收合计应等于六口径合计
	return snap, nil
}
