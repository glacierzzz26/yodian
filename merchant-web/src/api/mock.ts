import type {
  DashboardStats, TableArea, Order, ServiceCall, PrintTask, Dish,
  Staff, OpLog, BackupTask, RefundReq, ReconRow,
} from '@/types'

/**
 * mock 数据层：契约与设计方案 7.4/16.1 对齐。
 * 后端就绪后，把这里的函数体替换为 http 调用（见 http.ts），页面无需改动。
 * 数据诚实：未实现的能力返回空态，不造假。
 */

function delay<T>(data: T, ms = 200): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(data), ms))
}

/* ---------------- 营业概览 ---------------- */
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

/* ---------------- 桌台图 ---------------- */
const areaTpl: Array<{ id: number; name: string; zoneLabel: string; tableLabel: string; payMode: 'prepay' | 'postpay'; tableNos: string[] }> = [
  { id: 1, name: '大厅', zoneLabel: '一楼大厅', tableLabel: 'A', payMode: 'prepay', tableNos: Array.from({ length: 10 }, (_, i) => `A${String(i + 1).padStart(2, '0')}`) },
  { id: 2, name: '包间', zoneLabel: '包间 · 二楼', tableLabel: 'B', payMode: 'postpay', tableNos: Array.from({ length: 8 }, (_, i) => `B${String(i + 1).padStart(2, '0')}`) },
  { id: 3, name: '露台', zoneLabel: '露台', tableLabel: 'C', payMode: 'prepay', tableNos: Array.from({ length: 6 }, (_, i) => `C${String(i + 1).padStart(2, '0')}`) },
]

// 模拟部分桌台处于就餐中/未结账/待清台/并桌/锁单状态
const occupiedSpec: Record<string, { unsettled?: number; locked?: boolean; merged?: boolean; waitClean?: boolean }> = {
  A02: { unsettled: 0 },
  A03: { unsettled: 0 },
  A05: { unsettled: 0 },
  A07: { unsettled: 0, merged: true },
  B01: { unsettled: 486 },
  B03: { unsettled: 0 },
  B04: { unsettled: 0, locked: true },
  B06: { unsettled: 800, waitClean: true },
  C02: { unsettled: 0 },
}

export function fetchTableAreas(): Promise<TableArea[]> {
  return delay(
    areaTpl.map((area) => ({
      id: area.id,
      name: area.zoneLabel,
      zoneLabel: area.zoneLabel,
      tableLabel: area.tableLabel,
      defaultPayMode: area.payMode,
      tables: area.tableNos.map((no, idx) => {
        const spec = occupiedSpec[no]
        const occupied = !!spec
        return {
          id: idx + 1,
          tableNo: no,
          area: area.zoneLabel,
          zone: area.name,
          seats: area.payMode === 'postpay' ? 10 : 4,
          payMode: area.payMode,
          status: occupied ? 'occupied' : 'empty',
          session: occupied
            ? {
                sessionId: 1000 + idx,
                payMode: area.payMode,
                unsettled: spec?.unsettled ?? 0,
                lockedForBill: spec?.locked ?? false,
                merged: spec?.merged ?? false,
                pax: area.payMode === 'postpay' ? 8 : 3,
              }
            : undefined,
          waitClean: spec?.waitClean,
        }
      }),
    })),
  )
}

/* ---------------- 订单管理 ---------------- */
export function fetchOrders(): Promise<Order[]> {
  return delay([
    mkOrder(1, 'A05', 'prepay', 'paid', 'wechat', 126, '堂食 · 3 人', [
      { name: '宫保鸡丁', qty: 1, price: 46 },
      { name: '酸辣土豆丝', qty: 1, price: 18 },
      { name: '米饭', qty: 3, price: 6 },
    ]),
    mkOrder(2, 'A03', 'prepay', 'preparing', 'alipay', 88, '不要辣', [
      { name: '水煮牛肉', qty: 1, price: 68 },
      { name: '米饭', qty: 2, price: 6 },
    ]),
    mkOrder(3, 'B01', 'postpay', 'unsettled', '', 486, '包间 · 未结账', [
      { name: '清蒸鲈鱼', qty: 1, price: 128 },
      { name: '红烧肉', qty: 1, price: 78 },
      { name: '蒜蓉油麦菜', qty: 2, price: 24 },
    ]),
    mkOrder(4, 'A02', 'prepay', 'served', 'wechat', 152, '', [
      { name: '剁椒鱼头', qty: 1, price: 98 },
      { name: '米饭', qty: 4, price: 8 },
    ]),
    mkOrder(5, 'C02', 'prepay', 'pending', '', 64, '外带', [
      { name: '口水鸡', qty: 1, price: 38 },
      { name: '米饭', qty: 2, price: 6 },
    ]),
    mkOrder(6, 'B04', 'postpay', 'unsettled', '', 520, '结账中 · 锁单', [
      { name: '佛跳墙', qty: 1, price: 298 },
      { name: '红烧狮子头', qty: 2, price: 96 },
    ]),
  ])
}

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

/* ---------------- 需人工处理 ---------------- */
export function fetchManualOrders(): Promise<Order[]> {
  return delay([
    { ...mkOrder(31, 'A09', 'prepay', 'manual', 'wechat', 210, '重复扣款 · 已收到两笔成功回调', []), manualState: 'pending', items: [{ id: 1, name: '锅包肉', qty: 1, price: 52, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '米饭', qty: 2, price: 6, itemType: 'dish', isServed: false, isRefunded: false }] },
    { ...mkOrder(32, 'B08', 'postpay', 'manual', 'alipay', 358, '金额不符 · 回调金额 350 与账单 358 不一致', []), manualState: 'handling', items: [{ id: 1, name: '烤羊排', qty: 1, price: 128, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '啤酒', qty: 6, price: 30, itemType: 'dish', isServed: false, isRefunded: false }] },
    { ...mkOrder(33, 'A01', 'prepay', 'manual', 'cash', 96, '支付失败 · 顾客现金未找到支付记录', []), manualState: 'done', items: [{ id: 1, name: '口水鸡', qty: 1, price: 38, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '酸辣土豆丝', qty: 1, price: 18, itemType: 'dish', isServed: false, isRefunded: false }] },
  ])
}

/* ---------------- 呼叫服务员 ---------------- */
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

/* ---------------- 打印任务 ---------------- */
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

/* ---------------- 菜品（代客点单/补录单） ---------------- */
const dishCatalog: Array<Omit<Dish, 'id'>> = [
  { name: '宫保鸡丁', category: '热菜', price: 46, unit: '份', itemType: 'dish', station: '热菜档' },
  { name: '水煮牛肉', category: '热菜', price: 68, unit: '份', itemType: 'dish', station: '热菜档' },
  { name: '剁椒鱼头', category: '热菜', price: 98, unit: '份', itemType: 'dish', station: '热菜档' },
  { name: '红烧肉', category: '热菜', price: 78, unit: '份', itemType: 'dish', station: '热菜档' },
  { name: '清蒸鲈鱼', category: '热菜', price: 128, unit: '份', itemType: 'dish', station: '热菜档' },
  { name: '佛跳墙', category: '热菜', price: 298, unit: '份', itemType: 'dish', station: '热菜档', soldOut: true },
  { name: '口水鸡', category: '凉菜', price: 38, unit: '份', itemType: 'dish', station: '凉菜档' },
  { name: '酸辣土豆丝', category: '凉菜', price: 18, unit: '份', itemType: 'dish', station: '凉菜档' },
  { name: '夫妻肺片', category: '凉菜', price: 42, unit: '份', itemType: 'dish', station: '凉菜档', status: 'off' },
  { name: '烤羊排', category: '热菜', price: 128, unit: '份', itemType: 'dish', station: '热菜档' },
  { name: '青菜豆腐汤', category: '汤羹', price: 22, unit: '煲', itemType: 'dish', station: '汤羹档' },
  { name: '西红柿蛋汤', category: '汤羹', price: 16, unit: '煲', itemType: 'dish', station: '汤羹档' },
  { name: '米饭', category: '主食', price: 3, unit: '碗', itemType: 'dish', station: '主食档' },
  { name: '手工水饺', category: '主食', price: 22, unit: '份', itemType: 'dish', station: '主食档' },
  { name: '青岛啤酒', category: '酒水', price: 10, unit: '瓶', itemType: 'dish', station: '吧台' },
  { name: '自制酸梅汤', category: '酒水', price: 12, unit: '杯', itemType: 'dish', station: '吧台' },
  { name: '自助火锅', category: '自助', price: 58, unit: '人', itemType: 'per_head', station: '热菜档' },
  { name: '自助烧烤', category: '自助', price: 68, unit: '人', itemType: 'per_head', station: '热菜档' },
]
export function fetchDishes(): Promise<Dish[]> {
  return delay(dishCatalog.map((d, i) => ({ ...d, id: i + 1, status: d.status ?? 'on' })))
}

/* ---------------- 补录人工单（降级/堂食代录） ---------------- */
export function fetchOfflineOrders(): Promise<Order[]> {
  return delay([
    { ...mkOrder(51, 'C04', 'prepay', 'paid', 'cash', 64, '补录 · 现金', []), source: 'offline', items: [{ id: 1, name: '口水鸡', qty: 1, price: 38, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '米饭', qty: 2, price: 6, itemType: 'dish', isServed: false, isRefunded: false }] },
    { ...mkOrder(52, 'B05', 'postpay', 'unsettled', '', 180, '补录 · 后付挂账', []), source: 'offline', items: [{ id: 1, name: '红烧肉', qty: 1, price: 78, itemType: 'dish', isServed: false, isRefunded: false }, { id: 2, name: '米饭', qty: 3, price: 9, itemType: 'dish', isServed: false, isRefunded: false }] },
  ])
}

/* ---------------- KDS 后厨 ---------------- */
const kdsSeed: Array<{ tableNo: string; payMode: 'prepay' | 'postpay'; remark: string; station: string; dish: string; specs?: string }> = [
  { tableNo: 'A05', payMode: 'prepay', remark: '不要香菜', station: '热菜档', dish: '宫保鸡丁' },
  { tableNo: 'A03', payMode: 'prepay', remark: '少辣', station: '热菜档', dish: '水煮牛肉' },
  { tableNo: 'B01', payMode: 'postpay', remark: '', station: '热菜档', dish: '清蒸鲈鱼' },
  { tableNo: 'C01', payMode: 'prepay', remark: '', station: '凉菜档', dish: '口水鸡', specs: '麻辣' },
  { tableNo: 'A07', payMode: 'prepay', remark: '免姜', station: '汤羹档', dish: '青菜豆腐汤' },
]
export function fetchKdsOrders(): Promise<Order[]> {
  return delay(
    kdsSeed.map((s, i) => ({
      ...mkOrder(41 + i, s.tableNo, s.payMode, 'preparing', '', 0, s.remark, []),
      no: `OD${202608240040 + i}`,
      station: s.station,
      items: [{ id: 1, name: s.dish, specs: s.specs, qty: 1, price: 0, itemType: 'dish' as const, isServed: false, isRefunded: false }],
    })),
  )
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

export function fetchOpLogs(): Promise<OpLog[]> {
  return delay([
    { id: 1, time: '18:24', operator: '张店长', action: '开启降级模式', target: '系统', detail: '降级模式开启，顾客端下单入口关闭', level: 'warn' },
    { id: 2, time: '18:22', operator: '李收银', action: '补录人工单', target: 'OD-MAN0002', detail: '桌台 B05 · 后付挂账 ¥180', level: 'info' },
    { id: 3, time: '18:10', operator: '李收银', action: '发起结账', target: 'B04', detail: '锁单，待收款 ¥520', level: 'info' },
    { id: 4, time: '17:55', operator: '张店长', action: '改价', target: '口水鸡', detail: '原价 ¥40 → ¥38', level: 'warn' },
    { id: 5, time: '17:30', operator: '王厨师', action: '沽清', target: '佛跳墙', detail: '食材售罄，当日沽清', level: 'info' },
    { id: 6, time: '16:00', operator: '系统', action: '备份', target: 'DB 全量', detail: '全量备份成功 128MB → 对象存储', level: 'info' },
  ])
}

export function fetchBackups(): Promise<BackupTask[]> {
  return delay([
    { id: 1, time: '今天 03:00', type: 'full', size: '128.4MB', status: 'success', note: '每日全量 · 自动' },
    { id: 2, time: '今天 00:10', type: 'wal', size: '12MB', status: 'success', note: 'WAL 归档 · 每小时' },
    { id: 3, time: '昨天 23:00', type: 'manual', size: '126.8MB', status: 'success', note: '打烊手动备份' },
    { id: 4, time: '前天 03:00', type: 'full', size: '124.1MB', status: 'failed', note: '对象存储上传超时（已重试成功）' },
  ])
}

export function fetchRefunds(): Promise<RefundReq[]> {
  return delay([
    { id: 1, orderNo: 'OD202608240031', tableNo: 'A09', amount: 52, reason: '顾客未收到菜，人工退款', channel: 'wechat', status: 'pending', requestedBy: '李收银', requestedAt: '18:20' },
    { id: 2, orderNo: 'OD202608240016', tableNo: 'A02', amount: 24, reason: '重复扣款差额退回', channel: 'alipay', status: 'approved', requestedBy: '李收银', requestedAt: '17:42', handledBy: '张店长', handledAt: '17:58' },
    { id: 3, orderNo: 'OD202608240005', tableNo: 'C01', amount: 6, reason: '米饭未上，取消该行', channel: 'wechat', status: 'done', requestedBy: '李收银', requestedAt: '16:30', handledBy: '张店长', handledAt: '16:40' },
    { id: 4, orderNo: 'OD202608240033', tableNo: '外带', amount: 38, reason: '顾客拿错餐，申请退回', channel: 'cash', status: 'rejected', requestedBy: '李收银', requestedAt: '15:10', handledBy: '张店长', handledAt: '15:22' },
  ])
}

export function fetchRecon(): Promise<ReconRow[]> {
  return delay([
    { id: 1, scope: '订单口径', name: '今日订单应收（线上）', expected: 8642.5, actual: 8642.5, diff: 0, note: '订单表已付金额汇总' },
    { id: 2, scope: '订单口径', name: '今日订单应收（人工）', expected: 1210, actual: 1210, diff: 0, note: '补录单 + 代录单' },
    { id: 3, scope: '支付网关口径', name: '微信支付账单', expected: 3975.6, actual: 3975.6, diff: 0, note: '商户平台今日实收' },
    { id: 4, scope: '支付网关口径', name: '支付宝账单', expected: 2852.05, actual: 2852.05, diff: 0, note: '商家中心今日实收' },
    { id: 5, scope: '收银口径', name: '现金 + POS', expected: 1814.85, actual: 1810, diff: -4.85, note: '现金短款 ¥4.85，待班次清点确认' },
    { id: 6, scope: '渠道口径', name: '微信到账账户', expected: 3975.6, actual: 3975.6, diff: 0, note: '结算账户今日流水' },
    { id: 7, scope: '渠道口径', name: '支付宝到账账户', expected: 2852.05, actual: 2852.05, diff: 0, note: '结算账户今日流水' },
    { id: 8, scope: '财务口径', name: '退款/核销冲减', expected: -82, actual: -82, diff: 0, note: '已通过退款 3 笔' },
  ])
}
