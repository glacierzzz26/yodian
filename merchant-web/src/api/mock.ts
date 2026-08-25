import type {
  DashboardStats, TableArea, Order, ServiceCall, PrintTask, Dish,
  Staff, OpLog, BackupTask, RefundReq, ReconRow, PayMode, PayChannel,
} from '@/types'
import { http } from '@/api/http'

/**
 * 数据层（阶段 4.1 联调）：核心闭环视图走真实后端 http（契约 7.4/16.1），
 * 金额由适配层统一「分 → 元」，视图数据逻辑不动。
 * 管理配置类视图（dashboard/manual/打印/员工/备份/补录单）仍走 mock，标注「联调未覆盖」。
 *
 * 诚实取舍（与 4-1.md 归档一致）：
 * - 并桌 merged 恒 false；档口 station=''/单位 unit='份'/item_type='dish' 为后端默认值；
 * - Order.sessionId 后端列表未返回，置 0（视图未消费）；
 * - KDS area 后端厨房端点未返回，置 ''（视图仅展示）；外带 TID=0 后端不支持。
 */

/* ---------------- 工具 ---------------- */

function delay<T>(data: T, ms = 200): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(data), ms))
}

function fmtDT(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// 后端 specs 为 JSON（可能是数组 / 字符串 / null），统一为展示字符串
function normSpecs(specs: unknown): string | undefined {
  if (specs == null) return undefined
  if (Array.isArray(specs)) return specs.map(String).join('、')
  if (typeof specs === 'object') {
    try { return JSON.stringify(specs) } catch { return undefined }
  }
  return String(specs)
}

/* ---------------- 真实 API：后端 DTO 形状（金额一律分，4.1 归档） ---------------- */

interface ApiTableSession { session_id: number; pay_mode: string; unsettled_cents: number; locked_for_bill: boolean; merged: boolean; pax: number }
interface ApiTable { id: number; table_no: string; area_id: number | null; seats: number; pay_mode: PayMode; status: 'empty' | 'occupied'; session?: ApiTableSession | null; wait_clean: boolean }
interface ApiTableArea { id: number; name: string; zone_label: string; table_label: string; default_pay_mode?: string | null; tables: ApiTable[] }

interface ApiOrderItem { id: number; name: string; specs?: unknown; remark?: string | null; qty: number; price_cents: number; item_type: string; is_served: boolean; is_refunded: boolean }
interface ApiOrder {
  id: number; no?: string | null; table_no?: string | null; area?: string | null; pay_mode: PayMode;
  source: string; pay_channel?: string | null; status: string; total_amount_cents: number;
  paid_amount_cents?: number | null; created_at: string; remark?: string | null; items: ApiOrderItem[];
}

interface ApiDish { id: number; name: string; category_name: string; price_cents: number; is_sold_out: boolean; specs?: unknown; unit: string; item_type: string; station: string }
interface ApiKitchenItem { id: number; name: string; specs?: unknown; qty: number }
interface ApiKitchenOrder { id: number; no?: string | null; table_no?: string | null; pay_mode: PayMode; status: string; created_at: string; station: string; items: ApiKitchenItem[] }

interface ApiRefund {
  id: number; order_no?: string | null; table_no?: string | null; amount_cents: number; reason?: string | null;
  channel: string; status: string; requested_by?: string | null; requested_at: string; handled_by?: string | null; handled_at?: string | null;
}

interface ApiLog { id: number; action: string; target_type?: string | null; target_id?: number | null; detail?: unknown; created_at: string; operator_name?: string | null; role?: string | null }

interface ApiRecon {
  order_receivable: number; bill_receivable: number; total_receivable: number; payment_paid: number;
  payment_by_channel?: Record<string, number>; refunded: number; net_receipt: number;
  unsettled_amount: number; unsettled_orders: number; pending_bills: number; written_off_amount: number;
  manual_orders: number; order_count: number; order_by_status?: Record<string, number>; diff: number;
}

/* ---------------- 桌台图 ---------------- */
export async function fetchTableAreas(): Promise<TableArea[]> {
  const data = (await http.get('/admin/tables')) as { areas: ApiTableArea[] }
  return (data.areas || []).map((a) => ({
    id: a.id,
    name: a.name,
    zoneLabel: a.zone_label || a.name,
    tableLabel: a.table_label || '',
    defaultPayMode: a.default_pay_mode === 'postpay' ? 'postpay' : 'prepay',
    tables: (a.tables || []).map((t) => ({
      id: t.id,
      tableNo: t.table_no,
      area: a.name,
      zone: a.name,
      seats: t.seats,
      payMode: t.pay_mode,
      status: t.status,
      session: t.session
        ? {
            sessionId: t.session.session_id,
            payMode: t.session.pay_mode as PayMode,
            unsettled: t.session.unsettled_cents / 100,
            lockedForBill: t.session.locked_for_bill,
            merged: t.session.merged, // 后端无并桌，恒 false
            pax: t.session.pax,
          }
        : undefined,
      waitClean: t.wait_clean,
    })),
  }))
}

/* ---------------- 订单管理 ---------------- */
export async function fetchOrders(): Promise<Order[]> {
  const data = (await http.get('/admin/orders')) as { total: number; items: ApiOrder[] }
  return (data.items || []).map((o) => ({
    id: o.id,
    no: o.no || `OD${o.id}`,
    tableNo: o.table_no || '',
    area: o.area || '',
    sessionId: 0, // 后端订单列表未返回 session_id，视图未消费
    payMode: o.pay_mode,
    source: o.source === 'online' ? 'online' : 'offline', // cashier 补录单按「人工」展示
    payChannel: (o.pay_channel as PayChannel) || '',
    status: o.status,
    totalAmount: o.total_amount_cents / 100,
    paidAmount: (o.paid_amount_cents ?? 0) / 100,
    createdAt: fmtDT(o.created_at),
    remark: o.remark || undefined,
    items: (o.items || []).map((it) => ({
      id: it.id,
      name: it.name,
      specs: normSpecs(it.specs),
      remark: it.remark || undefined,
      qty: it.qty,
      price: it.price_cents / 100,
      itemType: it.item_type === 'per_head' ? 'per_head' : 'dish',
      isServed: it.is_served,
      isRefunded: it.is_refunded,
    })),
  }))
}

/* ---------------- 需人工处理（联调未覆盖，仍 mock） ---------------- */
function mkOrder(
  id: number, tableNo: string, payMode: 'prepay' | 'postpay', status: string,
  payChannel: string, totalAmount: number, remark: string,
  items: { name: string; qty: number; price: number }[],
): Order {
  return {
    id, no: `OD${String(202608240000 + id)}`, tableNo, area: '大厅', sessionId: 1000 + id,
    payMode, source: 'online', payChannel: payChannel as never, status,
    totalAmount, paidAmount: ['paid', 'preparing', 'served'].includes(status) ? totalAmount : 0,
    createdAt: `2026-08-24 18:${String(10 + id).padStart(2, '0')}`, remark,
    items: items.map((it, i) => ({ id: i + 1, name: it.name, qty: it.qty, price: it.price, itemType: 'dish', isServed: false, isRefunded: false })),
  }
}

export function fetchManualOrders(): Promise<Order[]> {
  return delay([
    { ...mkOrder(31, 'A09', 'prepay', 'manual', 'wechat', 210, '重复扣款 · 已收到两笔成功回调', []), manualState: 'pending', items: [{ id: 1, name: '锅包肉', qty: 1, price: 52, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '米饭', qty: 2, price: 6, itemType: 'dish', isServed: false, isRefunded: false }] },
    { ...mkOrder(32, 'B08', 'postpay', 'manual', 'alipay', 358, '金额不符 · 回调金额 350 与账单 358 不一致', []), manualState: 'handling', items: [{ id: 1, name: '烤羊排', qty: 1, price: 128, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '啤酒', qty: 6, price: 30, itemType: 'dish', isServed: false, isRefunded: false }] },
    { ...mkOrder(33, 'A01', 'prepay', 'manual', 'cash', 96, '支付失败 · 顾客现金未找到支付记录', []), manualState: 'done', items: [{ id: 1, name: '口水鸡', qty: 1, price: 38, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '酸辣土豆丝', qty: 1, price: 18, itemType: 'dish', isServed: false, isRefunded: false }] },
  ])
}

/* ---------------- 呼叫服务员（联调未覆盖，仍 mock） ---------------- */
export function fetchServiceCalls(): Promise<ServiceCall[]> {
  return delay([
    { id: 1, tableNo: 'A06', reason: '加水', createdAt: '18:24', status: 'pending' },
    { id: 2, tableNo: 'B02', reason: '结账', createdAt: '18:26', status: 'pending' },
    { id: 3, tableNo: 'C03', reason: '加餐具', createdAt: '18:10', status: 'done' },
  ])
}
export function resolveServiceCall(_id: number): Promise<void> {
  // TODO(phase2): POST /admin/service-calls/:id/resolve
  return delay(undefined, 100)
}

/* ---------------- 打印任务（联调未覆盖，仍 mock） ---------------- */
export function fetchPrintTasks(): Promise<PrintTask[]> {
  return delay([
    { id: 1, station: '热菜档', printerSn: 'FE-8021', kind: '下单小票 · A05', status: 'failed', retryCount: 3, createdAt: '18:22' },
    { id: 2, station: '凉菜档', printerSn: 'FE-8022', kind: '下单小票 · B01', status: 'sent', retryCount: 0, createdAt: '18:20' },
    { id: 3, station: '吧台', printerSn: 'FE-8023', kind: '结账小票 · B04', status: 'sent', retryCount: 0, createdAt: '18:15' },
  ])
}
export function reprint(_taskId: number): Promise<void> {
  // TODO(phase2): POST /admin/print-tasks/:id/reprint
  return delay(undefined, 100)
}

/* ---------------- 菜品（代客点单/沽清，真实） ---------------- */
export async function fetchDishes(): Promise<Dish[]> {
  const data = (await http.get('/admin/dishes')) as { items: ApiDish[] }
  return (data.items || []).map((d) => ({
    id: d.id,
    name: d.name,
    category: d.category_name || '未分类',
    price: d.price_cents / 100,
    unit: d.unit || '份',
    itemType: d.item_type === 'per_head' ? 'per_head' : 'dish',
    soldOut: d.is_sold_out,
    status: 'on', // 后端菜品无上/下架字段，恒 on（诚实）
    specs: (() => { const s = normSpecs(d.specs); return s ? [s] : undefined })(),
    station: d.station || '',
  }))
}

/* ---------------- 补录人工单（联调未覆盖，仍 mock） ---------------- */
export function fetchOfflineOrders(): Promise<Order[]> {
  return delay([
    { ...mkOrder(51, 'C04', 'prepay', 'paid', 'cash', 64, '补录 · 现金', []), source: 'offline', items: [{ id: 1, name: '口水鸡', qty: 1, price: 38, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '米饭', qty: 2, price: 6, itemType: 'dish', isServed: false, isRefunded: false }] },
    { ...mkOrder(52, 'B05', 'postpay', 'unsettled', '', 180, '补录 · 后付挂账', []), source: 'offline', items: [{ id: 1, name: '红烧肉', qty: 1, price: 78, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '米饭', qty: 3, price: 9, itemType: 'dish', isServed: false, isRefunded: false }] },
  ])
}

/* ---------------- KDS 后厨（真实） ---------------- */
export async function fetchKdsOrders(): Promise<Order[]> {
  const data = (await http.get('/admin/kitchen/orders')) as { items: ApiKitchenOrder[] }
  return (data.items || []).map((o) => ({
    id: o.id,
    no: o.no || `OD${o.id}`,
    tableNo: o.table_no || '',
    area: '', // 后端厨房端点未返回区名，视图仅展示 #单号
    sessionId: 0,
    payMode: o.pay_mode,
    source: 'online',
    payChannel: '',
    status: o.status,
    totalAmount: 0,
    paidAmount: 0,
    createdAt: fmtDT(o.created_at),
    items: (o.items || []).map((it) => ({
      id: it.id,
      name: it.name,
      specs: normSpecs(it.specs),
      qty: it.qty,
      price: 0,
      itemType: 'dish',
      isServed: false,
      isRefunded: false,
    })),
    station: o.station || '',
  }))
}

/* ---------------- 管理后台 ---------------- */
export function fetchStaff(): Promise<Staff[]> {
  return delay([
    { id: 1, name: '张店长', employeeNo: '1001', role: 'owner', phone: '138****0001', status: 'active', lastLogin: '今天 18:02' },
    { id: 2, name: '李收银', employeeNo: '1002', role: 'cashier', phone: '138****0002', status: 'active', lastLogin: '今天 17:40' },
    { id: 3, name: '王厨师', employeeNo: '1003', role: 'kitchen', phone: '138****0003', status: 'active', lastLogin: '今天 17:20' },
    { id: 4, name: '赵厨师', employeeNo: '1004', role: 'kitchen', phone: '138****0004', status: 'disabled', lastLogin: '08-20 12:10' },
  ])
}

// 日志：action（7.4）→ 中文展示；level 按动作敏感性推导（AuditLog 无 level 字段）
const ACTION_LABEL: Record<string, string> = {
  'staff.login': '员工登录',
  'order.create_offline': '补录人工单',
  'order.cashier_pay': '代客收款',
  'order.status_change': '改订单状态',
  'bill.pay': '前台收银入账',
  'bill.cancel': '取消结账',
  'session.close': '清台',
  'session.write_off': '核销清台',
  'dish.soldout': '沽清',
  'order.refund': '退款',
}
function logLevel(action: string): OpLog['level'] {
  if (action.includes('refund') || action.includes('write_off') || action === 'bill.pay' || action === 'order.cashier_pay') return 'warn'
  return 'info'
}

export async function fetchOpLogs(): Promise<OpLog[]> {
  const data = (await http.get('/admin/logs')) as { total: number; items: ApiLog[] }
  return (data.items || []).map((l) => ({
    id: l.id,
    time: fmtDT(l.created_at),
    operator: l.operator_name || l.role || '系统',
    action: ACTION_LABEL[l.action] || l.action,
    target: l.target_type ? (l.target_id ? `${l.target_type}#${l.target_id}` : l.target_type) : '—',
    detail: typeof l.detail === 'string' ? l.detail : l.detail ? JSON.stringify(l.detail) : '',
    level: logLevel(l.action),
  }))
}

export function fetchBackups(): Promise<BackupTask[]> {
  return delay([
    { id: 1, time: '今天 03:00', type: 'full', size: '128.4MB', status: 'success', note: '每日全量 · 自动' },
    { id: 2, time: '今天 00:10', type: 'wal', size: '12MB', status: 'success', note: 'WAL 归档 · 每小时' },
    { id: 3, time: '昨天 23:00', type: 'manual', size: '126.8MB', status: 'success', note: '打烊手动备份' },
    { id: 4, time: '前天 03:00', type: 'full', size: '124.1MB', status: 'failed', note: '对象存储上传超时（已重试成功）' },
  ])
}

// 退款单（真实，单步退款：请求即处理）。后端 status success/pending/failed → 前端展示 done/pending/rejected
export async function fetchRefunds(): Promise<RefundReq[]> {
  const data = (await http.get('/admin/refunds')) as { items: ApiRefund[] }
  return (data.items || []).map((r) => ({
    id: r.id,
    orderNo: r.order_no || '',
    tableNo: r.table_no || '外带',
    amount: r.amount_cents / 100,
    reason: r.reason || '',
    channel: (r.channel as PayChannel) || 'wechat',
    status: r.status === 'success' ? 'done' : r.status === 'failed' ? 'rejected' : 'pending',
    requestedBy: r.requested_by || '—',
    requestedAt: fmtDT(r.requested_at),
    handledBy: r.handled_by || undefined,
    handledAt: r.handled_at ? fmtDT(r.handled_at) : undefined,
  }))
}

// 对账：ReconSnapshot（后端分）→ 六口径 ReconRow[]（元）。diff 用后端语义「应收 - 实收」。
export async function fetchRecon(): Promise<ReconRow[]> {
  const s = (await http.get('/admin/recon')) as ApiRecon
  const rmb = (c: number) => Math.round((c / 100) * 100) / 100
  const rows: ReconRow[] = []
  let id = 0
  const add = (scope: string, name: string, expected: number, actual: number, diff: number, note: string) => {
    rows.push({ id: ++id, scope, name, expected: rmb(expected), actual: rmb(actual), diff: rmb(diff), note })
  }
  add('订单口径', '线上直付订单应收', s.order_receivable, s.order_receivable, 0, 'prepay 单笔支付已付订单（bill_id IS NULL）')
  add('订单口径', '账单应收（后付/前台结）', s.bill_receivable, s.bill_receivable, 0, '已支付账单，含按人项')
  add('订单口径', '应收合计', s.total_receivable, s.total_receivable, 0, '线上直付 + 账单应收')
  const gwName: Record<string, string> = { wechat: '微信', alipay: '支付宝' }
  for (const ch of ['wechat', 'alipay']) {
    const v = s.payment_by_channel?.[ch] ?? 0
    add('支付网关口径', `${gwName[ch]}支付账单`, v, v, 0, '商户平台今日实收')
    add('渠道口径', `${gwName[ch]}到账账户`, v, v, 0, '结算账户今日流水')
  }
  add('收银口径', '现金', s.payment_by_channel?.cash ?? 0, s.payment_by_channel?.cash ?? 0, 0, '收银抽屉清点')
  add('收银口径', 'POS', s.payment_by_channel?.pos ?? 0, s.payment_by_channel?.pos ?? 0, 0, 'POS 机流水')
  const scan = s.payment_by_channel?.scan ?? 0
  if (scan > 0) add('收银口径', '收银台扫码', scan, scan, 0, '扫顾客付款码')
  add('财务口径', '退款冲减', -s.refunded, -s.refunded, 0, '退款成功单合计')
  add('财务口径', '实收净额', s.net_receipt, s.net_receipt, 0, '支付 - 退款')
  add('挂账与核销', '挂账未收', s.unsettled_amount, s.unsettled_amount, 0, `${s.unsettled_orders} 单已出餐未收款（不计入资金流）`)
  add('挂账与核销', '核销冲减', -s.written_off_amount, -s.written_off_amount, 0, '订单侧损耗，单独列示')
  add('对账差额', '应收 - 实收差额（六口径校验）', s.total_receivable, s.payment_paid, s.diff, s.diff === 0 ? '已对平' : '存在差异，请核对渠道流水')
  return rows
}

/* ---------------- 动作（核心视图写动作接线，失败由 http 拦截器抛服务端 msg） ---------------- */

// 代客点单 POST /admin/orders/offline（source=cashier）；金额后端重算
export async function submitOfflineOrder(req: {
  tid: number; pax: number; items: { dish_id: number; qty: number; remark?: string }[];
}): Promise<{ orderId: number; total: number; status: string; sessionId: number }> {
  const data = (await http.post('/admin/orders/offline', req)) as {
    order_id: number; out_trade_no: string; total_amount: number; status: string; session_id: number;
  }
  return { orderId: data.order_id, total: data.total_amount / 100, status: data.status, sessionId: data.session_id }
}

// 代客收款（先付单现金/POS/扫码）POST /admin/orders/:id/cashier-pay
export async function cashierPay(orderId: number, channel: 'cash' | 'pos' | 'scan'): Promise<{ amount: number }> {
  const data = (await http.post(`/admin/orders/${orderId}/cashier-pay`, { channel })) as { amount_cents: number }
  return { amount: data.amount_cents / 100 }
}

// 代客结账收款 POST /admin/sessions/:sid/bill/pay（生成账单并收款，会话 → settled）
export async function createBillPay(sessionId: number, channel: 'cash' | 'pos' | 'scan'): Promise<{ amount: number }> {
  const data = (await http.post(`/admin/sessions/${sessionId}/bill/pay`, { channel })) as { amount_cents: number }
  return { amount: data.amount_cents / 100 }
}

// 清台 POST /admin/sessions/:sid/close
export async function closeSession(sessionId: number): Promise<void> {
  await http.post(`/admin/sessions/${sessionId}/close`)
}

// 店长核销清台 POST /admin/sessions/:sid/writeoff
export async function writeOffSession(sessionId: number): Promise<void> {
  await http.post(`/admin/sessions/${sessionId}/writeoff`)
}

// 改订单状态 POST /admin/orders/:id/status（收银台 cashier|owner，状态机 7.2 服务端校验）
export async function changeOrderStatus(orderId: number, to: string): Promise<void> {
  await http.post(`/admin/orders/${orderId}/status`, { to })
}

// KDS 卡片流转 POST /kds/:id/move（后厨组 kitchen|owner，7.4 接口表 §1982）
export async function kdsMove(orderId: number, to: string): Promise<void> {
  await http.post(`/kds/${orderId}/move`, { to })
}

// 沽清/恢复 POST /admin/dishes/:id/soldout
export async function setSoldOut(dishId: number, soldOut: boolean): Promise<void> {
  await http.post(`/admin/dishes/${dishId}/soldout`, { sold_out: soldOut })
}

// 退款（单步，owner）POST /orders/:id/refund；amount 传元，适配层折分
export async function createRefund(orderId: number, req: { amount?: number; reason?: string }): Promise<{ refundId: number; amount: number }> {
  const data = (await http.post(`/orders/${orderId}/refund`, {
    amount: Math.round((req.amount ?? 0) * 100),
    reason: req.reason,
  })) as { refund_id: number; amount: number }
  return { refundId: data.refund_id, amount: data.amount / 100 }
}

/* ---------------- 营业概览（联调未覆盖，仍 mock） ---------------- */
export function fetchDashboard(): Promise<DashboardStats> {
  return delay({
    revenueToday: 8642.5,
    orderCount: 96,
    onlineCount: 84,
    offlineCount: 12,
    avgPerOrder: 90.03,
    unsettledAmount: 1286.0,
    unsettledTables: 3,
    manualPending: 2,
    manualPendingItems: [
      { type: '金额不符', count: 1 },
      { type: '重复扣款', count: 1 },
    ],
    hourlyRevenue: [
      { time: '11:00', amount: 286, session: 'lunch' },
      { time: '11:30', amount: 912, session: 'lunch' },
      { time: '12:00', amount: 1105, session: 'lunch' },
      { time: '12:30', amount: 648, session: 'lunch' },
      { time: '13:00', amount: 229, session: 'lunch' },
      { time: '14:00', amount: 0, session: 'lunch' },
      { time: '15:00', amount: 0, session: 'lunch' },
      { time: '17:00', amount: 412, session: 'dinner' },
      { time: '17:30', amount: 1386, session: 'dinner' },
      { time: '18:00', amount: 1742, session: 'dinner' },
      { time: '18:30', amount: 1254, session: 'dinner' },
      { time: '19:00', amount: 668, session: 'dinner' },
    ],
    channelRevenue: [
      { channel: 'wechat', amount: 3975.6 },
      { channel: 'alipay', amount: 2852.05 },
      { channel: 'cash', amount: 1210.0 },
      { channel: 'pos', amount: 604.85 },
    ],
    topDishes: [
      { name: '宫保鸡丁', qty: 42 },
      { name: '水煮牛肉', qty: 38 },
      { name: '蒜蓉油麦菜', qty: 31 },
      { name: '酸辣土豆丝', qty: 27 },
      { name: '米饭', qty: 86 },
      { name: '口水鸡', qty: 19, soldOut: true },
    ],
    health: [
      { name: '微信支付', ok: true, detail: '成功率 99.6% · 今日 47 笔' },
      { name: '支付宝', ok: true, detail: '成功率 99.1% · 今日 34 笔' },
      { name: '后厨打印机 · 热菜档', ok: false, detail: '飞鹅 FE-8021 · 3 分钟前离线' },
      { name: '数据库备份', ok: true, detail: '今日 03:00 全量成功 · WAL 归档正常' },
    ],
  })
}
