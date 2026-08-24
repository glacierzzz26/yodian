package respond

import "fmt"

// BizErr 业务错误：code 为业务错误码（16.1 区间表），HTTP 为对应 HTTP 状态
type BizErr struct {
	Code int
	Msg  string
	HTTP int
}

func (e *BizErr) Error() string { return e.Msg }

func NewBiz(code int, msg string, http int) *BizErr { return &BizErr{Code: code, Msg: msg, HTTP: http} }

// WithMsg 复制错误并替换消息（保留 code/HTTP）
func (e *BizErr) WithMsg(format string, args ...any) *BizErr {
	return &BizErr{Code: e.Code, Msg: fmt.Sprintf(format, args...), HTTP: e.HTTP}
}

// 错误码区间（16.1）：0 成功 / 1xxxx 顾客端参数 / 2xxxx 订单桌台会话 /
// 3xxxx 支付 / 4xxxx 鉴权权限 / 5xxxx 系统第三方

// ---- 通用 ----
var (
	ErrBadRequest    = NewBiz(10000, "参数错误", 400)
	ErrNotFound      = NewBiz(40400, "资源不存在", 404)
	ErrUnauthorized  = NewBiz(40000, "未登录或登录已过期", 401)
	ErrForbidden     = NewBiz(40003, "无权限访问该功能", 403)
	ErrTooManyReqs   = NewBiz(42900, "请求过于频繁，请稍后再试", 429)
	ErrInternal      = NewBiz(50000, "系统繁忙，请稍后重试", 500)
	ErrNotImplemented = NewBiz(50001, "功能尚未实现", 501)
)

// ---- 1xxxx 顾客端 / 参数 ----
var (
	ErrSendFrequent   = NewBiz(10001, "发送过于频繁", 200)
	ErrCodeExpired    = NewBiz(10002, "验证码错误或过期", 200)
	ErrPhoneInvalid   = NewBiz(10003, "手机号格式非法", 200)
)

// ---- 2xxxx 订单 / 桌台 / 会话 ----
var (
	ErrTableNotFound  = NewBiz(20001, "桌台不存在或签名无效", 200)
	ErrDishSoldOut    = NewBiz(20002, "菜品已沽清，请刷新菜单", 200)
	ErrAmountMismatch = NewBiz(20003, "金额不符，请重新下单", 200)
	ErrSessionLocked  = NewBiz(20004, "会话结账中，禁止加菜", 200)
	ErrBillExists     = NewBiz(20005, "会话已有待支付账单", 200)
	ErrSessionClosed  = NewBiz(20006, "会话已锁定或关闭", 200)
	ErrBillTotalMismatch = NewBiz(20007, "子单合计与账单不符", 200)
	ErrSessionInvalid = NewBiz(20008, "会话不存在或已结束", 200)
	ErrSessionSettled = NewBiz(20017, "会话已结账，请先清台后再开台", 200)
)

// ---- 3xxxx 支付 ----
var (
	ErrChannelDisabled  = NewBiz(30001, "支付渠道未启用", 200)
	ErrOriginalNotClosed = NewBiz(30002, "原渠道订单未关闭", 200)
	ErrPayAmountMismatch = NewBiz(30003, "回调金额与服务端不符", 200)
	ErrDuplicatePay     = NewBiz(30004, "该订单已支付，请勿重复支付", 200)
	ErrBillNotPayable   = NewBiz(30005, "账单状态不允许入账", 200)
	ErrBillAmountMismatch = NewBiz(30006, "入账金额与账单不符", 200)
)

// ---- 4xxxx 鉴权 / 权限 ----
var (
	ErrCodeInvalid = NewBiz(40001, "登录凭证无效", 200)
	ErrChannelOff  = NewBiz(40003, "该渠道未启用", 200)
	ErrCredInvalid = NewBiz(40004, "工号或密码错误", 200)
	ErrStaffDisabled = NewBiz(40005, "账号已停用，请联系店长", 200)
)
