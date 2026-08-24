<script setup lang="ts">
import { computed, ref } from 'vue'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'

const range = ref('今日')
const days = ref(1)
const RANGES = [
  { key: '今日', label: '今日' },
  { key: '昨日', label: '昨日' },
  { key: '本周', label: '本周（周一至今）' },
  { key: '本月', label: '本月（1 日至今）' },
]

// mock 基数：按范围给出（数据诚实：标注为 mock，阶段 2 接真实报表接口）
const BASE: Record<string, { revenue: number; orders: number; refund: number; unsettled: number; avg: number; hours: Array<[string, number, 'lunch' | 'dinner']>; channels: Array<[string, number]> }> = {
  今日: {
    revenue: 8642.5, orders: 96, refund: 82, unsettled: 1286, avg: 90.03,
    hours: [['11:00', 286, 'lunch'], ['12:00', 1105, 'lunch'], ['13:00', 229, 'lunch'], ['17:00', 412, 'dinner'], ['18:00', 1742, 'dinner'], ['19:00', 668, 'dinner']],
    channels: [['微信支付', 3975.6], ['支付宝', 2852.05], ['现金', 1210], ['POS 刷卡', 604.85]],
  },
  昨日: {
    revenue: 7686.2, orders: 88, refund: 40, unsettled: 0, avg: 87.34,
    hours: [['11:00', 255, 'lunch'], ['12:00', 968, 'lunch'], ['13:00', 210, 'lunch'], ['17:00', 380, 'dinner'], ['18:00', 1550, 'dinner'], ['19:00', 590, 'dinner']],
    channels: [['微信支付', 3512.4], ['支付宝', 2490.7], ['现金', 1090], ['POS 刷卡', 593.1]],
  },
  本周: {
    revenue: 52147.9, orders: 612, refund: 356, unsettled: 1286, avg: 85.21,
    hours: [['11:00', 1830, 'lunch'], ['12:00', 7520, 'lunch'], ['13:00', 1680, 'lunch'], ['17:00', 2680, 'dinner'], ['18:00', 11240, 'dinner'], ['19:00', 4620, 'dinner']],
    channels: [['微信支付', 23958], ['支付宝', 17189.2], ['现金', 7260], ['POS 刷卡', 3740.7]],
  },
  本月: {
    revenue: 183740.5, orders: 2146, refund: 1280, unsettled: 1286, avg: 85.6,
    hours: [['11:00', 6610, 'lunch'], ['12:00', 27200, 'lunch'], ['13:00', 6140, 'lunch'], ['17:00', 9680, 'dinner'], ['18:00', 39800, 'dinner'], ['19:00', 16380, 'dinner']],
    channels: [['微信支付', 85120], ['支付宝', 60140.5], ['现金', 25280], ['POS 刷卡', 13200]],
  },
}

const d = computed(() => BASE[range.value])
const maxHour = computed(() => Math.max(1, ...d.value.hours.map(([, v]) => v)))
const channelSum = computed(() => d.value.channels.reduce((s, [, v]) => s + v, 0))
const dayText = computed(() => (range.value === '今日' || range.value === '昨日') ? dayjs().subtract(range.value === '昨日' ? 1 : 0, 'day').format('YYYY-MM-DD') : range.value === '本周' ? `${dayjs().startOf('week').format('MM-DD')} ~ ${dayjs().format('MM-DD')}` : `${dayjs().startOf('month').format('YYYY-MM-DD')} ~ ${dayjs().format('YYYY-MM-DD')}`)

function setRange(k: string) {
  range.value = k
  days.value = k === '今日' || k === '昨日' ? 1 : k === '本周' ? 7 : 30
}

function exportCsv() {
  message.success('报表已导出 CSV（mock）：含分时、渠道、退款明细')
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>营业报表</h1>
        <p>{{ dayText }} · 数据为演示 mock，阶段 2 接真实报表接口</p>
      </div>
      <div class="page-head-right">
        <div class="seg">
          <div v-for="r in RANGES" :key="r.key" :class="{ on: range === r.key }" @click="setRange(r.key)">{{ r.label }}</div>
        </div>
        <a-button @click="exportCsv">⬇ 导出 CSV</a-button>
      </div>
    </div>

    <div class="stats s5">
      <div class="stat"><div class="stat-icon">¥</div><div class="stat-label">营业额（已收）</div><div class="stat-value num">{{ d.revenue.toLocaleString() }}</div><div class="stat-foot">按结账时间归属</div></div>
      <div class="stat"><div class="stat-icon">☰</div><div class="stat-label">订单数</div><div class="stat-value num">{{ d.orders }}<small> 单</small></div><div class="stat-foot">线上 + 人工</div></div>
      <div class="stat"><div class="stat-icon">◎</div><div class="stat-label">客单价</div><div class="stat-value num">{{ d.avg.toFixed(2) }}</div><div class="stat-foot">营业额 / 订单数</div></div>
      <div class="stat err"><div class="stat-icon">↩</div><div class="stat-label">退款/核销</div><div class="stat-value num" style="color:var(--er)">-{{ d.refund.toFixed(0) }}</div><div class="stat-foot">冲减营业额口径</div></div>
      <div class="stat warn"><div class="stat-icon">⏱</div><div class="stat-label">未结账挂账</div><div class="stat-value num" style="color:var(--wn)">{{ d.unsettled.toFixed(2) }}</div><div class="stat-foot">不计入已收营业额</div></div>
    </div>

    <div class="dash-grid">
      <div class="card">
        <div class="card-head"><div><h3>分时营业额</h3><p>单位：元 · 午市 / 晚市</p></div></div>
        <div class="card-body">
          <div class="bars">
            <div v-for="(h, i) in d.hours" :key="i">
              <b class="num">{{ h[1] ? h[1].toLocaleString() : '' }}</b>
              <i :class="{ a: h[2] === 'dinner' }" :style="{ height: Math.max(2, Math.round((h[1] / maxHour) * 100)) + '%' }"></i>
              <span>{{ h[0] }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-head"><div><h3>收款渠道构成</h3><p>合计 ¥{{ channelSum.toLocaleString() }}</p></div></div>
        <div class="card-body">
          <div class="lgd">
            <div v-for="(c, i) in d.channels" :key="i">
              <i :style="{ background: ['#07C160', '#1677FF', '#FAAD14', '#722ED1'][i] }"></i>
              <span>{{ c[0] }}</span>
              <b class="num">¥{{ c[1].toLocaleString() }}<small style="font-size:11px;color:var(--t3)"> ({{ Math.round((c[1] / channelSum) * 100) }}%)</small></b>
            </div>
          </div>
          <div class="hint">微信与支付宝进<b>两个不同结算账户</b>，请分别提现核对（详见对账）。</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dash-grid { display: grid; grid-template-columns: 1fr 380px; gap: 16px; margin-top: 16px }
.hint { margin-top: 16px; padding-top: 12px; border-top: 1px dashed var(--split); font-size: 12px; color: var(--t3) }
</style>
