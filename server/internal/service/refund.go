// 退款分流（设计 3.9，阶段 2.3 先付全退/部分退；账单级退款随 2.4 对账落地）。
// 状态：prepay paid/preparing/served → refunded（全额）或 partially_refunded（部分）。
// 退款金额 = paid_amount - 已退；退款调用网关 Refund（mock 直接成功），记录 refunds 存根。
package service

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

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
	return &RefundService{db: db, gw: gw, shopID: shopID}
}

// RefundReq 退款入参；amount 为 0 表示全额退款
type RefundReq struct {
	Amount int64  `json:"amount"`
	Reason string `json:"reason"`
}

// RefundOrder 全退/部分退。order 必须是 prepay 且处于可退状态。
func (s *RefundService) RefundOrder(ctx context.Context, orderID, operatorID int64, req RefundReq) (*model.Refund, *respond.BizErr) {
	var o model.Order
	if err := s.db.First(&o, "shop_id = ? AND id = ?", s.shopID, orderID).Error; err != nil {
		return nil, respond.NewBiz(20012, "订单不存在", 200)
	}
	if o.PayMode != "prepay" {
		return nil, respond.NewBiz(20013, "仅先付订单支持线上退款", 200)
	}
	// 可退状态：paid / preparing / served / partially_refunded（prepay 主干）
	if !(o.Status == "paid" || o.Status == "preparing" || o.Status == "served" || o.Status == "partially_refunded") {
		return nil, respond.NewBiz(20014, fmt.Sprintf("订单状态不可退款: %s", o.Status), 200)
	}

	paid := o.TotalAmount
	if o.PaidAmount != nil {
		paid = *o.PaidAmount
	}
	refunded := s.alreadyRefunded(ctx, o.ID)
	refundable := paid - refunded
	if refundable <= 0 {
		return nil, respond.NewBiz(20015, "订单已全额退款", 200)
	}

	amount := model.Money(req.Amount)
	if amount == 0 || amount > refundable {
		amount = refundable
	}

	// 调网关退款（mock 成功；真实渠道接入后替换）
	gw, err := s.gw.Get(orderChannel(&o))
	if err != nil {
		return nil, respond.ErrChannelDisabled.WithMsg("退款渠道未接入: %s", err)
	}
	outRefundNo := genRefundNo()
	if _, err := gw.Refund(ctx, pay.RefundReq{
		OutTradeNo:  *o.OutTradeNo,
		OutRefundNo: outRefundNo,
		Amount:      amount,
		Reason:      req.Reason,
	}); err != nil {
		slog.Error("gateway refund failed", "err", err, "order_id", o.ID)
		return nil, respond.NewBiz(30007, "退款失败，请稍后重试", 200)
	}

	// 状态机流转 + 退款记录（同事务）
	to := "partially_refunded"
	if amount == paid {
		to = "refunded"
	}
	if bizErr := AssertTransition(o.PayMode, o.Status, to); bizErr != nil {
		return nil, bizErr
	}
	rf := model.Refund{
		OrderID: &o.ID, OutRefundNo: outRefundNo, Channel: string(orderChannel(&o)),
		Amount: amount, Status: "success", Reason: nullableString(req.Reason), OperatorID: &operatorID,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&rf).Error; err != nil {
			return err
		}
		return tx.Model(&model.Order{}).Where("id = ?", o.ID).Update("status", to).Error
	})
	if err != nil {
		slog.Error("refund tx failed", "err", err)
		return nil, respond.ErrInternal
	}
	slog.Info("order refunded", "order_id", o.ID, "amount_cents", int64(amount), "to", to, "operator", operatorID)
	return &rf, nil
}

func (s *RefundService) alreadyRefunded(ctx context.Context, orderID int64) model.Money {
	var total model.Money
	s.db.Model(&model.Refund{}).Where("order_id = ? AND status IN ('success','pending')", orderID).
		Select("COALESCE(SUM(amount), 0)").Scan(&total)
	return total
}

// orderChannel 已支付订单必有支付渠道；异常兜底 wechat（对账与告警在 2.4 覆盖）
func orderChannel(o *model.Order) pay.PayChannel {
	if o.PayChannel != nil {
		return pay.PayChannel(*o.PayChannel)
	}
	return pay.ChannelWechat
}

func genRefundNo() string {
	return genNo("RF")
}
