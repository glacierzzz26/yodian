// 订单状态轮询（9.3 决策：3–5s 轮询替代 WS，规避双端 WS 差异）。阶段 3.5 接入订单页/支付页。
// 统一入口：支付后必须轮询定终态（9.8），不做本地乐观更新。
//
// P1-6：轮询耗尽 maxTries 后原先静默 stop → 支付页永久卡在「支付中」。现补 onTimeout 回调
// 由调用方转失败态；stop() 后 start() 可重启（重置 stopped/tries），支撑「重新轮询」。
import { request } from './request'
import type { OrderStatusResp, SessionOrdersResp } from '@/types/api'

export type OrderStatusListener = (s: OrderStatusResp) => void

export interface Poller {
  start: () => void
  stop: () => void
}

export function pollOrderStatus(
  orderId: number,
  listener: OrderStatusListener,
  intervalMs = 3000,
  maxTries = 20,
  onTimeout?: () => void,
): Poller {
  let timer: ReturnType<typeof setTimeout> | null = null
  let tries = 0
  let stopped = false

  const stop = () => {
    stopped = true
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  const tick = async () => {
    if (stopped) return
    tries += 1
    let terminal = false
    try {
      const data = await request<OrderStatusResp>({ url: `/orders/${orderId}/status` })
      listener(data)
      if (data.status === 'paid' || data.status === 'done' || data.status === 'refunded') {
        terminal = true
      }
    } catch {
      // 网络抖动忽略，下一轮继续；超次由 onTimeout 兜底
    }
    if (terminal) {
      stop()
      return
    }
    if (!stopped && tries < maxTries) {
      timer = setTimeout(tick, intervalMs)
    } else {
      stop()
      if (onTimeout) onTimeout()
    }
  }

  // start 可重复调用：已 stopped 则重置计数与标志（P1-6 重启）
  return {
    start: () => {
      if (timer) return
      stopped = false
      tries = 0
      tick()
    },
    stop,
  }
}

// 会话轮询（3.5 结账）：GET /sessions/:sid/orders → data.status，settled 即终态
export function pollSessionSettled(
  sid: number,
  listener: (status: string) => void,
  intervalMs = 3000,
  maxTries = 20,
  onTimeout?: () => void,
): Poller {
  let timer: ReturnType<typeof setTimeout> | null = null
  let tries = 0
  let stopped = false

  const stop = () => {
    stopped = true
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  const tick = async () => {
    if (stopped) return
    tries += 1
    let terminal = false
    try {
      const data = await request<SessionOrdersResp>({ url: `/sessions/${sid}/orders` })
      listener(data.status)
      if (data.status === 'settled') terminal = true
    } catch {
      // 网络抖动忽略
    }
    if (terminal) {
      stop()
      return
    }
    if (!stopped && tries < maxTries) {
      timer = setTimeout(tick, intervalMs)
    } else {
      stop()
      if (onTimeout) onTimeout()
    }
  }

  return {
    start: () => {
      if (timer) return
      stopped = false
      tries = 0
      tick()
    },
    stop,
  }
}
