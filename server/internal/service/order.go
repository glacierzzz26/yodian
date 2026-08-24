package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pay"
	"github.com/yodian/server/internal/pkg/respond"
	"github.com/yodian/server/internal/ws"
)

// OrderService 下单 + 支付处理（核心交易）。shopID 为单租户店铺常量（部署即一家店）。
type OrderService struct {
	db     *gorm.DB
	rdb    *redis.Client
	shopID int64
	bills  *BillService // 2.3 挂接：回调单号为账单时委托账单入账
	hub    *ws.Hub      // 2.4 挂接：新单/支付/状态变更实时推送后厨
}

func NewOrderService(db *gorm.DB, rdb *redis.Client, shopID int64, hub *ws.Hub) *OrderService {
	return &OrderService{db: db, rdb: rdb, shopID: shopID, hub: hub}
}

// SetBillService 挂接账单服务：回调单号查无订单时按「账单支付」委托（阶段 2.3）
func (s *OrderService) SetBillService(b *BillService) { s.bills = b }

// push 事件广播（hub 未挂接时静默跳过）
func (s *OrderService) push(event string, payload any) {
	if s.hub != nil {
		s.hub.Broadcast(event, payload)
	}
}

// CreateOrderReq 创建订单（16.1 契约）
type CreateOrderReq struct {
	TID   int64           `json:"tid"`
	Pax   int             `json:"pax"`
	Items []OrderItemReq  `json:"items"`
}

type OrderItemReq struct {
	DishID int64  `json:"dish_id"`
	Qty    int    `json:"qty"`
	Specs  any    `json:"specs,omitempty"`
	Remark string `json:"remark,omitempty"`
}

type CreateOrderResp struct {
	OrderID     int64  `json:"order_id"`
	OutTradeNo  string `json:"out_trade_no"`
	TotalAmount int64  `json:"total_amount"`
	Status      string `json:"status"`
	PayMode     string `json:"pay_mode"`
	SessionID   int64  `json:"session_id"`
}

// CreateOrder 顾客下单：委托内部实现，source=online
func (s *OrderService) CreateOrder(ctx context.Context, customerID int64, req CreateOrderReq) (*CreateOrderResp, *respond.BizErr) {
	return s.createOrder(ctx, &customerID, nil, "online", req)
}

// CreateOfflineOrder 补录人工单（7.4 /admin/orders/offline）：代客点单，无顾客、记操作员
func (s *OrderService) CreateOfflineOrder(ctx context.Context, operatorID int64, req CreateOrderReq) (*CreateOrderResp, *respond.BizErr) {
	return s.createOrder(ctx, nil, &operatorID, "cashier", req)
}

// createOrder 创建订单：服务端重算金额（3.3）、沽清二次校验（9.6）、
// 桌台会话开台/快照（3.4/3.7）、prepay 首单按人项计入（3.8）
func (s *OrderService) createOrder(ctx context.Context, customerID, operatorID *int64, source string, req CreateOrderReq) (*CreateOrderResp, *respond.BizErr) {
	if len(req.Items) == 0 {
		return nil, respond.ErrBadRequest.WithMsg("请选择菜品")
	}
	if req.Pax < 1 || req.Pax > 20 {
		req.Pax = 2 // 未传时按 2 人（1–20 上限由顾客端约束，服务端兜底）
	}

	// 1. 桌台 + 签名由 handler 层校验，这里查桌台
	var t model.Table
	if err := s.db.First(&t, "shop_id = ? AND id = ?", s.shopID, req.TID).Error; err != nil {
		return nil, respond.ErrTableNotFound
	}

	// 2. 会话：查 active，无则开台（快照 pay_mode，3.7）
	sess, bizErr := s.ensureSession(ctx, &t, req.Pax)
	if bizErr != nil {
		return nil, bizErr
	}

	// 3. 菜品校验 + 服务端算金额
	var items []model.OrderItem
	var total model.Money
	for _, it := range req.Items {
		if it.Qty <= 0 {
			return nil, respond.ErrBadRequest.WithMsg("菜品数量非法")
		}
		var d model.Dish
		if err := s.db.First(&d, "shop_id = ? AND id = ?", s.shopID, it.DishID).Error; err != nil {
			return nil, respond.ErrBadRequest.WithMsg("菜品不存在: id=%d", it.DishID)
		}
		if d.IsSoldOut {
			return nil, respond.ErrDishSoldOut.WithMsg("「%s」已沽清，请刷新菜单", d.Name)
		}
		var specs []byte
		if it.Specs != nil {
			specs, _ = json.Marshal(it.Specs)
		}
		total += d.Price * model.Money(it.Qty)
		items = append(items, model.OrderItem{
			DishID: &d.ID, ItemType: "dish", Name: d.Name, Price: d.Price,
			Qty: it.Qty, Specs: specs, Remark: nullableString(it.Remark),
		})
	}

	// 4. prepay 会话首单：按人项一次性计入（3.8，去重）
	first := s.isFirstOrderOfSession(ctx, sess.ID)
	if t.PayMode == "prepay" && first {
		charges, bizErr := s.loadPerHead(ctx)
		if bizErr != nil {
			return nil, bizErr
		}
		for _, c := range charges {
			total += c.Price * model.Money(sess.Pax)
			items = append(items, model.OrderItem{
				ItemType: "per_head", Name: c.Name, Price: c.Price, Qty: sess.Pax,
			})
		}
	}

	// 5. 事务创建订单（生成 out_trade_no）
	outTradeNo := genOutTradeNo()
	payMode := t.PayMode
	order := model.Order{
		ShopID: s.shopID, TableID: &t.ID, SessionID: &sess.ID, CustomerID: customerID, OperatorID: operatorID,
		PayMode: payMode, OrderType: "dine_in", Source: source,
		Status: InitialStatus(payMode), TotalAmount: total,
		OutTradeNo: &outTradeNo,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].OrderID = order.ID
		}
		return tx.Create(&items).Error
	}); err != nil {
		slog.Error("create order tx failed", "err", err)
		return nil, respond.ErrInternal
	}

	slog.Info("order created", "order_id", order.ID, "pay_mode", payMode,
		"status", order.Status, "total_cents", int64(total))
	s.push("order.new", map[string]any{
		"order_id": order.ID, "table_id": t.ID, "session_id": sess.ID,
		"status": order.Status, "pay_mode": payMode, "total_cents": int64(total),
	})
	return &CreateOrderResp{
		OrderID: order.ID, OutTradeNo: outTradeNo,
		TotalAmount: int64(total), Status: order.Status,
		PayMode: payMode, SessionID: sess.ID,
	}, nil
}

// ensureSession 查找该桌 active 会话；无则开台并置桌台 occupied（3.5 派生字段同事务）。
// 若存在 settled 会话（账单已付未清台）则拒绝开新台，须先清台（3.7：settled 后不再接受新单）。
func (s *OrderService) ensureSession(ctx context.Context, t *model.Table, pax int) (*model.TableSession, *respond.BizErr) {
	var sess model.TableSession
	err := s.db.Where("table_id = ? AND status IN ('active','settled')", t.ID).First(&sess).Error
	if err == nil {
		if sess.Status == "settled" {
			return nil, respond.ErrSessionSettled
		}
		if sess.LockedForBill {
			return nil, respond.ErrSessionLocked
		}
		if sess.Pax != pax {
			sess.Pax = pax
			s.db.Model(&sess).Update("pax", pax)
		}
		return &sess, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, respond.ErrInternal
	}

	sess = model.TableSession{ShopID: s.shopID, TableID: t.ID, PayMode: t.PayMode, Pax: pax}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&sess).Error; err != nil {
			return err
		}
		return tx.Model(&model.Table{}).Where("id = ?", t.ID).Update("status", "occupied").Error
	}); err != nil {
		return nil, respond.ErrInternal
	}
	return &sess, nil
}

func (s *OrderService) isFirstOrderOfSession(ctx context.Context, sessionID int64) bool {
	var cnt int64
	s.db.Model(&model.Order{}).Where("session_id = ?", sessionID).Count(&cnt)
	return cnt == 0
}

func (s *OrderService) loadPerHead(ctx context.Context) ([]model.PerHeadCharge, *respond.BizErr) {
	var out []model.PerHeadCharge
	if err := s.db.Where("shop_id = ? AND is_active = true", s.shopID).Order("sort").Find(&out).Error; err != nil {
		return nil, respond.ErrInternal
	}
	return out, nil
}

// HandlePaidSuccess 支付成功核心处理（3.3）：幂等三重保障 + 金额校验 + 事务入账 + 出单
func (s *OrderService) HandlePaidSuccess(ctx context.Context, res pay.NotifyResult) error {
	lockKey := "pay:lock:" + res.OutTradeNo
	ok, err := s.rdb.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
	if err != nil {
		return fmt.Errorf("acquire pay lock: %w", err)
	}
	if !ok {
		// 另一回调处理中：等待后由唯一索引兜底（不阻塞双渠道）
		slog.Warn("pay lock contended, defer to unique index", "out_trade_no", res.OutTradeNo)
		return nil
	}
	defer s.rdb.Del(ctx, lockKey)

	// 幂等出口：该 out_trade_no 已有一笔成功支付 → 直接返回
	var existed model.Payment
	if err := s.db.Where("out_trade_no = ?", res.OutTradeNo).First(&existed).Error; err == nil && existed.Status == "success" {
		slog.Info("idempotent: payment already success", "out_trade_no", res.OutTradeNo)
		return nil
	}

	// 主体：先付 → orders；后付/前台结 → bills（锁已持有，委托时不再重复加锁）
	var o model.Order
	if err := s.db.Where("out_trade_no = ?", res.OutTradeNo).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if s.bills == nil {
				return fmt.Errorf("bill service not wired: %s", res.OutTradeNo)
			}
			return s.bills.HandleBillPaidLocked(ctx, res)
		}
		return fmt.Errorf("query order: %w", err)
	}
	if o.Status == "paid" || o.Status == "done" {
		slog.Info("idempotent: order already paid", "order_id", o.ID)
		return nil
	}

	// 金额校验（防篡改少付，3.3）：回调金额必须等于服务端金额
	if res.PaidAmount != o.TotalAmount {
		slog.Error("amount mismatch, mark manual", "order_id", o.ID,
			"expect_cents", int64(o.TotalAmount), "got_cents", int64(res.PaidAmount))
		s.db.Model(&o).Update("status", "manual")
		return respond.ErrPayAmountMismatch
	}

	// 事务：订单置已支付 + 插入支付成功记录（含回调原文）
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Order{}).Where("id = ?", o.ID).
			Updates(map[string]any{
				"status": "paid", "paid_at": time.Now(),
				"pay_channel": string(res.Channel), "paid_amount": o.TotalAmount,
			}).Error; err != nil {
			return err
		}
		pm := model.Payment{
			OrderID: &o.ID, OutTradeNo: res.OutTradeNo, Channel: string(res.Channel),
			ChannelTradeNo: nullableString(res.ChannelTradeNo),
			Amount: res.PaidAmount, Status: "success", CallbackRaw: res.RawPayload,
		}
		return tx.Create(&pm).Error
	})
	if err != nil {
		if isUniqueViolation(err) {
			// 核心防线：该订单已有一笔成功支付（6.10 兜底）→ 标记需人工 + 告警
			slog.Error("duplicate success payment, mark manual", "out_trade_no", res.OutTradeNo)
			s.db.Model(&model.Order{}).Where("id = ?", o.ID).Update("status", "manual")
			return respond.ErrDuplicatePay
		}
		return fmt.Errorf("paid tx: %w", err)
	}

	slog.Info("payment success", "order_id", o.ID, "out_trade_no", res.OutTradeNo, "channel", res.Channel)
	// 事务外：推打印任务 + WS 通知后厨（3.6 出单）
	s.enqueuePrint(ctx, o.ID, nil)
	s.push("order.paid", map[string]any{
		"order_id": o.ID, "table_id": o.TableID, "session_id": o.SessionID,
		"amount_cents": int64(o.TotalAmount), "channel": res.Channel,
	})
	return nil
}

// ChangeOrderStatus 商家端改订单状态（7.4 POST /orders/:id/status，cashier|owner 权限在路由层）
func (s *OrderService) ChangeOrderStatus(ctx context.Context, orderID, operatorID int64, to string) *respond.BizErr {
	var o model.Order
	if err := s.db.First(&o, "shop_id = ? AND id = ?", s.shopID, orderID).Error; err != nil {
		return respond.NewBiz(20012, "订单不存在", 200)
	}
	if bizErr := AssertTransition(o.PayMode, o.Status, to); bizErr != nil {
		return bizErr
	}
	updates := map[string]any{"status": to}
	if to == "done" {
		updates["finished_at"] = time.Now()
	}
	if err := s.db.Model(&o).Updates(updates).Error; err != nil {
		slog.Error("change order status failed", "err", err, "order_id", o.ID)
		return respond.ErrInternal
	}
	s.push("order.status", map[string]any{"order_id": o.ID, "from": o.Status, "to": to, "operator": operatorID})
	return nil
}


// enqueuePrint 出单任务落 print_tasks（MOCK_PRINTER=true 时仅记录）
func (s *OrderService) enqueuePrint(ctx context.Context, orderID int64, billID *int64) {
	pt := model.PrintTask{OrderID: &orderID, PrinterSN: "MOCK", Status: "pending"}
	if billID != nil {
		pt.BillID = billID
		pt.OrderID = nil
	}
	if err := s.db.Create(&pt).Error; err != nil {
		slog.Error("enqueue print failed", "err", err)
	}
}

// genOutTradeNo 生成全局唯一支付单号
func genOutTradeNo() string { return genNo("OD") }

// genNo 前缀 + 时间戳 + 8 位随机（OD 支付单 / RF 退款单 / BL 账单，全库唯一）
func genNo(prefix string) string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s%s%s", prefix, time.Now().Format("20060102150405"), hex.EncodeToString(buf)[:8])
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
