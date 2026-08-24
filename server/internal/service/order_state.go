// 订单状态机（设计 7.2）：按 pay_mode 三条主干分流，状态迁移只在服务端校验。
// 入库枚举 ASCII（7.1 枚举表唯一权威）。
package service

import "github.com/yodian/server/internal/pkg/respond"

// prepay 主干（7.2 A）：pending→paid→preparing→served→done
var prepayTransitions = map[string]map[string]bool{
	"pending":          {"paid": true, "cancelled": true, "manual": true},
	"paid":             {"preparing": true, "refunded": true, "partially_refunded": true, "manual": true},
	"preparing":        {"served": true, "partially_refunded": true, "refunded": true, "manual": true},
	"partially_refunded": {"served": true, "refunded": true, "manual": true},
	"served":           {"done": true, "partially_refunded": true, "refunded": true, "manual": true},
	"done":             {"manual": true},
	"cancelled":        {"manual": true},
	"refunded":         {"manual": true},
	"manual":           {}, // 需人工处理：阻断自动流转，等人工介入
}

// postpay 主干（7.2 B）：unsettled→paid→done；unsettled→voided/written_off/manual
var postpayTransitions = map[string]map[string]bool{
	"unsettled":     {"paid": true, "voided": true, "written_off": true, "manual": true},
	"paid":          {"done": true, "partially_refunded": true, "manual": true},
	"partially_refunded": {"done": true, "manual": true},
	"voided":        {"manual": true},
	"written_off":   {"manual": true},
	"done":          {"manual": true},
	"manual":        {},
}

// frontend 主干（7.2 C）：与后付共享 unsettled，区别在收款主体
var frontendTransitions = map[string]map[string]bool{
	"unsettled": {"paid": true, "voided": true, "manual": true},
	"paid":      {"done": true, "partially_refunded": true, "manual": true},
	"partially_refunded": {"done": true, "manual": true},
	"voided":    {"manual": true},
	"done":      {"manual": true},
	"manual":    {},
}

func transitionsOf(payMode string) map[string]map[string]bool {
	switch payMode {
	case "postpay":
		return postpayTransitions
	case "frontend":
		return frontendTransitions
	default:
		return prepayTransitions
	}
}

// InitialStatus 新订单初始状态：先付=待支付；后付/前台结=未结账（提交即出单，3.7）
func InitialStatus(payMode string) string {
	if payMode == "prepay" {
		return "pending"
	}
	return "unsettled"
}

// CanTransition 校验状态迁移是否合法
func CanTransition(payMode, from, to string) bool {
	return transitionsOf(payMode)[from][to]
}

// ValidStatuses 某主干的全部合法状态
func ValidStatuses(payMode string) []string {
	out := make([]string, 0, len(transitionsOf(payMode)))
	for s := range transitionsOf(payMode) {
		out = append(out, s)
	}
	return out
}

// AssertTransition 带错误返回的状态迁移校验（服务端最终防线）
func AssertTransition(payMode, from, to string) *respond.BizErr {
	if !CanTransition(payMode, from, to) {
		return respond.NewBiz(20010, "非法状态迁移: "+from+" → "+to, 200)
	}
	return nil
}
