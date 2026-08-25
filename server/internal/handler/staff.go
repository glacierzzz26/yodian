// 员工端 HTTP 处理（阶段 2.3：员工登录 / 清台 / 核销 / 退款）。
// 完整权限矩阵（角色→路由）在阶段 2.4 落地；此处仅按业务需要挂 RequireRole。
package handler

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yodian/server/internal/auth"
	"github.com/yodian/server/internal/config"
	"github.com/yodian/server/internal/middleware"
	"github.com/yodian/server/internal/pkg/respond"
	"github.com/yodian/server/internal/service"
)

type Staff struct {
	cfg      *config.Config
	db       *gorm.DB
	staffSvc *service.StaffService
	bills    *service.BillService
	refunds  *service.RefundService
}

func NewStaff(cfg *config.Config, db *gorm.DB, staffSvc *service.StaffService, bills *service.BillService, refunds *service.RefundService) *Staff {
	return &Staff{cfg: cfg, db: db, staffSvc: staffSvc, bills: bills, refunds: refunds}
}

func (h *Staff) shopID() int64 { return h.cfg.ShopID }

// Login POST /api/auth/staff/login {employee_no, password}
func (h *Staff) Login(c *gin.Context) {
	var req struct {
		EmployeeNo string `json:"employee_no"`
		Password   string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	emp, bizErr := h.staffSvc.Authenticate(c.Request.Context(), req.EmployeeNo, req.Password)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	token, err := auth.SignStaff(h.cfg.JWTStaffSecret, emp.ID, h.shopID(), emp.Role, "access",
		time.Duration(h.cfg.JWTStaffAccessH)*time.Hour)
	if err != nil {
		slog.Error("sign staff token failed", "err", err)
		respond.Err(c, respond.ErrInternal)
		return
	}
	// 阶段 4.1 联调：补 id，前端 Operator.id 需要（后续管理 API 以 operator_id 关联）
	respond.OK(c, gin.H{"token": token, "id": emp.ID, "name": emp.Name, "employee_no": emp.EmployeeNo, "role": emp.Role})
}

// CloseSession POST /api/sessions/:sid/close 清台（未结账被拒）
func (h *Staff) CloseSession(c *gin.Context) {
	sid := parseIDParam(c, "sid")
	if sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	if bizErr := h.bills.CloseSession(c.Request.Context(), sid, c.GetInt64("oid"), "close"); bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	middleware.RecordAudit(c, h.db, h.shopID(), "session.close", "session", sid, gin.H{"action": "close"})
	respond.OK(c, gin.H{"session_id": sid, "action": "close"})
}

// WriteOff POST /api/sessions/:sid/write-off 店长核销（owner 角色，强制关台）
func (h *Staff) WriteOff(c *gin.Context) {
	sid := parseIDParam(c, "sid")
	if sid == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("会话参数非法"))
		return
	}
	if bizErr := h.bills.CloseSession(c.Request.Context(), sid, c.GetInt64("oid"), "write_off"); bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	middleware.RecordAudit(c, h.db, h.shopID(), "session.write_off", "session", sid, gin.H{"action": "write_off"})
	respond.OK(c, gin.H{"session_id": sid, "action": "write_off"})
}

// RefundOrder POST /api/orders/:id/refund {amount?, reason} 全退/部分退
func (h *Staff) RefundOrder(c *gin.Context) {
	id := parseIDParam(c, "id")
	if id == 0 {
		respond.Err(c, respond.ErrBadRequest.WithMsg("订单参数非法"))
		return
	}
	var req service.RefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Err(c, respond.ErrBadRequest)
		return
	}
	rf, bizErr := h.refunds.RefundOrder(c.Request.Context(), id, c.GetInt64("oid"), req)
	if bizErr != nil {
		respond.Err(c, bizErr)
		return
	}
	middleware.RecordAudit(c, h.db, h.shopID(), "order.refund", "order", id,
		gin.H{"refund_id": rf.ID, "amount_cents": int64(rf.Amount)})
	respond.OK(c, gin.H{"refund_id": rf.ID, "out_refund_no": rf.OutRefundNo, "amount": int64(rf.Amount), "status": rf.Status})
}

func parseIDParam(c *gin.Context, name string) int64 {
	v, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
