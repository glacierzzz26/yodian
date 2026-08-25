// 会话/桌台上下文（Pinia）：entry 页解析 tid/sig → 校验菜单 → 存 table/menu。
// 购物车草稿在 stores/cart.ts（阶段 3.4）；本 store 只放「当前桌台会话」事实。
import { defineStore } from 'pinia'
import { request } from '@/utils/request'
import type { MenuResp } from '@/types/api'

interface SessionState {
  tid: number | null
  sig: string
  table: MenuResp['table'] | null
  menu: MenuResp | null
  sid: number | null // 本桌会话 id（下单后获得；拼桌互见/结账/呼叫均需）
  hasOrders: boolean // 本会话是否已成功下单：首单 pax 即开台定数，改人数唯一入口 PATCH /sessions/:sid/pax（16.1）
}

export const useSessionStore = defineStore('session', {
  state: (): SessionState => ({ tid: null, sig: '', table: null, menu: null, sid: null, hasOrders: false }),
  actions: {
    setEntry(tid: number, sig: string) {
      this.tid = tid
      this.sig = sig
    },
    // 校验桌码签名 + 拉菜单：sig 无效 → 后端 20001 → reject(BizError) → entry 显示无效屏
    async fetchMenu(): Promise<MenuResp> {
      const menu = await request<MenuResp>({
        url: `/tables/${this.tid}/menu?sig=${encodeURIComponent(this.sig)}`,
      })
      this.table = menu.table
      this.menu = menu
      return menu
    },
    // 下单成功标记：记录会话 id + 首单后人数锁定（后续改人数走 PATCH /sessions/:sid/pax）
    markOrdered(sid: number) {
      this.sid = sid
      this.hasOrders = true
    },
  },
})
