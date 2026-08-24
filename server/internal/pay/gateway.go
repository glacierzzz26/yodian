// Package pay 支付网关抽象层（设计 3.2 本版核心）。
// 业务层只依赖 PaymentGateway 接口，按 pay_channel 路由；微信/支付宝/Mock 各自实现。
// 金额一律 int64 分；RawPayload 必须原文入库存档（对账与纠纷举证唯一凭据）。
package pay

import (
	"context"
	"fmt"

	"github.com/yodian/server/internal/model"
)

type PayChannel string

const (
	ChannelWechat PayChannel = "wechat"
	ChannelAlipay PayChannel = "alipay"
)

// PrepayReq 预下单请求（order_id / bill_id 二选一，由上层保证）
type PrepayReq struct {
	OrderID     *int64
	BillID      *int64
	OutTradeNo  string
	TotalAmount model.Money
	Subject     string
	ChannelUID  string // 微信 openid / 支付宝 user_id
	NotifyURL   string
}

// PrepayResp 预下单返回，Params 为前端调起支付所需参数（渠道各异）
type PrepayResp struct {
	OutTradeNo string
	Params     map[string]any
}

// NotifyRaw 渠道回调原始报文
type NotifyRaw struct {
	Channel PayChannel
	Payload []byte
}

// NotifyResult 各渠道回调归一化后的统一结果
type NotifyResult struct {
	Channel        PayChannel
	OutTradeNo     string
	ChannelTradeNo string // 微信 transaction_id / 支付宝 trade_no
	PaidAmount     model.Money
	Success        bool
	RawPayload     []byte // 原文存档
}

// TradeStatus 主动查单结果（弱网补单/对账）
type TradeStatus string

const (
	TradeSuccess TradeStatus = "SUCCESS"
	TradeNotPay  TradeStatus = "NOTPAY"
	TradeClosed  TradeStatus = "CLOSED"
)

// RefundReq 退款请求
type RefundReq struct {
	OutTradeNo  string // 原支付单号
	OutRefundNo string // 退款单号（全局唯一）
	Amount      model.Money
	Reason      string
}

type RefundResp struct {
	OutRefundNo string
	ChannelRefundNo string
	Success     bool
}

// PaymentGateway 统一支付网关接口
type PaymentGateway interface {
	Channel() PayChannel
	// Prepay 预下单，返回前端调起支付所需参数
	Prepay(ctx context.Context, req PrepayReq) (PrepayResp, error)
	// VerifyNotify 验签并解析回调，失败必须返回 error（禁止吞验签错误）
	VerifyNotify(ctx context.Context, raw NotifyRaw) (NotifyResult, error)
	// QueryOrder 主动查单，用于弱网补单与对账
	QueryOrder(ctx context.Context, outTradeNo string) (TradeStatus, error)
	// CloseOrder 关闭未支付订单，换渠道重付前必须调用（6.10）
	CloseOrder(ctx context.Context, outTradeNo string) error
	// Refund 退款，支持部分退款
	Refund(ctx context.Context, req RefundReq) (RefundResp, error)
}

// GatewayRegistry 运行时路由
type GatewayRegistry map[PayChannel]PaymentGateway

func (r GatewayRegistry) Get(c PayChannel) (PaymentGateway, error) {
	gw, ok := r[c]
	if !ok {
		return nil, fmt.Errorf("unsupported pay channel: %s", c)
	}
	return gw, nil
}
