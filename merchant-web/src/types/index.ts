// 与设计文档第 7/16 章接口契约对齐的类型定义（枚举值见设计方案 7.1 枚举表）

export interface ApiResp<T> {
  code: number
  msg: string
  data: T
}

export type Role = 'cashier' | 'kitchen' | 'owner'
export type PayMode = 'prepay' | 'postpay' | 'frontend'
export type PayChannel = 'wechat' | 'alipay' | 'cash' | 'pos' | ''
export type Source = 'online' | 'offline'

export interface Operator {
  id: number
  name: string
  role: Role
  employeeNo: string
}

export interface OrderItem {
  id: number
  name: string
  specs?: string
  remark?: string
  qty: number
  price: number
  itemType: 'dish' | 'per_head'
  isServed: boolean
  isRefunded: boolean
}

export interface Order {
  id: number
  no: string
  tableNo: string
  area: string
  sessionId: number
  payMode: PayMode
  source: Source
  payChannel: PayChannel
  status: string // 见 7.1 枚举表：pending/paid/preparing/served/unsettled/manual/...
  totalAmount: number
  paidAmount: number
  createdAt: string
  items: OrderItem[]
  remark?: string
  station?: string // 出品档口（KDS 用，设计 8 章档口配置）
  manualState?: 'pending' | 'handling' | 'done' // 需人工处理队列状态
  placedAt?: number // 落单时间戳（ms，KDS 计时用，服务端就绪后取 created_at）
}

export interface Dish {
  id: number
  name: string
  category: string
  price: number // 元
  unit: string
  itemType: 'dish' | 'per_head'
  soldOut?: boolean // 沽清（当日）
  status?: 'on' | 'off' // 上架/下架
  specs?: string[]
  station?: string
}

export interface Staff {
  id: number
  name: string
  employeeNo: string
  role: Role
  phone: string
  status: 'active' | 'disabled'
  lastLogin: string
}

export interface OpLog {
  id: number
  time: string
  operator: string
  action: string // 如 登录/结账/核销/改价/开关降级
  target: string
  detail: string
  level: 'info' | 'warn' | 'danger'
}

export interface BackupTask {
  id: number
  time: string
  type: 'full' | 'wal' | 'manual'
  size: string
  status: 'success' | 'running' | 'failed'
  note: string
}

export interface RefundReq {
  id: number
  orderNo: string
  tableNo: string
  amount: number
  reason: string
  channel: PayChannel
  status: 'pending' | 'approved' | 'rejected' | 'done'
  requestedBy: string
  requestedAt: string
  handledBy?: string
  handledAt?: string
}

export interface ReconRow {
  id: number
  scope: string // 六口径之一
  name: string
  expected: number // 应收/账上
  actual: number // 实收/对到
  diff: number // 差额
  note: string
}

export interface TableSessionInfo {
  sessionId: number
  payMode: PayMode
  unsettled: number // 未结账挂账金额
  lockedForBill: boolean // 结账中锁单
  merged: boolean // 并桌
  pax: number
}

export interface TableInfo {
  id: number
  tableNo: string
  area: string
  zone: string
  seats: number
  payMode: PayMode
  status: 'empty' | 'occupied'
  session?: TableSessionInfo
  waitClean?: boolean // 待清台（派生，见设计 3.5）
}

export interface TableArea {
  id: number
  name: string
  zoneLabel: string
  tableLabel: string
  defaultPayMode: PayMode
  tables: TableInfo[]
}

export interface ServiceCall {
  id: number
  tableNo: string
  reason: string
  createdAt: string
  status: 'pending' | 'done'
}

export interface PrintTask {
  id: number
  station: string
  printerSn: string
  kind: string
  status: 'pending' | 'sent' | 'failed'
  retryCount: number
  createdAt: string
}

export interface DashboardHealth {
  name: string
  ok: boolean
  detail: string
}

export interface DashboardStats {
  revenueToday: number
  orderCount: number
  onlineCount: number
  offlineCount: number
  avgPerOrder: number
  unsettledAmount: number
  unsettledTables: number
  manualPending: number
  manualPendingItems: { type: string; count: number }[]
  hourlyRevenue: { time: string; amount: number; session: 'lunch' | 'dinner' }[]
  channelRevenue: { channel: PayChannel; amount: number }[]
  topDishes: { name: string; qty: number; soldOut?: boolean }[]
  health: DashboardHealth[]
}
