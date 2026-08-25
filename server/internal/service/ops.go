// 运营支撑：沽清开关 + 六口径对账 + 商家端读端点（设计 5.3 / 7.4，阶段 4.1 联调）。
package service

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/yodian/server/internal/model"
	"github.com/yodian/server/internal/pkg/respond"
)

type OpsService struct {
	db     *gorm.DB
	shopID int64
}

func NewOpsService(db *gorm.DB, shopID int64) *OpsService {
	return &OpsService{db: db, shopID: shopID}
}

// SetSoldOut 标记沽清/恢复（7.4 POST /admin/dishes/:id/soldout）
func (s *OpsService) SetSoldOut(ctx context.Context, dishID int64, soldOut bool) *respond.BizErr {
	res := s.db.Model(&model.Dish{}).Where("shop_id = ? AND id = ?", s.shopID, dishID).
		Update("is_sold_out", soldOut)
	if res.Error != nil {
		slog.Error("set soldout failed", "err", res.Error)
		return respond.ErrInternal
	}
	if res.RowsAffected == 0 {
		return respond.NewBiz(20016, "菜品不存在", 200)
	}
	return nil
}

// ReconSnapshot 六口径对账快照（金额单位：分，与 API 其它金额一致）。
// 口径（5.3）：payments 成功流水按渠道（微信/支付宝/现金/POS/扫）= 六口径合计；
// 应收 = 订单直付（prepay 单笔支付）+ 已支付账单（后付/前台结，含按人项）；
// 挂账/核销单独列示、不混入资金流。diff = 应收合计 - 支付实收，0 为平。
type ReconSnapshot struct {
	OrderReceivable int64            `json:"order_receivable"` // 订单直付应收（prepay，分）
	BillReceivable  int64            `json:"bill_receivable"`  // 已支付账单应收（含按人项，分）
	TotalReceivable int64            `json:"total_receivable"` // 应收合计 = 订单 + 账单
	PaymentPaid     int64            `json:"payment_paid"`     // 六口径合计 = 成功支付（分）
	PaymentByChan   map[string]int64 `json:"payment_by_channel"`
	Refunded        int64            `json:"refunded"`      // 退款总额（分）
	NetReceipt      int64            `json:"net_receipt"`   // 实收净额 = 支付 - 退款
	UnsettledAmount int64            `json:"unsettled_amount"` // 挂账额：已出餐未收款（分，单独列示）
	UnsettledOrders int64            `json:"unsettled_orders"`
	PendingBills    int64            `json:"pending_bills"`
	WrittenOffAmount int64           `json:"written_off_amount"` // 核销额：订单侧损耗（分）
	ManualOrders    int64            `json:"manual_orders"`
	OrderCount      int64            `json:"order_count"`
	OrderByStatus   map[string]int64 `json:"order_by_status"`
	Diff            int64            `json:"diff"` // total_receivable - payment_paid，0 为平
}

// Recon 生成六口径对账快照。注：DB 金额列以元存储（numeric），SQL 内 ×100 折分。
func (s *OpsService) Recon(ctx context.Context) (*ReconSnapshot, *respond.BizErr) {
	snap := &ReconSnapshot{
		OrderByStatus: map[string]int64{},
		PaymentByChan: map[string]int64{},
	}

	// ① 订单状态拆分（展示 + 挂账/核销单独列示）
	var rows []struct {
		Status string
		Sum    int64
		Count  int64
	}
	if err := s.db.Raw(`SELECT status, (COALESCE(SUM(total_amount),0)*100)::bigint AS sum,
			COUNT(*)::bigint AS count FROM orders WHERE shop_id = ? GROUP BY status`,
		s.shopID).Scan(&rows).Error; err != nil {
		slog.Error("recon orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	for _, r := range rows {
		snap.OrderByStatus[r.Status] = r.Sum
		snap.OrderCount += r.Count
		switch r.Status {
		case "unsettled":
			snap.UnsettledAmount += r.Sum
			snap.UnsettledOrders += r.Count
		case "written_off":
			snap.WrittenOffAmount += r.Sum
		case "manual":
			snap.ManualOrders += r.Count
		}
	}

	// ② 订单直付应收：prepay 单笔支付（未挂账单的已支付订单）
	if err := s.db.Raw(`SELECT (COALESCE(SUM(total_amount),0)*100)::bigint FROM orders
			WHERE shop_id = ? AND bill_id IS NULL
			  AND status IN ('paid','preparing','served','done','partially_refunded')`, s.shopID).
		Scan(&snap.OrderReceivable).Error; err != nil {
		slog.Error("recon direct orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	// ③ 已支付账单应收（后付/前台结，含按人项；closed = 已支付后的清台归档，同为已收）
	if err := s.db.Raw(`SELECT (COALESCE(SUM(payable_amount),0)*100)::bigint FROM bills
			WHERE shop_id = ? AND status IN ('paid','closed')`, s.shopID).Scan(&snap.BillReceivable).Error; err != nil {
		slog.Error("recon paid bills failed", "err", err)
		return nil, respond.ErrInternal
	}
	// ④ 六口径资金流：payments 成功流水按渠道（微信/支付宝/现金/POS/扫）
	var chans []struct {
		Channel string
		Sum     int64
	}
	if err := s.db.Raw(`SELECT channel, (COALESCE(SUM(amount),0)*100)::bigint AS sum
			FROM payments WHERE status = 'success' GROUP BY channel`).Scan(&chans).Error; err != nil {
		slog.Error("recon payments failed", "err", err)
		return nil, respond.ErrInternal
	}
	for _, r := range chans {
		snap.PaymentByChan[r.Channel] = r.Sum
		snap.PaymentPaid += r.Sum
	}
	// ⑤ 退款总额
	if err := s.db.Raw(`SELECT (COALESCE(SUM(amount),0)*100)::bigint
			FROM refunds WHERE status IN ('success','pending')`).Scan(&snap.Refunded).Error; err != nil {
		slog.Error("recon refunds failed", "err", err)
		return nil, respond.ErrInternal
	}
	// ⑥ 待支付账单（挂账提醒）
	if err := s.db.Raw(`SELECT COUNT(*)::bigint FROM bills
			WHERE shop_id = ? AND status = 'pending'`, s.shopID).Scan(&snap.PendingBills).Error; err != nil {
		slog.Error("recon pending bills failed", "err", err)
		return nil, respond.ErrInternal
	}

	snap.TotalReceivable = snap.OrderReceivable + snap.BillReceivable
	snap.NetReceipt = snap.PaymentPaid - snap.Refunded
	snap.Diff = snap.TotalReceivable - snap.PaymentPaid // 5.3：应收合计应等于六口径合计
	return snap, nil
}

/* ---------------- 阶段 4.1 联调：商家端读端点（金额一律分，7.4 接口表） ----------------
   诚实取舍：并桌 merged 恒 false（后端无并桌概念）；档口 station / 单位 unit / 按人项
   item_type 后端菜品表无字段，统一给默认值（station:"", unit:"份", item_type:"dish"），
   联调标注这些展示字段待后端管理 API 补齐。 */

// TableSessionSummary 桌台活动/待清台会话摘要（桌台图）
type TableSessionSummary struct {
	SessionID      int64 `json:"session_id"`
	PayMode        string `json:"pay_mode"`
	UnsettledCents int64 `json:"unsettled_cents"` // 会话内未结账子单合计（挂账，分）
	LockedForBill  bool  `json:"locked_for_bill"`
	Merged         bool  `json:"merged"` // 并桌：后端未实现，恒 false
	Pax            int   `json:"pax"`
}

// AdminTableDTO 桌台（联调展示）
type AdminTableDTO struct {
	ID        int64                `json:"id"`
	TableNo   string               `json:"table_no"`
	AreaID    *int64               `json:"area_id"`
	Seats     int                  `json:"seats"`
	PayMode   string               `json:"pay_mode"`
	Status    string               `json:"status"` // empty / occupied
	Session   *TableSessionSummary `json:"session,omitempty"`
	WaitClean bool                 `json:"wait_clean"` // 存在已付未清台会话
}

// AdminTableAreaDTO 桌台区（联调展示）
type AdminTableAreaDTO struct {
	ID             int64           `json:"id"`
	Name           string          `json:"name"`
	ZoneLabel      string          `json:"zone_label"`
	TableLabel     string          `json:"table_label"` // 由桌号字母前缀推导（A/B/C）
	DefaultPayMode *string         `json:"default_pay_mode"`
	Tables         []AdminTableDTO `json:"tables"`
}

// TablesResp GET /admin/tables 响应
type TablesResp struct {
	Areas []AdminTableAreaDTO `json:"areas"`
}

// ListTables GET /admin/tables：桌台图 + 每桌活动/待清台会话摘要（挂账额/锁单/人数）。
func (s *OpsService) ListTables(ctx context.Context) (*TablesResp, *respond.BizErr) {
	resp := &TablesResp{Areas: []AdminTableAreaDTO{}}
	var areas []model.TableArea
	if err := s.db.Where("shop_id = ? AND is_active = true", s.shopID).Order("sort_order, id").Find(&areas).Error; err != nil {
		slog.Error("list areas failed", "err", err)
		return nil, respond.ErrInternal
	}
	var tables []model.Table
	if err := s.db.Where("shop_id = ?", s.shopID).Order("area_id, sort_order, id").Find(&tables).Error; err != nil {
		slog.Error("list tables failed", "err", err)
		return nil, respond.ErrInternal
	}

	// 每桌活动/待清台会话（同一桌最多一个 active 或 settled，见 ensureSession）
	sessions := map[int64]model.TableSession{}
	var sessRows []model.TableSession
	if err := s.db.Where("shop_id = ? AND status IN ('active','settled')", s.shopID).Find(&sessRows).Error; err != nil {
		slog.Error("list sessions failed", "err", err)
		return nil, respond.ErrInternal
	}
	for _, se := range sessRows {
		if _, ok := sessions[se.TableID]; !ok {
			sessions[se.TableID] = se
		}
	}

	// 会话内挂账合计（unsettled 子单）
	unsettled := map[int64]int64{}
	if len(sessions) > 0 {
		ids := make([]int64, 0, len(sessions))
		for _, se := range sessions {
			ids = append(ids, se.ID)
		}
		var rows []struct {
			SessionID int64
			Sum       int64
		}
		if err := s.db.Raw(`SELECT session_id, (COALESCE(SUM(total_amount),0)*100)::bigint AS sum
			FROM orders WHERE session_id IN ? AND status = 'unsettled' GROUP BY session_id`, ids).Scan(&rows).Error; err != nil {
			slog.Error("tables unsettled sum failed", "err", err)
			return nil, respond.ErrInternal
		}
		for _, r := range rows {
			unsettled[r.SessionID] = r.Sum
		}
	}

	byArea := map[int64][]model.Table{}
	for _, t := range tables {
		aid := int64(0)
		if t.AreaID != nil {
			aid = *t.AreaID
		}
		byArea[aid] = append(byArea[aid], t)
	}
	for _, a := range areas {
		ts := byArea[a.ID]
		dto := AdminTableAreaDTO{
			ID: a.ID, Name: a.Name, ZoneLabel: a.Name,
			DefaultPayMode: a.DefaultPayMode, Tables: []AdminTableDTO{},
		}
		if len(ts) > 0 {
			dto.TableLabel = tableLabel(ts[0].TableNo)
		}
		for _, t := range ts {
			row := AdminTableDTO{ID: t.ID, TableNo: t.TableNo, AreaID: t.AreaID,
				Seats: t.Seats, PayMode: t.PayMode, Status: t.Status}
			if se, ok := sessions[t.ID]; ok {
				row.Status = "occupied"
				if se.Status == "settled" {
					row.WaitClean = true
				}
				row.Session = &TableSessionSummary{
					SessionID: se.ID, PayMode: se.PayMode,
					UnsettledCents: unsettled[se.ID],
					LockedForBill:  se.LockedForBill, Merged: false, Pax: se.Pax,
				}
			}
			dto.Tables = append(dto.Tables, row)
		}
		resp.Areas = append(resp.Areas, dto)
	}
	return resp, nil
}

// tableLabel 由桌号字母前缀推导区标签（A01→A，B04→B）
func tableLabel(no string) string {
	for i, r := range no {
		if r < 'A' || r > 'Z' {
			return no[:i]
		}
	}
	return no
}

// AdminOrderItemDTO 订单行（商家端订单/补录列表）
type AdminOrderItemDTO struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Specs      any     `json:"specs,omitempty"`
	Remark     *string `json:"remark,omitempty"`
	Qty        int     `json:"qty"`
	PriceCents int64   `json:"price_cents"`
	ItemType   string  `json:"item_type"` // dish / per_head
	IsServed   bool    `json:"is_served"`
	IsRefunded bool    `json:"is_refunded"`
}

// AdminOrderDTO 订单（商家端订单管理）
type AdminOrderDTO struct {
	ID               int64              `json:"id"`
	No               string             `json:"no"` // out_trade_no
	TableNo          *string            `json:"table_no"`
	Area             *string            `json:"area"`
	PayMode          string             `json:"pay_mode"`
	Source           string             `json:"source"` // online / cashier
	PayChannel       *string            `json:"pay_channel,omitempty"`
	Status           string             `json:"status"`
	TotalAmountCents int64              `json:"total_amount_cents"`
	PaidAmountCents  *int64             `json:"paid_amount_cents,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	Remark           *string            `json:"remark,omitempty"` // 订单无备注字段，恒 null（诚实）
	Items            []AdminOrderItemDTO `json:"items"`
}

// AdminOrdersResp GET /admin/orders 响应
type AdminOrdersResp struct {
	Total int64           `json:"total"`
	Items []AdminOrderDTO `json:"items"`
}

// AdminOrderFilter 订单列表过滤
type AdminOrderFilter struct {
	Status  string
	Source  string
	TableNo string
	Kw      string // 匹配单号 / 桌号
	Limit   int
	Offset  int
}

// ListOrders GET /admin/orders：订单管理列表（join 桌台取桌号/区名，子单批量加载）。
func (s *OpsService) ListOrders(ctx context.Context, f AdminOrderFilter) (*AdminOrdersResp, *respond.BizErr) {
	base := s.db.Table("orders o").
		Joins("LEFT JOIN tables t ON t.id = o.table_id").
		Joins("LEFT JOIN table_areas a ON a.id = t.area_id").
		Where("o.shop_id = ?", s.shopID)
	if f.Status != "" {
		base = base.Where("o.status = ?", f.Status)
	}
	if f.Source != "" {
		base = base.Where("o.source = ?", f.Source)
	}
	if f.TableNo != "" {
		base = base.Where("t.table_no ILIKE ?", "%"+f.TableNo+"%")
	}
	if f.Kw != "" {
		base = base.Where("(o.out_trade_no ILIKE ? OR t.table_no ILIKE ?)", "%"+f.Kw+"%", "%"+f.Kw+"%")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		slog.Error("count orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	var rows []struct {
		model.Order
		TableNo *string
		Area    *string
	}
	if err := base.Select("o.*, t.table_no, a.name AS area").
		Order("o.id DESC").Limit(f.Limit).Offset(f.Offset).Scan(&rows).Error; err != nil {
		slog.Error("list orders failed", "err", err)
		return nil, respond.ErrInternal
	}

	orderIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		orderIDs = append(orderIDs, r.ID)
	}
	itemsByOrder := map[int64][]model.OrderItem{}
	if len(orderIDs) > 0 {
		var its []model.OrderItem
		if err := s.db.Where("order_id IN ?", orderIDs).Order("id").Find(&its).Error; err != nil {
			slog.Error("list order items failed", "err", err)
			return nil, respond.ErrInternal
		}
		for _, it := range its {
			itemsByOrder[it.OrderID] = append(itemsByOrder[it.OrderID], it)
		}
	}

	out := make([]AdminOrderDTO, 0, len(rows))
	for _, r := range rows {
		no := ""
		if r.OutTradeNo != nil {
			no = *r.OutTradeNo
		}
		dto := AdminOrderDTO{
			ID: r.ID, No: no, TableNo: r.TableNo, Area: r.Area,
			PayMode: r.PayMode, Source: r.Source, PayChannel: r.PayChannel,
			Status: r.Status, TotalAmountCents: int64(r.TotalAmount),
			CreatedAt: r.CreatedAt, Items: []AdminOrderItemDTO{},
		}
		if r.PaidAmount != nil {
			v := int64(*r.PaidAmount)
			dto.PaidAmountCents = &v
		}
		for _, it := range itemsByOrder[r.ID] {
			dto.Items = append(dto.Items, AdminOrderItemDTO{
				ID: it.ID, Name: it.Name, Specs: it.Specs, Remark: it.Remark,
				Qty: it.Qty, PriceCents: int64(it.Price), ItemType: it.ItemType,
				IsServed: it.IsServed, IsRefunded: it.IsRefunded,
			})
		}
		out = append(out, dto)
	}
	return &AdminOrdersResp{Total: total, Items: out}, nil
}

// AdminDishDTO 菜品（代客点单/沽清）
type AdminDishDTO struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	CategoryName string `json:"category_name"`
	PriceCents   int64  `json:"price_cents"`
	IsSoldOut    bool   `json:"is_sold_out"`
	Specs        any    `json:"specs,omitempty"`
	Unit         string `json:"unit"`     // 后端无单位字段，默认「份」
	ItemType     string `json:"item_type"` // 后端无按人项区分，默认 dish
	Station      string `json:"station"`   // 后端无档口字段，默认空
}

// AdminDishesResp GET /admin/dishes 响应
type AdminDishesResp struct {
	Items []AdminDishDTO `json:"items"`
}

// ListDishes GET /admin/dishes：菜品列表（join 分类取分类名，沽清/规格随行）。
func (s *OpsService) ListDishes(ctx context.Context) (*AdminDishesResp, *respond.BizErr) {
	var rows []struct {
		model.Dish
		CategoryName string
	}
	if err := s.db.Table("dishes d").
		Joins("LEFT JOIN categories c ON c.id = d.category_id").
		Where("d.shop_id = ?", s.shopID).
		Order("d.sort, d.id").
		Select("d.*, c.name AS category_name").Scan(&rows).Error; err != nil {
		slog.Error("list dishes failed", "err", err)
		return nil, respond.ErrInternal
	}
	out := make([]AdminDishDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, AdminDishDTO{
			ID: r.ID, Name: r.Name, CategoryName: r.CategoryName,
			PriceCents: int64(r.Price), IsSoldOut: r.IsSoldOut, Specs: r.Specs,
			Unit: "份", ItemType: "dish", Station: "",
		})
	}
	return &AdminDishesResp{Items: out}, nil
}

// AdminKitchenItemDTO KDS 后厨行
type AdminKitchenItemDTO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Specs any    `json:"specs,omitempty"`
	Qty   int    `json:"qty"`
}

// AdminKitchenOrderDTO KDS 订单
type AdminKitchenOrderDTO struct {
	ID        int64                  `json:"id"`
	No        string                 `json:"no"`
	TableNo   *string                `json:"table_no"`
	PayMode   string                 `json:"pay_mode"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	Station   string                 `json:"station"` // 后端无档口，空
	Items     []AdminKitchenItemDTO  `json:"items"`
}

// AdminKitchenResp GET /admin/kitchen/orders 响应
type AdminKitchenResp struct {
	Items []AdminKitchenOrderDTO `json:"items"`
}

// ListKitchenOrders GET /admin/kitchen/orders：KDS 待做/制作中订单。
// 状态含 unsettled（后付下单即出单）/ paid（先付已付待做）/ preparing（制作中），
// 过滤掉按人项（不在后厨），按开单顺序展示。
func (s *OpsService) ListKitchenOrders(ctx context.Context) (*AdminKitchenResp, *respond.BizErr) {
	var rows []struct {
		model.Order
		TableNo *string
	}
	if err := s.db.Table("orders o").
		Joins("LEFT JOIN tables t ON t.id = o.table_id").
		Where("o.shop_id = ? AND o.status IN ('unsettled','paid','preparing')", s.shopID).
		Order("o.id").
		Select("o.*, t.table_no").Scan(&rows).Error; err != nil {
		slog.Error("list kitchen orders failed", "err", err)
		return nil, respond.ErrInternal
	}
	orderIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		orderIDs = append(orderIDs, r.ID)
	}
	itemsByOrder := map[int64][]model.OrderItem{}
	if len(orderIDs) > 0 {
		var its []model.OrderItem
		if err := s.db.Where("order_id IN ?", orderIDs).Order("id").Find(&its).Error; err != nil {
			slog.Error("list kitchen items failed", "err", err)
			return nil, respond.ErrInternal
		}
		for _, it := range its {
			if it.ItemType == "per_head" {
				continue // 按人项不进后厨
			}
			itemsByOrder[it.OrderID] = append(itemsByOrder[it.OrderID], it)
		}
	}

	out := make([]AdminKitchenOrderDTO, 0, len(rows))
	for _, r := range rows {
		its := itemsByOrder[r.ID]
		if len(its) == 0 {
			continue // 只剩按人项的单子不进 KDS
		}
		no := ""
		if r.OutTradeNo != nil {
			no = *r.OutTradeNo
		}
		dto := AdminKitchenOrderDTO{
			ID: r.ID, No: no, TableNo: r.TableNo, PayMode: r.PayMode,
			Status: r.Status, CreatedAt: r.CreatedAt, Station: "",
			Items: make([]AdminKitchenItemDTO, 0, len(its)),
		}
		for _, it := range its {
			dto.Items = append(dto.Items, AdminKitchenItemDTO{ID: it.ID, Name: it.Name, Specs: it.Specs, Qty: it.Qty})
		}
		out = append(out, dto)
	}
	return &AdminKitchenResp{Items: out}, nil
}

// AdminRefundDTO 退款单（商家端退款）
type AdminRefundDTO struct {
	ID          int64     `json:"id"`
	OrderNo     *string   `json:"order_no"`
	TableNo     *string   `json:"table_no"`
	AmountCents int64     `json:"amount_cents"`
	Reason      *string   `json:"reason,omitempty"`
	Channel     string    `json:"channel"`
	Status      string    `json:"status"` // success / pending / failed（前端映射展示）
	RequestedBy *string   `json:"requested_by,omitempty"`
	RequestedAt time.Time `json:"requested_at"`
	HandledBy   *string   `json:"handled_by,omitempty"`
	HandledAt   *time.Time `json:"handled_at,omitempty"`
}

// AdminRefundsResp GET /admin/refunds 响应
type AdminRefundsResp struct {
	Items []AdminRefundDTO `json:"items"`
}

// ListRefunds GET /admin/refunds：退款单列表（单步退款，请求=处理，处理人=操作员）。
func (s *OpsService) ListRefunds(ctx context.Context) (*AdminRefundsResp, *respond.BizErr) {
	var rows []struct {
		model.Refund
		OrderNo *string
		TableNo *string
		OpName  *string
	}
	if err := s.db.Table("refunds r").
		Joins("LEFT JOIN orders o ON o.id = r.order_id").
		Joins("LEFT JOIN tables t ON t.id = o.table_id").
		Joins("LEFT JOIN employees e ON e.id = r.operator_id").
		Where("o.shop_id = ?", s.shopID).
		Order("r.id DESC").
		Select("r.*, o.out_trade_no AS order_no, t.table_no, e.name AS op_name").Scan(&rows).Error; err != nil {
		slog.Error("list refunds failed", "err", err)
		return nil, respond.ErrInternal
	}
	out := make([]AdminRefundDTO, 0, len(rows))
	for _, r := range rows {
		at := r.CreatedAt
		out = append(out, AdminRefundDTO{
			ID: r.ID, OrderNo: r.OrderNo, TableNo: r.TableNo,
			AmountCents: int64(r.Amount), Reason: r.Reason, Channel: r.Channel,
			Status: r.Status, RequestedBy: r.OpName, RequestedAt: r.CreatedAt,
			HandledBy: r.OpName, HandledAt: &at, // 单步退款：请求即处理
		})
	}
	return &AdminRefundsResp{Items: out}, nil
}
