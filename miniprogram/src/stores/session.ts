// 会话/桌台上下文（Pinia）：entry 页解析 tid/sig → 校验菜单 → 存 table/menu。
// 购物车草稿在 stores/cart.ts（阶段 3.4）；本 store 只放「当前桌台会话」事实。
//
// P1-3：tid/sig/sid/hasOrders 持久化到 storage —— 小程序冷启动/切后台回来后仍能从存储水合，
// 免去重新扫码；sid 也随 join 或下单获得。table/menu 不持久化（体积大且会变），由 fetchMenu 重取。
import { defineStore } from 'pinia'
import { request } from '@/utils/request'
import type { MenuResp } from '@/types/api'

// storage key（单桌会话，覆盖写即可）
const K_TID = 'yd_session_tid'
const K_SIG = 'yd_session_sig'
const K_SID = 'yd_session_sid'
const K_HAS_ORDERS = 'yd_session_has_orders'

interface JoinResp {
  session_id: number | null
  table_id: number
  pay_mode: string
  pax: number
  status?: string
}

interface SessionState {
  tid: number | null
  sig: string
  table: MenuResp['table'] | null
  menu: MenuResp | null
  sid: number | null // 本桌会话 id（扫码 join 或下单后获得；拼桌互见/结账/呼叫均需）
  hasOrders: boolean // 本会话是否已成功下单：首单 pax 即开台定数，改人数唯一入口 PATCH /sessions/:sid/pax（16.1）
}

// 从 storage 水合（冷启动恢复；pid 无值返回 null）
function loadNumber(key: string): number | null {
  const v = uni.getStorageSync(key)
  return v === '' || v === undefined || v === null ? null : Number(v)
}

export const useSessionStore = defineStore('session', {
  state: (): SessionState => {
    const tid = loadNumber(K_TID)
    const sid = loadNumber(K_SID)
    return {
      tid,
      sig: uni.getStorageSync(K_SIG) || '',
      table: null,
      menu: null,
      sid,
      hasOrders: uni.getStorageSync(K_HAS_ORDERS) === true,
    }
  },
  actions: {
    setEntry(tid: number, sig: string) {
      this.tid = tid
      this.sig = sig
      uni.setStorageSync(K_TID, tid)
      uni.setStorageSync(K_SIG, sig)
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
    // P1-3 扫码加入本桌会话：登记为会话成员后即得 sid（拼桌互见前提）。
    // 后端返回 session_id=null 表示尚未开台（本桌第一单时才会开台），此时 sid 保持 null，
    // 待首单 createOrder 响应回填。
    async joinTable(): Promise<JoinResp> {
      if (this.tid == null) throw new Error('no tid')
      const resp = await request<JoinResp>({
        url: `/tables/${this.tid}/join?sig=${encodeURIComponent(this.sig)}`,
        method: 'POST',
      })
      if (resp.session_id != null) {
        this.setSid(resp.session_id)
      }
      return resp
    },
    setSid(sid: number) {
      this.sid = sid
      uni.setStorageSync(K_SID, sid)
    },
    // 下单成功标记：记录会话 id + 首单后人数锁定（后续改人数走 PATCH /sessions/:sid/pax）
    markOrdered(sid: number) {
      this.setSid(sid)
      this.hasOrders = true
      uni.setStorageSync(K_HAS_ORDERS, true)
    },
    // P1-4：改就餐人数（16.1 唯一入口）——服务端按人项 qty 与总额重算（3.8）
    async updatePax(pax: number) {
      if (this.sid == null) throw new Error('no sid')
      await request({
        url: `/sessions/${this.sid}/pax`,
        method: 'PATCH',
        data: { pax },
      })
    },
    // 清台/换桌时清理本地会话上下文
    reset() {
      this.table = null
      this.menu = null
      this.sid = null
      this.hasOrders = false
      uni.removeStorageSync(K_SID)
      uni.removeStorageSync(K_HAS_ORDERS)
    },
  },
})
