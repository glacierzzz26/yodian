<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchManualOrders } from '@/api/mock'
import { useAuthStore } from '@/stores/auth'
import type { Order } from '@/types'

const auth = useAuthStore()
const list = ref<Order[]>([])
const state = ref('全部')
const loading = ref(false)
const current = ref<Order | null>(null)

const STATE_META: Record<string, { label: string; color: string }> = {
  pending: { label: '待处理', color: 'red' },
  handling: { label: '处理中', color: 'orange' },
  done: { label: '已处理', color: 'green' },
}
const STATE_KEYS = Object.keys(STATE_META)

function exType(remark: string): string {
  if (remark.includes('重复扣款')) return '重复扣款'
  if (remark.includes('金额不符')) return '金额不符'
  if (remark.includes('支付失败')) return '支付失败'
  return '其他异常'
}

const columns = [
  { title: '订单号', dataIndex: 'no', width: 150 },
  { title: '桌台', dataIndex: 'tableNo', width: 76 },
  { title: '异常类型', key: 'ex', width: 110 },
  { title: '说明', dataIndex: 'remark', ellipsis: true },
  { title: '金额', dataIndex: 'totalAmount', width: 100, align: 'right' as const },
  { title: '提交时间', dataIndex: 'createdAt', width: 130 },
  { title: '状态', dataIndex: 'manualState', width: 96 },
  { title: '操作', key: 'op', width: 120 },
]

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = await fetchManualOrders()
  } catch {
    message.error('加载异常队列失败')
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => (state.value === '全部' ? list.value : list.value.filter((o) => o.manualState === state.value)))

function open(o: Order) {
  current.value = o
}

function setState(o: Order, next: 'pending' | 'handling' | 'done') {
  o.manualState = next
  if (current.value === o) current.value = null
  message.success(`已${STATE_META[next].label}（模拟）`)
}

function onResolve(how: string) {
  const o = current.value
  if (!o) return
  Modal.confirm({
    title: how,
    content: `对订单 ${o.no} ${how === '确认入账' ? '按回调金额确认到账并补差额入账' : '发起原路退差额'}。此操作留痕并计入六口径对账差异说明。`,
    okText: '确认',
    okButtonProps: { danger: how !== '确认入账' },
    onOk() { setState(o, 'done') },
  })
}

function onTakeOver() {
  const o = current.value
  if (!o) return
  Modal.confirm({
    title: '开始处理',
    content: '将该异常单标记为「处理中」，需在当日对账前闭环。',
    okText: '开始处理',
    onOk() { setState(o, 'handling') },
  })
}

function onClose() {
  const o = current.value
  if (!o) return
  Modal.confirm({
    title: '关闭该异常单',
    content: '关闭后计入对账差异说明，需店长本人操作留痕。',
    okText: '关闭', okButtonProps: { danger: true },
    onOk() { o.manualState = 'done'; current.value = null; message.success('已关闭（模拟）') },
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>需人工处理</h1>
        <p>支付回调异常、金额不符等需人工闭环的事项 · 处理动作留痕，计入对账差异</p>
      </div>
      <div class="page-head-right">
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <a-alert v-if="auth.role === 'cashier'" type="info" show-icon style="margin-bottom:16px">
      <template #message>收银员可查看，处理动作（确认入账 / 退差额 / 关闭）需<b>店长</b>操作。发现异常请当面转交店长。</template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <div class="fchips">
          <span :class="{ on: state === '全部' }" @click="state = '全部'">全部 {{ list.length }}</span>
          <span v-for="k in STATE_KEYS" :key="k" :class="{ on: state === k }" @click="state = k">{{ STATE_META[k].label }} {{ list.filter((o) => o.manualState === k).length }}</span>
        </div>

        <a-table :loading="loading" :data-source="filtered" :columns="columns" size="middle"
          :row-key="(r: Order) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'ex'">
              <a-tag :color="(record as Order).remark?.includes('重复扣款') ? 'volcano' : (record as Order).remark?.includes('金额不符') ? 'gold' : 'purple'">{{ exType((record as Order).remark || '') }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'totalAmount'">
              <b class="num">¥{{ (record as Order).totalAmount.toFixed(2) }}</b>
            </template>
            <template v-else-if="column.dataIndex === 'manualState'">
              <a-tag :color="STATE_META[(record as Order).manualState || 'pending'].color">{{ STATE_META[(record as Order).manualState || 'pending'].label }}</a-tag>
            </template>
            <template v-else-if="column.key === 'op'">
              <a-button size="small" type="link" @click.stop="open(record as Order)">详情 / 处理</a-button>
            </template>
          </template>
        </a-table>
      </div>
    </div>

    <!-- 异常单详情 -->
    <a-drawer :open="!!current" :title="`异常单 ${current?.no}`" width="480" @close="current = null">
      <template v-if="current">
        <a-descriptions :column="1" size="small" bordered>
          <a-descriptions-item label="桌台">{{ current.tableNo }} · {{ current.area }}</a-descriptions-item>
          <a-descriptions-item label="异常类型">{{ exType(current.remark || '') }}</a-descriptions-item>
          <a-descriptions-item label="说明">{{ current.remark }}</a-descriptions-item>
          <a-descriptions-item label="金额"><b class="num" style="color:var(--er)">¥{{ current.totalAmount.toFixed(2) }}</b></a-descriptions-item>
          <a-descriptions-item label="提交时间">{{ current.createdAt }}</a-descriptions-item>
          <a-descriptions-item label="状态"><a-tag :color="STATE_META[current.manualState || 'pending'].color">{{ STATE_META[current.manualState || 'pending'].label }}</a-tag></a-descriptions-item>
        </a-descriptions>

        <h3 style="margin:18px 0 8px">菜品明细</h3>
        <div class="ilist">
          <div v-for="it in current.items" :key="it.id" class="irow">
            <span>{{ it.name }}</span><b class="num">×{{ it.qty }}</b><em class="num">¥{{ (it.price * it.qty).toFixed(2) }}</em>
          </div>
        </div>

        <div class="note" v-if="auth.role === 'owner'">
          <b>处理说明</b>
          <p>重复扣款：确认入账并按原路退一单差额；金额不符：以顾客实付金额为准补差额；支付失败：核实支付记录后人工确认到账。</p>
        </div>

        <div v-if="auth.role === 'owner'" style="margin-top:16px;display:flex;gap:8px;justify-content:flex-end;flex-wrap:wrap">
          <template v-if="current.manualState === 'pending'">
            <a-button @click="onTakeOver">开始处理</a-button>
            <a-button @click="onClose" danger>关闭</a-button>
          </template>
          <template v-else-if="current.manualState === 'handling'">
            <a-button type="primary" @click="onResolve('确认入账')">确认入账</a-button>
            <a-button @click="onResolve('原路退差额')" danger>原路退差额</a-button>
          </template>
          <a-tag v-else color="green">已处理 · 只读</a-tag>
        </div>
        <div v-else style="margin-top:16px;color:var(--t3);font-size:13px">仅店长可处理此异常单。</div>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.fchips { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 14px }
.fchips span { height: 28px; padding: 0 12px; border-radius: 14px; border: 1px solid var(--split);
  font-size: 13px; color: var(--t2); cursor: pointer; background: var(--card) }
.fchips span.on { border-color: var(--brand); color: var(--brand); background: var(--brand-bg) }
.ilist { border: 1px solid var(--split); border-radius: 8px; overflow: hidden }
.irow { display: flex; align-items: center; gap: 8px; padding: 9px 12px; border-bottom: 1px dashed var(--split); font-size: 13px }
.irow:last-child { border-bottom: none }
.irow span { flex: 1 }
.irow b { color: var(--t2); font-weight: 500 }
.irow em { font-style: normal; color: var(--t1); font-weight: 600 }
.note { margin-top: 16px; padding: 10px 12px; background: var(--brand-bg); border-radius: 6px; font-size: 12px; color: var(--brand-a); line-height: 1.7 }
.note b { display: block }
</style>
