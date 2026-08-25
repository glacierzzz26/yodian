<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { fetchKdsOrders, changeOrderStatus, kdsMove } from '@/api/mock'
import { useAuthStore } from '@/stores/auth'
import type { Order } from '@/types'

const router = useRouter()
const auth = useAuthStore()
const orders = ref<Order[]>([])
const station = ref('全部')
const filter = ref<'new' | 'cooking' | 'done'>('new')
const now = ref(Date.now())
const st = ref<Record<number, 'new' | 'cooking' | 'done'>>({})

let clock: number
let poll: number

onMounted(async () => {
  await load()
  clock = window.setInterval(() => { now.value = Date.now() }, 1000)
  poll = window.setInterval(load, 15000) // KDS 15s 自动拉新单
})
onUnmounted(() => {
  window.clearInterval(clock)
  window.clearInterval(poll)
})

async function load() {
  try {
    const list = await fetchKdsOrders()
    const base = Date.now()
    orders.value = list.map((o, i) => ({
      ...o,
      placedAt: base - (i + 1) * 75_000, // mock：模拟逐单落单时间
    }))
    for (const o of orders.value) if (st.value[o.id] === undefined) st.value[o.id] = 'new'
  } catch {
    message.error('拉取后厨单失败')
  }
}

function stateOf(o: Order): 'new' | 'cooking' | 'done' {
  return st.value[o.id] || 'new'
}
function elapsed(o: Order): number {
  return Math.floor((now.value - (o.placedAt ?? now.value)) / 1000)
}
function fmtElapsed(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}
function isTimeout(o: Order): boolean {
  return stateOf(o) !== 'done' && elapsed(o) > 15 * 60
}

const stations = computed(() => {
  const s = new Set(orders.value.map((o) => o.station).filter(Boolean))
  return ['全部', ...s] as string[]
})

const shown = computed(() =>
  orders.value.filter((o) => {
    if (station.value !== '全部' && o.station !== station.value) return false
    if (filter.value !== stateOf(o)) return false
    return true
  }),
)
const counts = computed(() => {
  const c: Record<string, number> = { new: 0, cooking: 0, done: 0 }
  for (const o of orders.value) c[stateOf(o)]++
  return c
})

// 状态写入口按角色分流：后厨走 KDS 卡片流转 /kds/:id/move（16.2），owner/cashier 走收银台改状态。
// 后付单状态机（7.2 B）无 preparing/served：仅本地标记，不写后端。
async function startCooking(o: Order) {
  if (o.payMode === 'postpay') {
    st.value[o.id] = 'cooking'
    message.info(`已开始制作（后付单无后端状态，仅本地标记）· ${o.no}`)
    return
  }
  try {
    if (auth.role === 'kitchen') await kdsMove(o.id, 'preparing')
    else await changeOrderStatus(o.id, 'preparing')
    st.value[o.id] = 'cooking'
    message.success(`已开始制作 · ${o.no}`)
  } catch (e) {
    message.error((e as Error).message || '更新制作状态失败')
  }
}
async function finish(o: Order) {
  if (o.payMode === 'postpay') {
    st.value[o.id] = 'done'
    message.info(`已出餐（后付单无后端状态，仅本地标记）· ${o.no}`)
    return
  }
  try {
    if (auth.role === 'kitchen') await kdsMove(o.id, 'served')
    else await changeOrderStatus(o.id, 'served')
    st.value[o.id] = 'done'
    message.success(`已出餐 · ${o.no}`)
  } catch (e) {
    message.error((e as Error).message || '更新出餐状态失败')
  }
}

function clockText(): string {
  const d = new Date(now.value)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
function toggleFullscreen() {
  if (document.fullscreenElement) document.exitFullscreen()
  else document.documentElement.requestFullscreen?.()
}
</script>

<template>
  <div class="kds">
    <!-- 顶部状态条 -->
    <header class="kh">
      <div class="kl">
        <i class="logo">◧</i>
        <div>
          <b>悦点 · 后厨大屏（KDS）</b>
          <span>自动刷新 15s · 新单蜂鸣提醒（印刷小票同步）</span>
        </div>
      </div>
      <div class="kr">
        <span class="chip on"><i></i>晚市营业中</span>
        <span class="chip">新单 <b>{{ counts.new }}</b></span>
        <span class="chip">制作中 <b>{{ counts.cooking }}</b></span>
        <span class="chip dim">已出 <b>{{ counts.done }}</b></span>
        <span class="clock">{{ clockText() }}</span>
        <button class="kb" @click="toggleFullscreen">⛶ 全屏</button>
        <button class="kb" @click="router.push('/')">↩ 返回商家端</button>
      </div>
    </header>

    <!-- 筛选栏 -->
    <nav class="kf">
      <div class="st-tabs">
        <button v-for="s in stations" :key="s" :class="{ on: station === s }" @click="station = s">{{ s }}</button>
      </div>
      <div class="st-tabs right">
        <button :class="{ on: filter === 'new' }" @click="filter = 'new'">新单 {{ counts.new }}</button>
        <button :class="{ on: filter === 'cooking' }" @click="filter = 'cooking'">制作中 {{ counts.cooking }}</button>
        <button :class="{ on: filter === 'done' }" @click="filter = 'done'">已完成 {{ counts.done }}</button>
      </div>
    </nav>

    <!-- 订单卡片 -->
    <main class="kc">
      <div v-if="!shown.length" class="kempty">当前没有{{ filter === 'new' ? '新单' : filter === 'cooking' ? '制作中' : '已完成' }}订单</div>
      <div v-for="o in shown" :key="o.id" class="kcard" :class="[{ time: isTimeout(o) }, stateOf(o)]">
        <div class="kt">
          <div class="kt-l">
            <b>{{ o.tableNo }}</b>
            <span class="kbz">{{ o.payMode === 'postpay' ? '后付' : '先付' }}</span>
          </div>
          <div class="kt-r">
            <em class="el num">{{ fmtElapsed(elapsed(o)) }}</em>
            <small>已过时长</small>
          </div>
        </div>
        <div class="kno">#{{ o.no.slice(-6) }} · {{ o.area }}</div>
        <ul class="ki">
          <li v-for="it in o.items" :key="it.id">
            <span>{{ it.name }}<small v-if="it.specs"> · {{ it.specs }}</small></span>
            <b class="num">×{{ it.qty }}</b>
          </li>
        </ul>
        <div v-if="o.remark" class="krm">⚠ {{ o.remark }}</div>
        <div v-if="isTimeout(o)" class="ktime">⏰ 已超时（>15 分钟）</div>
        <div class="kop">
          <button v-if="stateOf(o) === 'new'" class="b-ok" @click="startCooking(o)">开始制作</button>
          <button v-if="stateOf(o) === 'cooking'" class="b-ok" @click="finish(o)">✓ 完成出餐</button>
          <span v-if="stateOf(o) === 'done'" class="tag-done">✓ 已出餐</span>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.kds { min-height: 100vh; background: #0d1117; color: #e6edf3; display: flex; flex-direction: column; font-family: -apple-system, 'PingFang SC', 'Microsoft YaHei', sans-serif }
/* 顶部 */
.kh { height: 64px; background: #161b22; border-bottom: 1px solid #21262d; display: flex; align-items: center; padding: 0 24px; gap: 16px }
.kl { display: flex; align-items: center; gap: 12px }
.logo { width: 36px; height: 36px; border-radius: 8px; background: linear-gradient(135deg,#1677ff,#0958d9); display: grid; place-items: center; font-style: normal; font-size: 18px }
.kl b { font-size: 16px; display: block }
.kl span { font-size: 12px; color: #8b949e }
.kr { margin-left: auto; display: flex; align-items: center; gap: 10px }
.chip { font-size: 12px; padding: 5px 12px; border-radius: 14px; background: #21262d; color: #c9d1d9 }
.chip i { display: inline-block; width: 7px; height: 7px; border-radius: 50%; background: #3fb950; margin-right: 5px; animation: blink 1.6s infinite }
.chip.on { background: rgba(63,185,80,.15); color: #3fb950; border: 1px solid rgba(63,185,80,.4) }
.chip.dim { color: #8b949e }
.chip b { color: #f0f6fc }
@keyframes blink { 0%,100% { opacity: 1 } 50% { opacity: .35 } }
.clock { font-size: 22px; font-weight: 600; font-variant-numeric: tabular-nums; color: #f0f6fc; margin-left: 6px }
.kb { height: 32px; padding: 0 14px; border: 1px solid #30363d; border-radius: 6px; background: #21262d; color: #c9d1d9; cursor: pointer; font-size: 13px }
.kb:hover { background: #30363d; color: #f0f6fc }
/* 筛选 */
.kf { height: 52px; background: #161b22; border-bottom: 1px solid #21262d; display: flex; align-items: center; padding: 0 24px; gap: 12px }
.st-tabs { display: flex; gap: 8px }
.st-tabs button { height: 32px; padding: 0 16px; border: 1px solid #30363d; border-radius: 16px; background: transparent; color: #8b949e; cursor: pointer; font-size: 13px }
.st-tabs button.on { background: #1677ff; border-color: #1677ff; color: #fff }
.st-tabs.right { margin-left: auto }
/* 卡片 */
.kc { flex: 1; padding: 20px 24px; display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 16px; align-content: start; overflow-y: auto }
.kempty { grid-column: 1 / -1; color: #8b949e; text-align: center; padding: 60px 0; font-size: 15px }
.kcard { background: #161b22; border: 1px solid #30363d; border-radius: 12px; padding: 16px; position: relative }
.kcard.new { border-top: 4px solid #58a6ff }
.kcard.cooking { border-top: 4px solid #d29922 }
.kcard.done { border-top: 4px solid #3fb950; opacity: .62 }
.kcard.time { border: 2px solid #f85149; box-shadow: 0 0 0 3px rgba(248,81,73,.15) }
.kt { display: flex; align-items: flex-start; justify-content: space-between }
.kt-l b { font-size: 26px; font-weight: 700 }
.kbz { font-size: 11px; padding: 2px 8px; border-radius: 4px; background: #21262d; color: #8b949e; margin-left: 6px; vertical-align: 4px }
.kt-r { text-align: right }
.el { font-size: 20px; font-weight: 700; color: #58a6ff; font-variant-numeric: tabular-nums }
.kt-r small { font-size: 11px; color: #8b949e; display: block }
.kno { font-size: 12px; color: #8b949e; margin: 2px 0 10px }
.ki { list-style: none; margin: 0; padding: 0; border-top: 1px dashed #30363d }
.ki li { display: flex; justify-content: space-between; padding: 8px 0; font-size: 15px }
.ki li span small { color: #d29922; font-size: 12px }
.ki li b { color: #f0f6fc }
.krm { margin-top: 6px; padding: 7px 9px; border-radius: 6px; background: rgba(210,153,34,.12); color: #d29922; font-size: 12px }
.ktime { margin-top: 6px; padding: 7px 9px; border-radius: 6px; background: rgba(248,81,73,.14); color: #f85149; font-size: 12px; font-weight: 600 }
.kop { margin-top: 12px; display: flex; justify-content: flex-end }
.b-ok { height: 34px; padding: 0 18px; border: none; border-radius: 8px; background: #238636; color: #fff; font-size: 14px; cursor: pointer; font-weight: 500 }
.b-ok:hover { background: #2ea043 }
.tag-done { font-size: 13px; color: #3fb950 }
</style>
