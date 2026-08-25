// Package handler 顾客端 HTTP 处理（阶段 2.2：登录 / 菜单 / 下单 / 预支付 / mock 回调）。
// 鉴权：login 公开；orders/prepay 需顾客 JWT（middleware.CustomerAuth）。
// 签名：菜单接口需 QR sig（qrsign）。
package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yodian/server/internal/auth"
	"github.com/yodian/server/internal/config"
	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pay"
	"github.com/yodian/server/internal/pkg/qrsign"
	"github.com/yodian/server/internal/pkg/respond"
	"github.com/yodian/server/internal/service"
)

type Customer struct {
	cfg   *config.Config
	db    *gorm.DB
	rdb   *redis.Client // 3.1：手机号验证码 / 发送频率
	svc   *service.OrderService
	bills *service.BillService
	gw    pay.GatewayRegistry
}

func NewCustomer(cfg *config.Config, db *gorm.DB, rdb *redis.Client, svc *service.OrderService, bills *service.BillService, gw pay.GatewayRegistry) *Customer {
	return &Customer{cfg: cfg, db: db, rdb: rdb, svc: svc, bills: bills, gw: gw}
}

func (h *Customer) shopID() int64 { return h.cfg.ShopID }

// Login POST /api/auth/login {channel, code}
// mock 渠道：code 即 channel_uid（16.2 微信静默登录；手机号短信登录后续补齐）
func (h *Customer) Login(c *gin.Context) {
	var req struct {
		Channel string `json:"channel"`
		Code    string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	if req.Code == "" {
		respond.Err(c, respond.ErrCodeInvalid.WithMsg("code 不能为空"))
		return
	}
	if !h.channelEnabled(pay.PayChannel(req.Channel)) {
		respond.Err(c, respond.ErrChannelOff.WithMsg("渠道未启用: %s", req.Channel))
		return
	}

	// mock 换 openid：code 即 channel_uid（接入微信后替换为 wx.Login → openid）
	uid := req.Code
	cust, err := h.upsertCustomer(req.Channel, uid)
	if err != nil {
		slog.Error("upsert customer failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}

	token, err := auth.SignCustomer(h.cfg.JWTCustomerSecret, cust.ID, h.shopID(), req.Channel,
		time.Duration(h.cfg.JWTCustomerTTLH)*time.Hour)
	if err != nil {
		slog.Error("sign customer token failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}
	respond.OK(c, gin.H{"token": token, "customer_id": cust.ID, "channel": req.Channel})
}

func (h *Customer) channelEnabled(ch pay.PayChannel) bool {
	switch ch {
	case pay.ChannelWechat:
		return h.cfg.EnableWechatPay
	case pay.ChannelAlipay:
		return h.cfg.EnableAlipay
	default:
		return false
	}
}

// upsertCustomer 按 (channel, channel_uid) 幂等创建
func (h *Customer) upsertCustomer(channel, uid string) (*model.Customer, error) {
	cust := model.Customer{Channel: channel, ChannelUID: uid}
	err := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel"}, {Name: "channel_uid"}},
		DoNothing: true,
	}).Create(&cust).Error
	if err != nil {
		return nil, err
	}
	if err := h.db.Where("channel = ? AND channel_uid = ?", channel, uid).First(&cust).Error; err != nil {
		return nil, err
	}
	return &cust, nil
}

// Menu GET /api/tables/:tid/menu?sig=xxx 菜单（QR 签名校验 + 沽清标记）
func (h *Customer) Menu(c *gin.Context) {
	tid, err := strconv.ParseInt(c.Param("tid"), 10, 64)
	if err != nil {
		respond.Err(c, respond.ErrBadRequest.WithMsg("桌台参数非法"))
		return
	}
	var t model.Table
	if err := h.db.First(&t, "shop_id = ? AND id = ?", h.shopID(), tid).Error; err != nil {
		respond.Err(c, respond.ErrTableNotFound)
		return
	}
	if !qrsign.Verify(h.cfg.QR.SignSecret, h.shopID(), t.ID, t.QrVersion, c.Query("sig")) {
		respond.Err(c, respond.ErrTableNotFound.WithMsg("桌台不存在或签名无效"))
		return
	}

	// 分类 → 菜品（按 sort 排序，沽清字段原样下发，客户端置灰；下单时服务端二次拦截）
	var cats []model.Category
	if err := h.db.Where("shop_id = ?", h.shopID()).Order("sort, id").Find(&cats).Error; err != nil {
		slog.Error("query categories failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}
	var dishes []model.Dish
	if err := h.db.Where("shop_id = ?", h.shopID()).Order("category_id, sort, id").Find(&dishes).Error; err != nil {
		slog.Error("query dishes failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}
	var perHead []model.PerHeadCharge
	if err := h.db.Where("shop_id = ? AND is_active = true", h.shopID()).Order("sort, id").Find(&perHead).Error; err != nil {
		slog.Error("query per_head failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}

	type dishDTO struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Price     int64  `json:"price"` // 分
		IsSoldOut bool   `json:"is_sold_out"`
		ImageURL  *string `json:"image_url"`
		Specs     any    `json:"specs"`
	}
	catsDTO := make([]gin.H, 0, len(cats))
	for _, cat := range cats {
		ds := make([]dishDTO, 0)
		for _, d := range dishes {
			if d.CategoryID != cat.ID {
				continue
			}
			ds = append(ds, dishDTO{
				ID: d.ID, Name: d.Name, Price: int64(d.Price),
				IsSoldOut: d.IsSoldOut, ImageURL: d.ImageURL, Specs: d.Specs,
			})
		}
		catsDTO = append(catsDTO, gin.H{"id": cat.ID, "name": cat.Name, "dishes": ds})
	}
	phDTO := make([]gin.H, 0, len(perHead))
	for _, p := range perHead {
		// 3.1：补 is_default_checked（结算页默认勾选展示，替代独立 /per-head-charges 端点）
		phDTO = append(phDTO, gin.H{
			"id": p.ID, "name": p.Name, "price": int64(p.Price),
			"is_required": p.IsRequired, "is_default_checked": p.IsDefault,
		})
	}

	respond.OK(c, gin.H{
		"table": gin.H{
			"id": t.ID, "table_no": t.TableNo, "seats": t.Seats, "pay_mode": t.PayMode,
		},
		"categories": catsDTO,
		"per_head":   phDTO,
	})
}

// CreateOrder POST /api/orders 下单（服务端算金额 + 沽清二次校验）
func (h *Customer) CreateOrder(c *gin.Context) {
	cid := c.GetInt64("cid")
	var req service.CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	resp, bizErr := h.svc.CreateOrder(c.Request.Context(), cid, req)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, resp)
}

// CreateBill POST /api/sessions/:sid/bill 结账（后付/前台结，顾客发起）
func (h *Customer) CreateBill(c *gin.Context) {
	sid, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil || sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	bill, bizErr := h.bills.CreateBill(c.Request.Context(), sid)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, gin.H{
		"bill_id": bill.ID, "session_id": sid,
		"out_trade_no": bill.OutTradeNo, "total_amount": int64(bill.PayableAmount),
		"status": bill.Status,
	})
}

// Prepay POST /api/pay/prepay {order_id 或 bill_id, channel} 预下单（7.4；3.1 补 bill_id 支持）。
// order_id=prepay 先付单；bill_id=postpay/frontend 统一结账账单（回调幂等已覆盖 bills，2.3）。
func (h *Customer) Prepay(c *gin.Context) {
	cid := c.GetInt64("cid")
	var req struct {
		OrderID int64  `json:"order_id"`
		BillID  int64  `json:"bill_id"`
		Channel string `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	if req.OrderID == 0 && req.BillID == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("order_id 或 bill_id 必填一项"))
		return
	}
	if req.OrderID != 0 && req.BillID != 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("order_id 与 bill_id 只能二选一"))
		return
	}
	ch := pay.PayChannel(req.Channel)
	if !h.channelEnabled(ch) {
		respond.Err(c, respond.ErrChannelDisabled.WithMsg("支付渠道未启用: %s", req.Channel))
		return
	}
	gw, err := h.gw.Get(ch)
	if err != nil {
		respond.Err(c, respond.ErrChannelDisabled.WithMsg("渠道未接入: %s", err))
		return
	}

	var shop model.Shop
	subject := "悦点点餐"
	if err := h.db.First(&shop, h.shopID()).Error; err == nil {
		subject = shop.Name
	}

	// 归一化支付主体：结账单（后付/前台结统一结账）
	if req.BillID != 0 {
		var bill model.Bill
		if err := h.db.First(&bill, "id = ? AND shop_id = ?", req.BillID, h.shopID()).Error; err != nil {
			respond.Err(c, respond.ErrBadRequest.WithMsg("结账单不存在"))
			return
		}
		if bill.OutTradeNo == nil {
			respond.Err(c, respond.ErrBadRequest.WithMsg("结账单无支付单号"))
			return
		}
		if bill.Status != "pending" {
			respond.Err(c, respond.ErrBillNotPayable.WithMsg("账单状态不允许支付: %s", bill.Status))
			return
		}
		resp, err := gw.Prepay(c.Request.Context(), pay.PrepayReq{
			BillID:      &bill.ID,
			OutTradeNo:  *bill.OutTradeNo,
			TotalAmount: bill.PayableAmount,
			Subject:     subject,
			NotifyURL:   h.cfg.Wechat.NotifyURL,
		})
		if err != nil {
			slog.Error("bill prepay failed", "err", err, "bill_id", bill.ID)
			respond.Err(c, respond.NewBiz(30010, "预下单失败，请稍后重试", 200))
			return
		}
		respond.OK(c, gin.H{
			"out_trade_no": resp.OutTradeNo, "params": resp.Params,
			"amount": int64(bill.PayableAmount), "bill_id": bill.ID,
		})
		return
	}

	// 支付主体：订单（prepay 先付）
	var o model.Order
	if err := h.db.First(&o, "id = ? AND shop_id = ?", req.OrderID, h.shopID()).Error; err != nil {
		respond.Err(c, respond.ErrBadRequest.WithMsg("订单不存在"))
		return
	}
	if o.CustomerID == nil || *o.CustomerID != cid {
		respond.Err(c, respond.ErrForbidden.WithMsg("只能支付本人订单"))
		return
	}
	if o.OutTradeNo == nil {
		respond.Err(c, respond.ErrBadRequest.WithMsg("订单无支付单号"))
		return
	}
	if o.Status != "pending" {
		respond.Err(c, respond.ErrOriginalNotClosed.WithMsg("订单状态不允许支付: %s", o.Status))
		return
	}

	resp, err := gw.Prepay(c.Request.Context(), pay.PrepayReq{
		OrderID:     &o.ID,
		OutTradeNo:  *o.OutTradeNo,
		TotalAmount: o.TotalAmount,
		Subject:     subject,
		NotifyURL:   h.cfg.Wechat.NotifyURL,
	})
	if err != nil {
		slog.Error("prepay failed", "err", err)
		respond.Err(c, respond.NewBiz(30010, "预下单失败，请稍后重试", 200))
		return
	}
	respond.OK(c, gin.H{"out_trade_no": resp.OutTradeNo, "params": resp.Params, "amount": int64(o.TotalAmount)})
}

// MockNotify POST /api/pay/notify/mock 模拟支付成功回调（仅 mock 开关开启时挂载）
// 触发 handlePaidSuccess，走与真实渠道完全相同的内部处理。
func (h *Customer) MockNotify(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	var raw pay.MockNotifyPayload
	if err := json.Unmarshal(body, &raw); err != nil || raw.OutTradeNo == "" {
		respond.Err(c, respond.ErrBadRequest.WithMsg("回调报文非法"))
		return
	}
	ch := pay.PayChannel(raw.Channel)
	gw, err := h.gw.Get(ch)
	if err != nil {
		respond.Err(c, respond.ErrChannelDisabled.WithMsg("渠道未接入: %s", err))
		return
	}
	res, err := gw.VerifyNotify(c.Request.Context(), pay.NotifyRaw{Channel: ch, Payload: body})
	if err != nil {
		respond.Err(c, respond.ErrBadRequest.WithMsg("%s", err))
		return
	}
	if !res.Success {
		respond.Err(c, respond.ErrBadRequest.WithMsg("回调非成功状态"))
		return
	}

	if err := h.svc.HandlePaidSuccess(c.Request.Context(), res); err != nil {
		var be *respond.BizErr
		if errors.As(err, &be) {
			// 金额不符/重复支付：已标记人工或幂等返回，仍回 200 信封阻止渠道重试
			slog.Warn("handle paid failed (biz)", "out_trade_no", res.OutTradeNo, "err", be.Msg)
			respond.Err(c, be)
			return
		}
		slog.Error("handle paid failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}
	respond.OK(c, gin.H{"handled": true, "out_trade_no": res.OutTradeNo})
}

// TableSig GET /api/dev/table-sig/:tid 开发期生成合法签名（仅 dev 挂载）
func (h *Customer) TableSig(c *gin.Context) {
	tid, err := strconv.ParseInt(c.Param("tid"), 10, 64)
	if err != nil {
		respond.Err(c, respond.ErrBadRequest.WithMsg("桌台参数非法"))
		return
	}
	var t model.Table
	if err := h.db.First(&t, "shop_id = ? AND id = ?", h.shopID(), tid).Error; err != nil {
		respond.Err(c, respond.ErrTableNotFound)
		return
	}
	respond.OK(c, gin.H{
		"tid":     t.ID,
		"payload": qrsign.Payload(h.shopID(), t.ID, t.QrVersion),
		"sig":     qrsign.Sign(h.cfg.QR.SignSecret, h.shopID(), t.ID, t.QrVersion),
	})
}

// ---------- 阶段 3.1：顾客端契约缺口补齐 ----------

// SessionOrders GET /api/sessions/:sid/orders：本桌会话全部订单（拼桌共享，9.7）
func (h *Customer) SessionOrders(c *gin.Context) {
	sid, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil || sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	resp, bizErr := h.svc.SessionOrders(c.Request.Context(), sid)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, resp)
}

// UpdatePax PATCH /api/sessions/:sid/pax {pax}：就餐人数（16.1 唯一入口）
func (h *Customer) UpdatePax(c *gin.Context) {
	sid, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil || sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	var req struct {
		Pax int `json:"pax"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	if bizErr := h.svc.UpdatePax(c.Request.Context(), sid, req.Pax); bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, gin.H{"session_id": sid, "pax": req.Pax})
}

// BillPreview GET /api/sessions/:sid/bill/preview：结账预览（只读聚合，不加锁）
func (h *Customer) BillPreview(c *gin.Context) {
	sid, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil || sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	resp, bizErr := h.bills.BillPreview(c.Request.Context(), sid)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, resp)
}

// CancelBill POST /api/sessions/:sid/bill/cancel：取消待支付账单并解锁会话（3.7 重新发起）
func (h *Customer) CancelBill(c *gin.Context) {
	sid, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil || sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	bill, bizErr := h.bills.CancelBill(c.Request.Context(), sid)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, gin.H{"bill_id": bill.ID, "status": bill.Status, "session_id": sid})
}

// SwitchChannel POST /api/pay/switch {order_id, to}：换渠道重付（时序图 4）。
// MVP：订单不记「尝试渠道」，切换即换新 out_trade_no——旧单号迟到回调查不到订单，不会双入账；
// mock CloseOrder 无副作用。真实渠道前需补支付尝试记录（阶段 4，6.10）。
func (h *Customer) SwitchChannel(c *gin.Context) {
	cid := c.GetInt64("cid")
	var req struct {
		OrderID int64  `json:"order_id"`
		To      string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	ch := pay.PayChannel(req.To)
	if !h.channelEnabled(ch) {
		respond.Err(c, respond.ErrChannelDisabled.WithMsg("支付渠道未启用: %s", req.To))
		return
	}
	gw, err := h.gw.Get(ch)
	if err != nil {
		respond.Err(c, respond.ErrChannelDisabled.WithMsg("渠道未接入: %s", err))
		return
	}

	var o model.Order
	if err := h.db.First(&o, "id = ? AND shop_id = ?", req.OrderID, h.shopID()).Error; err != nil {
		respond.Err(c, respond.ErrBadRequest.WithMsg("订单不存在"))
		return
	}
	if o.CustomerID == nil || *o.CustomerID != cid {
		respond.Err(c, respond.ErrForbidden.WithMsg("只能操作本人订单"))
		return
	}
	if o.Status != "pending" {
		respond.Err(c, respond.ErrOriginalNotClosed.WithMsg("订单状态不允许换渠道: %s", o.Status))
		return
	}
	newNo := service.NewOutTradeNo()
	if err := h.db.Model(&o).Update("out_trade_no", newNo).Error; err != nil {
		slog.Error("switch channel update out_trade_no failed", "err", err, "order_id", o.ID)
		respond.Err(c, respond.ErrInternal)
		return
	}

	var shop model.Shop
	subject := "悦点点餐"
	if err := h.db.First(&shop, h.shopID()).Error; err == nil {
		subject = shop.Name
	}
	resp, err := gw.Prepay(c.Request.Context(), pay.PrepayReq{
		OrderID:     &o.ID,
		OutTradeNo:  newNo,
		TotalAmount: o.TotalAmount,
		Subject:     subject,
		NotifyURL:   h.cfg.Wechat.NotifyURL,
	})
	if err != nil {
		slog.Error("switch prepay failed", "err", err)
		respond.Err(c, respond.NewBiz(30010, "换渠道预下单失败，请稍后重试", 200))
		return
	}
	respond.OK(c, gin.H{
		"out_trade_no": resp.OutTradeNo, "params": resp.Params,
		"amount": int64(o.TotalAmount), "to": req.To,
	})
}

// OrderStatus GET /api/orders/:id/status：主动查单（弱网补单定终态，9.8）
func (h *Customer) OrderStatus(c *gin.Context) {
	cid := c.GetInt64("cid")
	oid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || oid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("订单参数非法"))
		return
	}
	resp, bizErr := h.svc.OrderStatus(c.Request.Context(), cid, oid)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, resp)
}

// CallWaiter POST /api/sessions/:sid/call-waiter {reason?}：呼叫服务员
func (h *Customer) CallWaiter(c *gin.Context) {
	sid, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil || sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	cid := c.GetInt64("cid")
	var req struct {
		Reason *string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	call, bizErr := h.svc.CallWaiter(c.Request.Context(), sid, cid, req.Reason)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, gin.H{"call_id": call.ID, "status": call.Status, "session_id": sid})
}

// PhoneCode POST /api/auth/phone-code {phone}：发送手机号验证码（16.1，需顾客 JWT）。
// mock 供应商用固定码并回显 dev_code 便于联调；真实短信接入阶段 4。
func (h *Customer) PhoneCode(c *gin.Context) {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	if !isCNMobile(req.Phone) {
		respond.Err(c, respond.ErrPhoneInvalid)
		return
	}
	ctx := c.Request.Context()

	// 发送频率：每手机号小时窗口限次（SMS_CODE_LIMIT_PER_PHONE）
	sentKey := "sms:sent:" + req.Phone
	n, err := h.rdb.Incr(ctx, sentKey).Result()
	if err != nil {
		slog.Warn("sms sent counter failed", "err", err)
	} else {
		if n == 1 {
			h.rdb.Expire(ctx, sentKey, time.Hour)
		}
		if n > int64(h.cfg.SMS.Limit) {
			respond.Err(c, respond.ErrSendFrequent)
			return
		}
	}

	var code string
	if h.cfg.SMS.Provider == "mock" {
		code = "123456"
	} else {
		code, err = randomSMSCode()
		if err != nil {
			slog.Error("gen sms code failed", "err", err)
			respond.Err(c, respond.ErrInternal)
			return
		}
	}
	codeKey := "sms:code:" + req.Phone
	if err := h.rdb.Set(ctx, codeKey, code, time.Duration(h.cfg.SMS.CodeTTL)*time.Second).Err(); err != nil {
		slog.Error("sms code store failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}

	resp := gin.H{"sent": true}
	if h.cfg.SMS.Provider == "mock" {
		resp["dev_code"] = code
	}
	respond.OK(c, resp)
}

// PhoneBind POST /api/auth/phone-bind {phone, code}：校验验证码并绑定 customers.phone（16.1）
func (h *Customer) PhoneBind(c *gin.Context) {
	cid := c.GetInt64("cid")
	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	if !isCNMobile(req.Phone) {
		respond.Err(c, respond.ErrPhoneInvalid)
		return
	}
	ctx := c.Request.Context()

	want, err := h.rdb.Get(ctx, "sms:code:"+req.Phone).Result()
	if err != nil || !constantTimeEqual(want, req.Code) {
		respond.Err(c, respond.ErrCodeExpired)
		return
	}
	now := time.Now()
	if err := h.db.Model(&model.Customer{}).Where("id = ?", cid).
		Updates(map[string]any{"phone": req.Phone, "phone_verified_at": now}).Error; err != nil {
		slog.Error("phone bind failed", "err", err, "customer_id", cid)
		respond.Err(c, respond.ErrInternal)
		return
	}
	h.rdb.Del(ctx, "sms:code:"+req.Phone)
	slog.Info("phone bound", "customer_id", cid, "phone", req.Phone)
	respond.OK(c, gin.H{"phone": req.Phone, "phone_verified_at": now})
}

var cnMobileRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

func isCNMobile(phone string) bool { return cnMobileRe.MatchString(phone) }

// constantTimeEqual 验证码常量时间比较（防时序侧信道）
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// randomSMSCode 6 位数字验证码（crypto/rand，真实短信供应商场景）
func randomSMSCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := (int(b[0]) << 16) | (int(b[1]) << 8) | int(b[2])
	return fmt.Sprintf("%06d", n%1000000), nil
}
