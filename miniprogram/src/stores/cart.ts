// 购物车草稿（Pinia，9.2）：本会话内点餐草稿。金额仅本地预览展示，实收以服务端订单总额为准。
// 同一 dish + 同一组规格 合并为一行；加菜（prepay 二次下单）复用同一购物车，提交后清空。
import { defineStore } from 'pinia'
import type { DishDTO } from '@/types/api'

export interface CartSpecSelection {
  [groupName: string]: string
}

export interface CartItem {
  dish_id: number
  name: string
  price: number // 分（服务端菜单价，仅展示）
  qty: number
  specs?: CartSpecSelection
  specLabel?: string // 如「中辣 · 大份」，用于列表展示
}

// 规格合并键：同 dish + 同 specs 为同一行。
// 规格名排序后再序列化，避免 {辣度,份量} 与 {份量,辣度} 因键序不同被拆成两行（P1-7a）。
function specKey(item: { dish_id: number; specs?: CartSpecSelection }): string {
  const specs = item.specs
  if (!specs) return item.dish_id + ':null'
  const sorted: CartSpecSelection = {}
  for (const k of Object.keys(specs).sort()) sorted[k] = specs[k]
  return item.dish_id + ':' + JSON.stringify(sorted)
}

interface CartState {
  items: CartItem[]
}

export const useCartStore = defineStore('cart', {
  state: (): CartState => ({ items: [] }),
  getters: {
    totalQty: (s) => s.items.reduce((n, i) => n + i.qty, 0),
    // 本地预览合计（分）；下单实收以服务端 total_amount 为准
    totalPrice: (s) => s.items.reduce((n, i) => n + i.price * i.qty, 0),
    countByDish: (s) => {
      return (dishId: number) => s.items.filter((i) => i.dish_id === dishId).reduce((n, i) => n + i.qty, 0)
    },
  },
  actions: {
    addDish(dish: DishDTO, qty: number, specs?: CartSpecSelection, specLabel?: string) {
      if (qty <= 0) return
      const key = specKey({ dish_id: dish.id, specs })
      const exist = this.items.find((i) => specKey(i) === key)
      if (exist) {
        exist.qty += qty
      } else {
        this.items.push({
          dish_id: dish.id,
          name: dish.name,
          price: dish.price,
          qty,
          ...(specs ? { specs } : {}),
          ...(specLabel ? { specLabel } : {}),
        })
      }
    },
    decDish(dish_id: number, specs?: CartSpecSelection) {
      const key = specKey({ dish_id, specs })
      const idx = this.items.findIndex((i) => specKey(i) === key)
      if (idx < 0) return
      this.items[idx].qty -= 1
      if (this.items[idx].qty <= 0) this.items.splice(idx, 1)
    },
    // P1-7a：菜单步进的「−」是跨规格聚合数量（countByDish），无法反推单一规格 → 减该菜最后一条明细行。
    // 规格菜此前用 decDish(dishId) 会因规格键不匹配而静默失效（数量永远减不掉）。
    decDishLine(dishId: number) {
      let idx = -1
      for (let i = this.items.length - 1; i >= 0; i--) {
        if (this.items[i].dish_id === dishId) {
          idx = i
          break
        }
      }
      if (idx < 0) return
      this.items[idx].qty -= 1
      if (this.items[idx].qty <= 0) this.items.splice(idx, 1)
    },
    removeAt(idx: number) {
      this.items.splice(idx, 1)
    },
    clear() {
      this.items = []
    },
  },
})
