package pay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yodian/server/internal/model"
)

// MockGateway 开发期模拟网关（设计 8：MOCK_* 开关）。
// 触发 handlePaidSuccess 的内部处理函数与真实渠道完全相同，仅替换「预下单/验签」适配段。
type MockGateway struct {
	channel PayChannel
}

func NewMockGateway(c PayChannel) *MockGateway { return &MockGateway{channel: c} }

func (m *MockGateway) Channel() PayChannel { return m.channel }

// Prepay 返回假调起参数（含测试所需字段）
func (m *MockGateway) Prepay(ctx context.Context, req PrepayReq) (PrepayResp, error) {
	return PrepayResp{
		OutTradeNo: req.OutTradeNo,
		Params: map[string]any{
			"mock":       true,
			"channel":    string(m.channel),
			"out_trade_no": req.OutTradeNo,
			"amount":     int64(req.TotalAmount), // 分
			"subject":    req.Subject,
			"order_id":   req.OrderID,
			"bill_id":    req.BillID,
		},
	}, nil
}

// MockNotifyPayload 本地模拟回调请求体（金额单位：分）
type MockNotifyPayload struct {
	Channel        string `json:"channel"` // wechat / alipay，用于路由到对应网关
	OutTradeNo     string `json:"out_trade_no"`
	ChannelTradeNo string `json:"channel_trade_no"`
	Amount         int64  `json:"amount"`
}

// VerifyNotify mock 不做验签（直通），解析模拟回调报文为统一结果
func (m *MockGateway) VerifyNotify(ctx context.Context, raw NotifyRaw) (NotifyResult, error) {
	var p MockNotifyPayload
	if err := json.Unmarshal(raw.Payload, &p); err != nil {
		return NotifyResult{}, fmt.Errorf("mock notify: %w", err)
	}
	if p.OutTradeNo == "" {
		return NotifyResult{}, fmt.Errorf("mock notify: out_trade_no required")
	}
	return NotifyResult{
		Channel:        m.channel,
		OutTradeNo:     p.OutTradeNo,
		ChannelTradeNo: p.ChannelTradeNo,
		PaidAmount:     model.Fen(p.Amount),
		Success:        true,
		RawPayload:     raw.Payload,
	}, nil
}

// QueryOrder 模拟查单：无状态，默认 NOTPAY（真实弱网补单逻辑在接入真实渠道后生效）
func (m *MockGateway) QueryOrder(ctx context.Context, outTradeNo string) (TradeStatus, error) {
	return TradeNotPay, nil
}

// CloseOrder 模拟关单：无副作用
func (m *MockGateway) CloseOrder(ctx context.Context, outTradeNo string) error {
	return nil
}

// Refund 模拟退款：直接成功
func (m *MockGateway) Refund(ctx context.Context, req RefundReq) (RefundResp, error) {
	return RefundResp{OutRefundNo: req.OutRefundNo, Success: true}, nil
}
