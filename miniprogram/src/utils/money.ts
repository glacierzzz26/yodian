// 金额展示（3.2 硬约束：前端不参与金额计算，仅格式化显示服务端下发值）。
// 一律 int64 分 → 元字符串，如 4600 → "46.00"、450 → "4.50"。
export function fen2yuan(fen: number): string {
  const n = Math.trunc(fen || 0)
  const neg = n < 0
  const abs = Math.abs(n)
  const yuan = Math.floor(abs / 100)
  const rest = abs % 100
  return (neg ? '-' : '') + yuan + '.' + String(rest).padStart(2, '0')
}
