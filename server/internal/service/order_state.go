// 订单状态机（设计 7.2）：按 pay_mode 三条主干分流，状态迁移只在服务端校验。
// 入库枚举 ASCII（7.1 枚举表唯一权威）。
package service

import "github.com/yodian/server/internal/pkg/respond"

// prepay 主干（7.2 A）：pending→paid→preparing→served→done
var prepayTransitions = map[string]map[string]bool{
	"pending":            {"paid": true, "cancelled": true, "manual": true},
	"paid":               {"preparing": true, "refunded": true, "partially_refunded": true, "manual": true},
	"preparing":          {"served": true, "partially_refunded": true, "refunded": true, "manual": true},
	"partially_refunded": {"served": true, "refunded": true, "manual": true},
	"served":             {"done": true, "partially_refunded": true, "refunded": true, "manual": true},
	"done":               {"manual": true},
	"cancelled":          {"manual": true},
	"refunded":           {"manual": true},
	"manual":             {}, // 需人工处理：阻断自动流转，等人工介入
}

// postpay 主干（7.2 B）：unsettled→paid→done；unsettled→voided/written_off/manual
var postpayTransitions = map[string]map[string]bool{
	"unsettled":          {"paid": true, "voided": true, "written_off": true, "manual": true},
	"paid":               {"done": true, "partially_refunded": true, "manual": true},
	"partially_refunded": {"done": true, "manual": true},
	"voided":             {"manual": true},
	"written_off":        {"manual": true},
	"done":               {"manual": true},
	"manual":             {},
}

// frontend 主干（7.2 C）：与后付共享 unsettled，区别在收款主体
var frontendTransitions = map[string]map[string]bool{
	"unsettled":          {"paid": true, "voided": true, "manual": true},
	"paid":               {"done": true, "partially_refunded": true, "manual": true},
	"partially_refunded": {"done": true, "manual": true},
	"voided":             {"manual": true},
	"done":               {"manual": true},
	"manual":             {},
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

// ---- P0-2：状态机的角色二次收紧（16.2 权限矩阵） ----
// 状态机合法 ≠ 该角色有权。通用改状态端点 /admin/orders/:id/status（cashier|owner）与
// 后厨 KDS /admin/kds/:id/move（kitchen|owner）共用本校验：
//   - kitchen 仅「接单/出餐」（7.2 A 先付：paid→preparing→served），不得置 paid（白吃）或终态；
//   - cashier 不得置退款态（绕开 RefundService 不入 refunds 表 → 对账退款恒 0）与核销态（owner 专属）；
//   - owner 不得置退款态——退款唯一写入口是 RefundService（RefundOrder 内部用 AssertTransition）。
var (
	// kitchenTransitions 后厨可达迁移（先付 KDS 接单/出餐）
	kitchenTransitions = map[string]map[string]bool{
		"paid":      {"preparing": true},
		"preparing": {"served": true},
	}
	// cashierForbidden 收银员不可达状态
	cashierForbidden = map[string]bool{
		"refunded": true, "partially_refunded": true, "written_off": true,
	}
	// ownerDirectForbidden 店长经通用端点也不可达（退款只能走 RefundService）
	ownerDirectForbidden = map[string]bool{
		"refunded": true, "partially_refunded": true,
	}
)

// AssertRoleTransition 状态机 + 角色双重校验；未知角色一律拒绝
func AssertRoleTransition(role, payMode, from, to string) *respond.BizErr {
	if bizErr := AssertTransition(payMode, from, to); bizErr != nil {
		return bizErr
	}
	switch role {
	case "kitchen":
		if !kitchenTransitions[from][to] {
			return respond.NewBiz(40003, "后厨仅可接单/出餐", 403)
		}
	case "cashier":
		if cashierForbidden[to] {
			return respond.NewBiz(40003, "该状态仅店长可操作", 403)
		}
	case "owner":
		if ownerDirectForbidden[to] {
			return respond.NewBiz(40003, "退款请走退款接口", 403)
		}
	default:
		return respond.NewBiz(40003, "无权限操作订单状态", 403)
	}
	return nil
}
