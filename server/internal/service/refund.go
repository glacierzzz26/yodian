// 退款分流（设计 3.9，阶段 2.3 先付全退/部分退；账单级退款随 2.4 对账落地）。
// 状态：prepay paid/preparing/served/partially_refunded → refunded（全额）或 partially_refunded（部分）。
// 退款金额 = paid_amount - 已退；退款调用网关 Refund（mock 直接成功），记录 refunds 存根。
//
// P0-3：金额必须 0 < amount <= refundable（0 表示全额）；读已退 + 写 refund + 置订单状态
// 在一个持行锁的事务内完成（先落 pending 存根），网关调用在事务提交后进行——失败则回写 failed 并回退订单状态。
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pay"
	"github.com/yodian/server/internal/pkg/respond"
)

type RefundService struct {
	db     *gorm.DB
	shopID int64
	gw     pay.GatewayRegistry
}

func NewRefundService(db *gorm.DB, gw pay.GatewayRegistry, shopID int64) *RefundService {
	return &RefundService{db: db, shopID: shopID, gw: gw}
}

// RefundReq 退款入参；amount 为 0 表示全额退款
type RefundReq struct {
	Amount int64  `json:"amount"`
	Reason string `json:"reason"`
}

// RefundOrder 全退/部分退。order 必须是 prepay 且处于可退状态。
// P0-3 修复：金额正负/超额严格校验；行锁事务内重算可退额；退款可在提交后按网关结果回写。
func (s *RefundService) RefundOrder(ctx context.Context, orderID, operatorID int64, req RefundReq) (*model.Refund, *respond.BizErr) {
	var (
		o           model.Order
		rf          model.Refund
		origStatus  string
		outRefundNo = genRefundNo()
		needGateway bool
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 行锁读订单：并发部分退款串行化，避免总额超实收
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&o, "shop_id = ? AND id = ?", s.shopID, orderID).Error; err != nil {
			return respond.NewBiz(20012, "订单不存在", 200)
		}
		if o.PayMode != "prepay" {
			return respond.NewBiz(20013, "仅先付订单支持线上退款", 200)
		}
		if !(o.Status == "paid" || o.Status == "preparing" || o.Status == "served" || o.Status == "partially_refunded") {
			return respond.NewBiz(20014, fmt.Sprintf("订单状态不可退款: %s", o.Status), 200)
		}

		paid := o.TotalAmount
		if o.PaidAmount != nil {
			paid = *o.PaidAmount
		}
		refunded, err := alreadyRefunded(tx, o.ID)
		if err != nil {
			slog.Error("sum refunded failed", "err", err, "order_id", o.ID)
			return respond.ErrInternal
		}
		refundable := paid - refunded
		if refundable <= 0 {
			return respond.NewBiz(20015, "订单已全额退款", 200)
		}

		// 金额校验：0 = 全额；否则必须 0 < amount <= refundable（拒绝负数/超额）
		amount := model.Money(req.Amount)
		if amount == 0 {
			amount = refundable
		} else if amount < 0 || amount > refundable {
			return respond.NewBiz(30008, "退款金额非法", 200)
		}

		// 目标状态：退尽余额 → refunded，否则 partially_refunded
		to := "partially_refunded"
		if amount == refundable {
			to = "refunded"
		}
		if bizErr := AssertTransition(o.PayMode, o.Status, to); bizErr != nil {
			return bizErr
		}

		// 渠道：线上渠道走网关（先落 pending，提交后调用）；线下渠道当面退现，直接成功
		ch := orderChannel(&o)
		needGateway = ch == pay.ChannelWechat || ch == pay.ChannelAlipay
		rfStatus := "success"
		if needGateway {
			rfStatus = "pending"
		}

		origStatus = o.Status
		rf = model.Refund{
			OrderID: &o.ID, OutRefundNo: outRefundNo, Channel: string(ch),
			Amount: amount, Status: rfStatus, Reason: nullableString(req.Reason), OperatorID: &operatorID,
		}
		if err := tx.Create(&rf).Error; err != nil {
			return err
		}
		return tx.Model(&model.Order{}).Where("id = ?", o.ID).Update("status", to).Error
	})
	if err != nil {
		var be *respond.BizErr
		if errors.As(err, &be) {
			return nil, be
		}
		slog.Error("refund tx failed", "err", err, "order_id", orderID)
		return nil, respond.ErrInternal
	}

	// 线下渠道：无网关，已入账成功
	if !needGateway {
		slog.Info("offline cash refund, skip gateway", "order_id", o.ID, "channel", rf.Channel, "amount_cents", int64(rf.Amount))
		return &rf, nil
	}

	// 网关退款（事务外调用；失败回写 failed 并回退订单状态）
	gw, gerr := s.gw.Get(pay.PayChannel(rf.Channel))
	if gerr != nil {
		s.revertFailedRefund(&rf, origStatus)
		return nil, respond.ErrChannelDisabled.WithMsg("退款渠道未接入: %s", gerr)
	}
	if _, gerr := gw.Refund(ctx, pay.RefundReq{
		OutTradeNo:  derefStr(o.OutTradeNo),
		OutRefundNo: outRefundNo,
		Amount:      rf.Amount,
		Reason:      req.Reason,
	}); gerr != nil {
		slog.Error("gateway refund failed", "err", gerr, "order_id", o.ID)
		s.revertFailedRefund(&rf, origStatus)
		return nil, respond.NewBiz(30007, "退款失败，请稍后重试", 200)
	}

	if err := s.db.Model(&model.Refund{}).Where("id = ?", rf.ID).Update("status", "success").Error; err != nil {
		// 钱已退但存根未置成功：记错误，运维可按 out_refund_no 对账修正
		slog.Error("mark refund success failed", "err", err, "refund_id", rf.ID, "out_refund_no", outRefundNo)
	}
	rf.Status = "success"
	slog.Info("order refunded", "order_id", o.ID, "amount_cents", int64(rf.Amount), "to", origStatus, "operator", operatorID)
	return &rf, nil
}

// revertFailedRefund 网关退款失败：退款存根置 failed，订单状态回退到退款前（存根保留供审计）。
func (s *RefundService) revertFailedRefund(rf *model.Refund, origStatus string) {
	if err := s.db.Model(&model.Refund{}).Where("id = ?", rf.ID).Update("status", "failed").Error; err != nil {
		slog.Error("mark refund failed status failed", "err", err, "refund_id", rf.ID)
	}
	rf.Status = "failed"
	if err := s.db.Model(&model.Order{}).Where("id = ?", *rf.OrderID).Update("status", origStatus).Error; err != nil {
		slog.Error("revert order status failed", "err", err, "order_id", *rf.OrderID, "to", origStatus)
	}
}

// alreadyRefunded 订单已退（含处理中 pending）金额合计
func alreadyRefunded(db *gorm.DB, orderID int64) (model.Money, error) {
	var total model.Money
	err := db.Model(&model.Refund{}).Where("order_id = ? AND status IN ('success','pending')", orderID).
		Select("COALESCE(SUM(amount), 0)").Scan(&total).Error
	return total, err
}

// orderChannel 已支付订单必有支付渠道；异常兜底 wechat（对账与告警在 2.4 覆盖）
func orderChannel(o *model.Order) pay.PayChannel {
	if o.PayChannel != nil {
		return pay.PayChannel(*o.PayChannel)
	}
	return pay.ChannelWechat
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func genRefundNo() string {
	return genNo("RF")
}
