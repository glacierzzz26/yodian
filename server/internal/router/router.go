// Package router Gin 路由。前缀统一 /api（业务路径以 7.4 接口表为准，
// 16.2 示例中的双 admin 前缀为文档笔误，按契约 #5 归一为单 /api/admin）。
// 权限矩阵（16.2）在阶段 2.4 落地：cashier|owner 收银组 / owner 店长组 / kitchen|owner 后厨 WS。
package router

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/yodian/landing" // 3.2 中间页 H5（go:embed 进二进制）
	"github.com/yodian/server/internal/config"
	"github.com/yodian/server/internal/handler"
	"github.com/yodian/server/internal/middleware"
	"github.com/yodian/server/internal/pay"
	"github.com/yodian/server/internal/pkg/respond"
	"github.com/yodian/server/internal/service"
	"github.com/yodian/server/internal/ws"
)

func New(cfg *config.Config, db *gorm.DB, rdb *redis.Client, gw pay.GatewayRegistry) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 阶段 3.2：中间页 H5 一码双扫（4.3 兜底页）。挂根路径 /t（非 /api），QR_BASE_URL 即指向此页。
	landingHTML := func(c *gin.Context) { c.Data(200, "text/html; charset=utf-8", landing.IndexHTML()) }
	r.GET("/t", landingHTML)
	r.GET("/t/", landingHTML)

	api := r.Group("/api")
	api.Use(middleware.RateLimit(rdb, "global", time.Minute, 120)) // 16.3 全局兜底 IP 120/min
	{
		api.GET("/health", health(cfg, db, rdb))

		// 阶段 2.2：顾客端（登录/菜单/下单/预支付/mock 回调）
		custSvc := service.NewOrderService(db, rdb, cfg.ShopID, ws.Default)
		billSvc := service.NewBillService(db, rdb, cfg.ShopID, ws.Default)
		custSvc.SetBillService(billSvc) // 回调单号为账单时委托账单入账
		cust := handler.NewCustomer(cfg, db, rdb, custSvc, billSvc, gw)
		api.POST("/auth/login", middleware.RateLimit(rdb, "cust_login", time.Minute, 5), cust.Login)
		api.GET("/tables/:tid/menu", cust.Menu)

		authed := api.Group("", middleware.CustomerAuth(cfg))
		authed.POST("/orders", middleware.RateLimitBy(rdb, "orders", time.Minute, 10,
			func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("cid"), 10) }), cust.CreateOrder)
		authed.POST("/pay/prepay", middleware.RateLimitBy(rdb, "prepay", time.Minute, 10,
			func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("cid"), 10) }), cust.Prepay)

		if cfg.MockWechatPay || cfg.MockAlipay {
			api.POST("/pay/notify/mock", cust.MockNotify)
		}
		if cfg.Env == "dev" {
			api.GET("/dev/table-sig/:tid", cust.TableSig)
		}

		// 阶段 2.3：顾客结账（生成结账单）
		authed.POST("/sessions/:sid/bill", cust.CreateBill)

		// 阶段 3.1：顾客端契约缺口补齐（7.4 接口表；全走顾客 JWT）
		authed.GET("/sessions/:sid/orders", cust.SessionOrders)
		authed.PATCH("/sessions/:sid/pax", cust.UpdatePax)
		authed.GET("/sessions/:sid/bill/preview", cust.BillPreview)
		authed.POST("/sessions/:sid/bill/cancel", cust.CancelBill) // 契约补充：3.7 取消结账解锁会话
		authed.POST("/pay/switch", middleware.RateLimitBy(rdb, "switch", time.Minute, 10,
			func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("cid"), 10) }), cust.SwitchChannel)
		authed.GET("/orders/:id/status", cust.OrderStatus)
		authed.POST("/auth/phone-code", middleware.RateLimitBy(rdb, "sms_code", time.Minute, 5,
			func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("cid"), 10) }), cust.PhoneCode)
		authed.POST("/auth/phone-bind", cust.PhoneBind)
		authed.POST("/sessions/:sid/call-waiter", middleware.RateLimitBy(rdb, "call_waiter", 10*time.Second, 1,
			func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("cid"), 10) }), cust.CallWaiter)

		// 阶段 2.3/2.4：员工登录（16.3 登录限流 IP 5/min，防爆破）
		staffSvc := service.NewStaffService(db, cfg.ShopID)
		refundSvc := service.NewRefundService(db, gw, cfg.ShopID)
		staff := handler.NewStaff(cfg, db, staffSvc, billSvc, refundSvc)
		api.POST("/auth/staff/login", middleware.RateLimit(rdb, "staff_login", time.Minute, 5), staff.Login)

		// 阶段 2.4：商家/管理后台（单 /api/admin 前缀，按角色分流，越权 40003）
		opsSvc := service.NewOpsService(db, cfg.ShopID)
		admin := handler.NewAdmin(cfg, db, custSvc, billSvc, opsSvc)

		// 收银组：cashier|owner —— 清台 / 前台收银入账 / 改状态 / 补录人工单 / 代客收款 + 核心读端点
		cashier := api.Group("/admin", middleware.StaffAuth(cfg), middleware.RequireRole("cashier", "owner"),
			middleware.RateLimitBy(rdb, "admin", time.Minute, 60,
				func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("oid"), 10) }))
		cashier.POST("/sessions/:sid/close", staff.CloseSession)
		cashier.POST("/sessions/:sid/bill/pay", admin.BillPay)
		cashier.POST("/orders/:id/status", admin.ChangeOrderStatus)
		cashier.POST("/orders/offline", admin.CreateOfflineOrder)
		cashier.POST("/orders/:id/cashier-pay", admin.CashierPay)
		cashier.GET("/tables", admin.ListTables)
		cashier.GET("/orders", admin.ListOrders)
		cashier.GET("/dishes", admin.ListDishes)

		// 后厨组：kitchen|owner —— KDS 读 + KDS 卡片流转（7.4 §1982 kitchen.POST /kds/:id/move；
		// 通用改状态 /orders/:id/status 仍归收银台 cashier|owner）
		kitchen := api.Group("/admin", middleware.StaffAuth(cfg), middleware.RequireRole("kitchen", "owner"),
			middleware.RateLimitBy(rdb, "admin", time.Minute, 60,
				func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("oid"), 10) }))
		kitchen.GET("/kitchen/orders", admin.ListKitchenOrders)
		kitchen.POST("/kds/:id/move", admin.KdsMove)

		// 店长组：owner —— 核销 / 沽清 / 对账 / 日志 / 退款列表
		owner := api.Group("/admin", middleware.StaffAuth(cfg), middleware.RequireRole("owner"),
			middleware.RateLimitBy(rdb, "admin", time.Minute, 60,
				func(c *gin.Context) string { return strconv.FormatInt(c.GetInt64("oid"), 10) }))
		owner.POST("/sessions/:sid/writeoff", staff.WriteOff)
		owner.POST("/dishes/:id/soldout", admin.SoldOut)
		owner.GET("/recon", admin.Recon)
		owner.GET("/logs", admin.Logs)
		owner.GET("/refunds", admin.ListRefunds)

		// 退款：owner（16.2 店长权限，路径保持 /orders/:id/refund）
		ownerOnly := api.Group("", middleware.StaffAuth(cfg), middleware.RequireRole("owner"))
		ownerOnly.POST("/orders/:id/refund", staff.RefundOrder)

		// 阶段 2.4：后厨实时推送（16.4 WS 鉴权，kitchen|owner）
		api.GET("/ws/kitchen", ws.Default.HandleKitchen(cfg))
	}

	r.NoRoute(func(c *gin.Context) { respond.NotFound(c) })
	return r
}

func health(cfg *config.Config, db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbOK := "ok"
		if sqlDB, err := db.DB(); err != nil {
			dbOK = "error: " + err.Error()
		} else if err := sqlDB.Ping(); err != nil {
			dbOK = "error: " + err.Error()
		}

		redisOK := "ok"
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			redisOK = "error: " + err.Error()
		}

		status := "ok"
		if dbOK != "ok" || redisOK != "ok" {
			status = "degraded"
		}
		slog.Info("health check", "db", dbOK, "redis", redisOK)
		respond.OK(c, gin.H{
			"status": status,
			"db":     dbOK,
			"redis":  redisOK,
			"env":    cfg.Env,
		})
	}
}
