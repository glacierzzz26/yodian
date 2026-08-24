// Package handler 顾客端 HTTP 处理（阶段 2.2：登录 / 菜单 / 下单 / 预支付 / mock 回调）。
// 鉴权：login 公开；orders/prepay 需顾客 JWT（middleware.CustomerAuth）。
// 签名：菜单接口需 QR sig（qrsign）。
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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
	svc   *service.OrderService
	bills *service.BillService
	gw    pay.GatewayRegistry
}

func NewCustomer(cfg *config.Config, db *gorm.DB, svc *service.OrderService, bills *service.BillService, gw pay.GatewayRegistry) *Customer {
	return &Customer{cfg: cfg, db: db, svc: svc, bills: bills, gw: gw}
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
		phDTO = append(phDTO, gin.H{"id": p.ID, "name": p.Name, "price": int64(p.Price), "is_required": p.IsRequired})
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

// Prepay POST /api/pay/prepay {order_id, channel} 预下单（mock 网关）
func (h *Customer) Prepay(c *gin.Context) {
	cid := c.GetInt64("cid")
	var req struct {
		OrderID int64  `json:"order_id"`
		Channel string `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
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

	var shop model.Shop
	subject := "悦点点餐"
	if err := h.db.First(&shop, h.shopID()).Error; err == nil {
		subject = shop.Name
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
