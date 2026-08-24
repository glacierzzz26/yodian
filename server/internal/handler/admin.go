// 商家端 / 管理后台 HTTP 处理（阶段 2.4）。
// 路由前缀统一 /api/admin（7.4 接口表；16.2 示例的双 admin 前缀为笔误，按契约 #5 归一）。
// 权限矩阵（16.2）：cashier|owner = 清台/前台收银入账/改状态/补录；owner = 核销/退款/沽清/对账/日志。
// 前端隐藏仅为体验，后端中间件是最终防线 → 越权 40003。
package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yodian/server/internal/config"
	"github.com/yodian/server/internal/middleware"
	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pkg/respond"
	"github.com/yodian/server/internal/service"
)

type Admin struct {
	cfg    *config.Config
	db     *gorm.DB
	orders *service.OrderService
	bills  *service.BillService
	ops    *service.OpsService
}

func NewAdmin(cfg *config.Config, db *gorm.DB, orders *service.OrderService, bills *service.BillService, ops *service.OpsService) *Admin {
	return &Admin{cfg: cfg, db: db, orders: orders, bills: bills, ops: ops}
}

func (h *Admin) shopID() int64 { return h.cfg.ShopID }

// CreateOfflineOrder POST /admin/orders/offline 补录人工单（降级/代客场景，source=offline）
func (h *Admin) CreateOfflineOrder(c *gin.Context) {
	var req service.CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	resp, bizErr := h.orders.CreateOfflineOrder(c.Request.Context(), c.GetInt64("oid"), req)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	middleware.RecordAudit(c, h.db, h.shopID(), "order.create_offline", "order", resp.OrderID,
		gin.H{"tid": req.TID, "amount_cents": resp.TotalAmount})
	respond.OK(c, resp)
}

// ChangeOrderStatus POST /admin/orders/:id/status {to} 改状态（状态机 7.2 校验）
func (h *Admin) ChangeOrderStatus(c *gin.Context) {
	id := parseIDParam(c, "id")
	if id == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("订单参数非法"))
		return
	}
	var req struct {
		To string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.To == "" {
		respond.Err(c, respond.ErrBadRequest.WithMsg("缺少目标状态"))
		return
	}
	if bizErr := h.orders.ChangeOrderStatus(c.Request.Context(), id, c.GetInt64("oid"), req.To); bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	middleware.RecordAudit(c, h.db, h.shopID(), "order.status_change", "order", id, gin.H{"to": req.To})
	respond.OK(c, gin.H{"order_id": id, "status": req.To})
}

// BillPay POST /admin/sessions/:sid/bill/pay {channel: cash|pos|scan} 前台收银入账 / 代客结账
func (h *Admin) BillPay(c *gin.Context) {
	sid := parseIDParam(c, "sid")
	if sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	var req struct {
		Channel string `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	bill, bizErr := h.bills.BillPay(c.Request.Context(), sid, c.GetInt64("oid"), req.Channel)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	middleware.RecordAudit(c, h.db, h.shopID(), "bill.pay", "session", sid,
		gin.H{"channel": req.Channel, "bill_id": bill.ID, "amount_cents": int64(bill.PayableAmount)})
	respond.OK(c, gin.H{"bill_id": bill.ID, "session_id": sid, "channel": req.Channel, "amount_cents": int64(bill.PayableAmount), "status": "paid"})
}

// SoldOut POST /admin/dishes/:id/soldout {sold_out} 标记沽清 / 恢复
func (h *Admin) SoldOut(c *gin.Context) {
	id := parseIDParam(c, "id")
	if id == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("菜品参数非法"))
		return
	}
	var req struct {
		SoldOut bool `json:"sold_out"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	if bizErr := h.ops.SetSoldOut(c.Request.Context(), id, req.SoldOut); bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	middleware.RecordAudit(c, h.db, h.shopID(), "dish.soldout", "dish", id, gin.H{"sold_out": req.SoldOut})
	respond.OK(c, gin.H{"dish_id": id, "sold_out": req.SoldOut})
}

// Recon GET /admin/recon 六口径对账快照（5.3）
func (h *Admin) Recon(c *gin.Context) {
	snap, bizErr := h.ops.Recon(c.Request.Context())
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	respond.OK(c, snap)
}

// Logs GET /admin/logs 操作日志分页（16.2 追责 / 10.1 交接班）
func (h *Admin) Logs(c *gin.Context) {
	q := h.db.Model(&model.AuditLog{}).Where("shop_id = ?", h.shopID())
	if action := c.Query("action"); action != "" {
		q = q.Where("action = ?", action)
	}
	if op := c.Query("operator_id"); op != "" {
		if oid, err := strconv.ParseInt(op, 10, 64); err == nil {
			q = q.Where("operator_id = ?", oid)
		}
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	var total int64
	q.Count(&total)

	var logs []model.AuditLog
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		slog.Error("query audit logs failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}
	respond.OK(c, gin.H{"total": total, "items": logs})
}
