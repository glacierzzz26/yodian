<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { fetchDashboard } from '@/api/mock'
import type { DashboardStats, PayChannel } from '@/types'

const router = useRouter()
const stats = ref<DashboardStats | null>(null)
const seg = ref('今日')

const CHANNEL_META: Record<PayChannel, { color: string; label: string }> = {
  wechat: { color: '#07C160', label: '微信支付' },
  alipay: { color: '#1677FF', label: '支付宝' },
  cash: { color: '#FAAD14', label: '现金' },
  pos: { color: '#722ED1', label: 'POS 刷卡' },
  '': { color: '#999', label: '未收/挂账' },
}

onMounted(load)
async function load() {
  try {
    stats.value = await fetchDashboard()
  } catch {
    message.error('加载营业数据失败')
  }
}

const maxHourly = computed(() =>
  Math.max(1, ...(stats.value?.hourlyRevenue.map((h) => h.amount) ?? [1])),
)
function hourHeight(v: number) {
  return Math.max(2, Math.round((v / maxHourly.value) * 100))
}

const channelSum = computed(() => stats.value?.channelRevenue.reduce((s, c) => s + c.amount, 0) || 1)
// conic-gradient 分段，按占比生成
const donutBg = computed(() => {
  const parts = stats.value?.channelRevenue ?? []
  let acc = 0
  const segs = parts.map((c) => {
    const from = acc
    acc += (c.amount / channelSum.value) * 100
    return `${CHANNEL_META[c.channel].color} ${from}% ${acc}%`
  })
  return `conic-gradient(${segs.join(',')})`
})

const topDishMax = computed(() => Math.max(1, ...(stats.value?.topDishes.map((d) => d.qty) ?? [1])))
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>营业概览</h1>
        <p>2026-08-24 周日 · 数据每 30 秒刷新</p>
      </div>
      <div class="page-head-right">
        <div class="seg">
          <div v-for="r in ['今日','本周','本月']" :key="r" :class="{ on: seg === r }" @click="seg = r">{{ r }}</div>
        </div>
        <a-button @click="load">↻ 刷新</a-button>
        <a-button type="primary" @click="router.push('/kds')">▤ 打开 KDS 大屏</a-button>
      </div>
    </div>

    <a-alert v-if="stats && stats.unsettledTables > 0" type="warning" show-icon class="dash-alert">
      <template #message>
        <b>{{ stats.unsettledTables }} 张桌未结账，合计 ¥{{ stats.unsettledAmount.toFixed(2) }}</b>
        <span style="margin-left:8px">后付桌台需在清台前完成结账或走店长核销，避免跑单。挂账金额<b>不计入</b>当日已收营业额。</span>
      </template>
      <template #action>
        <a-button size="small" @click="router.push('/tables')">查看桌台</a-button>
      </template>
    </a-alert>

    <div v-if="stats" class="stats s5">
      <div class="stat">
        <div class="stat-icon">¥</div>
        <div class="stat-label">今日已收营业额</div>
        <div class="stat-value num">{{ stats.revenueToday.toLocaleString('zh-CN', { minimumFractionDigits: 2 }) }}</div>
        <div class="stat-foot"><span class="up">↑ 12.4%</span> 较昨日同时段</div>
      </div>
      <div class="stat">
        <div class="stat-icon">☰</div>
        <div class="stat-label">订单数</div>
        <div class="stat-value num">{{ stats.orderCount }}<small> 单</small></div>
        <div class="stat-foot">线上 {{ stats.onlineCount }} · 人工 {{ stats.offlineCount }}</div>
      </div>
      <div class="stat">
        <div class="stat-icon">◎</div>
        <div class="stat-label">客单价</div>
        <div class="stat-value num">{{ stats.avgPerOrder.toFixed(2) }}</div>
        <div class="stat-foot"><span class="dn">↓ 2.1%</span> 较昨日</div>
      </div>
      <div class="stat warn">
        <div class="stat-icon">⏱</div>
        <div class="stat-label">未结账挂账</div>
        <div class="stat-value num" style="color:var(--wn)">{{ stats.unsettledAmount.toFixed(2) }}</div>
        <div class="stat-foot">{{ stats.unsettledTables }} 桌 · 不计入营业额</div>
      </div>
      <div class="stat err">
        <div class="stat-icon">⚠</div>
        <div class="stat-label">需人工处理</div>
        <div class="stat-value num" style="color:var(--er)">{{ stats.manualPending }}<small> 单</small></div>
        <div class="stat-foot">{{ stats.manualPendingItems.map((i) => `${i.type} ${i.count}`).join(' · ') }}</div>
      </div>
    </div>

    <div v-if="stats" class="dash-grid">
      <div class="card">
        <div class="card-head">
          <div><h3>分时营业额</h3><p>单位：元 · 营业时段 11:00–14:00 / 17:00–21:30</p></div>
          <div class="card-head-right">
            <span class="tag b"><i class="dot" style="background:var(--brand)"></i>午市</span>
            <span class="tag g"><i class="dot" style="background:var(--ok)"></i>晚市</span>
          </div>
        </div>
        <div class="card-body">
          <div class="bars">
            <div v-for="h in stats.hourlyRevenue" :key="h.time">
              <b class="num">{{ h.amount ? h.amount.toLocaleString() : '' }}</b>
              <i :class="{ a: h.session === 'dinner' }" :style="{ height: hourHeight(h.amount) + '%' }"></i>
              <span>{{ h.time }}</span>
            </div>
          </div>
          <div class="hint">峰值 18:00–18:30，出餐平均 <b>9 分 12 秒</b>。营业额按<b>结账时间</b>归属，跨市挂账单计入结账所在时段。</div>
        </div>
      </div>

      <div class="card">
        <div class="card-head">
          <div><h3>收款渠道构成</h3><p>今日已收 · 六口径之一</p></div>
        </div>
        <div class="card-body">
          <div style="display:flex;gap:20px;align-items:center">
            <div class="donut" :style="{ background: donutBg }">
              <div class="donut-c"><b class="num">¥{{ channelSum.toLocaleString() }}</b><small>{{ stats.orderCount }} 单</small></div>
            </div>
            <div class="lgd">
              <div v-for="c in stats.channelRevenue" :key="c.channel">
                <i :style="{ background: CHANNEL_META[c.channel].color }"></i>
                <span>{{ CHANNEL_META[c.channel].label }}</span>
                <b class="num">{{ c.amount.toFixed(2) }}</b>
              </div>
            </div>
          </div>
          <div class="two-acc">ⓘ 微信与支付宝资金进<b>两个不同账户</b>，需分别提现、分别核对。</div>
        </div>
      </div>
    </div>

    <div v-if="stats" class="dash-grid two">
      <div class="card">
        <div class="card-head">
          <div><h3>热销菜品 TOP 6</h3></div>
          <div class="card-head-right"><a-button type="link" size="small" @click="router.push('/soldout')">沽清管理 →</a-button></div>
        </div>
        <div class="card-body" style="display:flex;flex-direction:column;gap:14px">
          <div v-for="d in stats.topDishes" :key="d.name">
            <div style="display:flex;justify-content:space-between;font-size:13px;margin-bottom:5px">
              <span :style="d.soldOut ? { color: 'var(--er)' } : {}">{{ d.name }}{{ d.soldOut ? '（已沽清）' : '' }}</span>
              <b class="num">{{ d.qty }} 份</b>
            </div>
            <div class="prog"><i :style="{ width: Math.max(8, Math.round(d.qty / topDishMax * 100)) + '%', background: d.soldOut ? 'var(--er)' : undefined }"></i></div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-head">
          <div><h3>系统健康</h3></div>
          <div class="card-head-right"><span class="tag g">整体正常</span></div>
        </div>
        <div class="card-body" style="display:flex;flex-direction:column;gap:11px">
          <div v-for="h in stats.health" :key="h.name" class="prt" :class="h.ok ? 'ok' : 'err'">
            <u>{{ h.ok ? '✓' : '✕' }}</u>
            <div class="prt-i"><b>{{ h.name }}</b><small>{{ h.detail }}</small></div>
            <a-button v-if="!h.ok" size="small" danger @click="router.push('/print')">去补打</a-button>
            <span v-else class="tag g">正常</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dash-alert { margin-bottom: 16px }
.dash-grid { display: grid; grid-template-columns: 1fr 380px; gap: 16px; margin-top: 16px }
.dash-grid.two { grid-template-columns: 1fr 1fr }
.hint { margin-top: 14px; padding-top: 12px; border-top: 1px dashed var(--split); font-size: 12px; color: var(--t3) }
.two-acc { margin-top: 16px; padding: 10px 12px; background: var(--brand-bg); border-radius: 6px; font-size: 12px; color: var(--brand-a) }
</style>
