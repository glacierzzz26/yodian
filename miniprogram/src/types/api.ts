// 后端信封与 DTO 对齐（16.1 信封唯一权威；金额一律 int64 分，前端只展示不算价）。
// 与 server 端契约保持一致，字段名即 JSON 字段名。

export interface ApiResp<T = unknown> {
  code: number
  msg: string
  data: T
}

export interface LoginResp {
  token: string
  customer_id: number
  channel: string
}

export interface TableInfo {
  id: number
  table_no: string
  seats: number
  pay_mode: 'prepay' | 'postpay' | 'frontend'
}

export interface DishDTO {
  id: number
  name: string
  price: number // 分
  is_sold_out: boolean
  image_url?: string | null
  specs?: unknown
}

export interface CategoryDTO {
  id: number
  name: string
  dishes: DishDTO[]
}

export interface PerHeadDTO {
  id: number
  name: string
  price: number // 分
  is_required: boolean
  is_default_checked: boolean
}

export interface MenuResp {
  table: TableInfo
  categories: CategoryDTO[]
  per_head: PerHeadDTO[]
}

export interface CreateOrderResp {
  order_id: number
  out_trade_no: string
  total_amount: number // 分
  status: string
  pay_mode: string
  session_id: number
}

export interface OrderStatusResp {
  order_id: number
  session_id?: number | null
  status: string
  pay_mode: string
  total_amount: number
  paid_amount?: number | null
  pay_channel?: string | null
  created_at: string
  payments: Array<{ channel: string; amount: number; status: string; created_at: string }>
  refunds: Array<{ channel: string; amount: number; status: string; created_at: string }>
}

// POST /pay/prepay 响应：{out_trade_no, params, amount}（bill_id 为后付结账单路径，3.5）
export interface PrepayResp {
  out_trade_no: string
  params: Record<string, unknown> // mock: {mock:true, channel, out_trade_no, amount, subject, order_id/bill_id}
  amount: number // 分
  bill_id?: number
}

// POST /pay/switch {order_id, to}：换渠道重付（时序图 4，3.5）
export interface SwitchResp {
  out_trade_no: string
  params: Record<string, unknown>
  amount: number
  to: string
}

// ---------- 会话订单（拼桌互见，3.5） ----------
export interface SessionOrderItem {
  name: string
  item_type: 'dish' | 'per_head'
  price: number // 分
  qty: number
  specs?: unknown
  remark?: string | null
}
export interface SessionOrderDTO {
  id: number
  customer_id?: number | null
  channel?: string | null
  status: string
  total_amount: number
  source: string
  created_at: string
  items: SessionOrderItem[]
}
export interface SessionOrdersResp {
  session_id: number
  table_id: number
  pay_mode: string
  pax: number
  status: 'active' | 'settled'
  orders: SessionOrderDTO[]
}

// ---------- 结账（3.5） ----------
export interface PerHeadPreview {
  id: number
  name: string
  price: number
  qty: number
  is_required: boolean
  is_default: boolean
}
export interface BillPreviewResp {
  session_id: number
  pay_mode: string
  pax: number
  orders_count: number
  dishes_amount: number
  per_head_items: PerHeadPreview[]
  per_head_amount: number
  total: number
}
export interface CreateBillResp {
  bill_id: number
  session_id: number
  out_trade_no: string
  total_amount: number
  status: string
}
export interface CancelBillResp {
  bill_id: number
  status: string
  session_id: number
}

// ---------- 呼叫服务员 / 手机号（3.5） ----------
export interface CallWaiterResp {
  call_id: number
  status: string
  session_id: number
}
export interface PhoneCodeResp {
  sent: boolean
  dev_code?: string
}
export interface PhoneBindResp {
  phone: string
  phone_verified_at: string
}
