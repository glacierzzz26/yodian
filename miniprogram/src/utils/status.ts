// 订单/会话/结账模式 状态文案映射（7.2 状态机 + 7.1 枚举，前端仅展示不改写）
export const ORDER_STATUS: Record<string, string> = {
  // prepay 主干
  pending: '待支付',
  paid: '已支付',
  preparing: '制作中',
  served: '已上菜',
  done: '已完成',
  cancelled: '已取消',
  refunded: '已退款',
  partially_refunded: '部分退款',
  // postpay/frontend 主干
  unsettled: '未结账',
  voided: '已作废',
  written_off: '已核销',
  // 兜底
  manual: '需人工处理',
}

export const PAY_MODE_LABEL: Record<string, string> = {
  prepay: '先付',
  postpay: '后付',
  frontend: '前台结',
}

export const CHANNEL_LABEL: Record<string, string> = {
  wechat: '微信',
  alipay: '支付宝',
}

export function orderStatusLabel(s: string): string {
  return ORDER_STATUS[s] || s
}

export function channelLabel(s: string): string {
  return CHANNEL_LABEL[s] || s
}
