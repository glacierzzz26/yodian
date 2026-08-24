<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchOrders } from '@/api/mock'
import { useAuthStore } from '@/stores/auth'
import type { Order } from '@/types'

const auth = useAuthStore()
const orders = ref<Order[]>([])
const source = ref('全部')
const status = ref('全部')
const kw = ref('')
const loading = ref(false)
const detail = ref<Order | null>(null)

// 状态 → 展示（对齐设计 7.1 枚举表）
const STATUS_META: Record<string, { label: string; color: string }> = {
  pending: { label: '待支付', color: 'blue' },
  paid: { label: '已支付', color: 'default' },
  preparing: { label: '制作中', color: 'orange' },
  served: { label: '已出餐', color: 'green' },
  unsettled: { label: '未结账', color: 'red' },
  manual: { label: '需人工', color: 'gold' },
}
const STATUS_KEYS = Object.keys(STATUS_META)
const PAY_LABEL: Record<string, string> = { wechat: '微信', alipay: '支付宝', cash: '现金', pos: 'POS' }

function statusDot(k: string): string {
  const map: Record<string, string> = { pending: 'var(--brand)', paid: 'var(--t3)', preparing: 'var(--wn)', served: 'var(--ok)', unsettled: 'var(--er)', manual: '#d48806' }
  return map[k] || 'var(--t3)'
}

const columns = [
  { title: '订单号', dataIndex: 'no', width: 150 },
  { title: '桌台', dataIndex: 'tableNo', width: 80 },
  { title: '来源', dataIndex: 'source', width: 76 },
  { title: '状态', dataIndex: 'status', width: 100 },
  { title: '金额', dataIndex: 'totalAmount', width: 110, align: 'right' as const },
  { title: '支付', dataIndex: 'payChannel', width: 90 },
  { title: '下单时间', dataIndex: 'createdAt', width: 130 },
  { title: '备注', dataIndex: 'remark', ellipsis: true },
]

onMounted(load)
async function load() {
  loading.value = true
  try {
    orders.value = await fetchOrders()
  } catch {
    message.error('加载订单失败')
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  const s = kw.value.trim().toLowerCase()
  return orders.value.filter((o) => {
    if (source.value === '线上' && o.source !== 'online') return false
    if (source.value === '人工' && o.source !== 'offline') return false
    if (status.value !== '全部' && o.status !== status.value) return false
    if (s && !(o.no.toLowerCase().includes(s) || o.tableNo.toLowerCase().includes(s))) return false
    return true
  })
})

const counts = computed(() => {
  const c: Record<string, number> = { 全部: orders.value.length }
  for (const k of STATUS_KEYS) c[k] = orders.value.filter((o) => o.status === k).length
  return c
})

function showDetail(o: Order) {
  detail.value = o
}

function onReprint() {
  Modal.confirm({ title: '补打订单小票', content: `按原单内容重新打印小票（含备注「${detail.value?.remark || '无'}」）。`, okText: '补打', onOk() { message.success('已加入打印队列（模拟）') } })
}

function onWriteOff(o: Order) {
  Modal.confirm({
    title: `核销订单 ${o.no}`,
    content: '仅用于支付回调异常等异常场景，将记录操作人并计入核销报表。',
    okText: '确认核销', okButtonProps: { danger: true },
    onOk() { message.success('已核销（模拟）'); o.status = 'manual' },
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>订单管理</h1>
        <p>共 {{ orders.length }} 单 · 已支付 {{ orders.filter((o) => ['paid', 'preparing', 'served'].includes(o.status)).length }} · 未结账 {{ counts.unsettled || 0 }} · 需人工 {{ counts.manual || 0 }}</p>
      </div>
      <div class="page-head-right">
        <div class="seg">
          <div v-for="v in ['全部', '线上', '人工']" :key="v" :class="{ on: source === v }" @click="source = v">{{ v }}</div>
        </div>
        <a-input v-model:value="kw" placeholder="搜订单号 / 桌号" allow-clear style="width:180px" />
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <div class="card">
      <div class="card-body">
        <div class="fchips">
          <span :class="{ on: status === '全部' }" @click="status = '全部'">全部 {{ counts['全部'] }}</span>
          <span v-for="k in STATUS_KEYS" :key="k" :class="{ on: status === k }" @click="status = k">
            <i class="dot" :style="{ background: statusDot(k) }"></i>
            {{ STATUS_META[k].label }} {{ counts[k] || 0 }}
          </span>
        </div>

        <a-table
          :loading="loading" :data-source="filtered" :columns="columns" size="middle"
          :row-key="(r: Order) => r.id" :pagination="{ pageSize: 10, showSizeChanger: false }"
          @row-click="showDetail"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="STATUS_META[(record as Order).status]?.color || 'default'">{{ STATUS_META[(record as Order).status]?.label || (record as Order).status }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'source'">
              <a-tag :color="(record as Order).source === 'online' ? 'blue' : 'default'">{{ (record as Order).source === 'online' ? '线上' : '人工' }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'totalAmount'">
              <b class="num" :style="{ color: (record as Order).status === 'unsettled' ? 'var(--er)' : 'inherit' }">¥{{ (record as Order).totalAmount.toFixed(2) }}</b>
            </template>
            <template v-else-if="column.dataIndex === 'payChannel'">
              <span v-if="(record as Order).payChannel">{{ PAY_LABEL[(record as Order).payChannel] || (record as Order).payChannel }}</span>
              <span v-else-if="(record as Order).status === 'unsettled'" style="color:var(--wn)">挂账</span>
              <span v-else style="color:var(--t3)">—</span>
            </template>
            <template v-else-if="column.dataIndex === 'remark'">
              <span style="color:var(--t3)">{{ (record as Order).remark || '—' }}</span>
            </template>
          </template>
        </a-table>
      </div>
    </div>

    <!-- 订单详情抽屉 -->
    <a-drawer :open="!!detail" :title="`订单 ${detail?.no}`" width="480" @close="detail = null">
      <template v-if="detail">
        <a-descriptions :column="1" size="small" bordered>
          <a-descriptions-item label="桌台">{{ detail.tableNo }} · {{ detail.area }}</a-descriptions-item>
          <a-descriptions-item label="结账模式">
            <a-tag :color="detail.payMode === 'postpay' ? 'purple' : detail.payMode === 'frontend' ? 'cyan' : 'blue'">{{ detail.payMode === 'postpay' ? '后付统一结账' : detail.payMode === 'frontend' ? '仅点单前台结' : '先付' }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="订单状态"><a-tag :color="STATUS_META[detail.status]?.color || 'default'">{{ STATUS_META[detail.status]?.label || detail.status }}</a-tag></a-descriptions-item>
          <a-descriptions-item label="支付渠道">{{ detail.payChannel ? (PAY_LABEL[detail.payChannel] || detail.payChannel) : '未支付/挂账' }}</a-descriptions-item>
          <a-descriptions-item label="下单时间">{{ detail.createdAt }}</a-descriptions-item>
          <a-descriptions-item v-if="detail.remark" label="备注">{{ detail.remark }}</a-descriptions-item>
        </a-descriptions>

        <h3 style="margin:18px 0 8px">菜品明细</h3>
        <div class="ilist">
          <div v-for="it in detail.items" :key="it.id" class="irow">
            <span>{{ it.name }}<small v-if="it.specs"> · {{ it.specs }}</small><small v-if="it.remark">（{{ it.remark }}）</small></span>
            <b class="num">×{{ it.qty }}</b>
            <em class="num">¥{{ (it.price * it.qty).toFixed(2) }}</em>
          </div>
        </div>

        <div class="itot">
          <span>应收总额</span>
          <b class="num" :style="{ color: detail.status === 'unsettled' ? 'var(--er)' : 'inherit' }">¥{{ detail.totalAmount.toFixed(2) }}</b>
        </div>

        <div style="margin-top:18px;display:flex;gap:8px;justify-content:flex-end">
          <a-button @click="onReprint">补打小票</a-button>
          <a-button v-if="auth.role === 'owner' && detail.status === 'manual'" @click="onWriteOff(detail)" danger>核销订单</a-button>
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.fchips { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 14px }
.fchips span { height: 28px; padding: 0 12px; border-radius: 14px; border: 1px solid var(--split);
  font-size: 13px; color: var(--t2); display: inline-flex; align-items: center; gap: 5px; cursor: pointer; background: var(--card) }
.fchips span.on { border-color: var(--brand); color: var(--brand); background: var(--brand-bg) }
.fchips .dot { width: 6px; height: 6px; border-radius: 50% }
.ilist { border: 1px solid var(--split); border-radius: 8px; overflow: hidden }
.irow { display: flex; align-items: center; gap: 8px; padding: 9px 12px; border-bottom: 1px dashed var(--split); font-size: 13px }
.irow:last-child { border-bottom: none }
.irow span { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap }
.irow span small { color: var(--t3); margin-left: 4px }
.irow b { color: var(--t2); font-weight: 500 }
.irow em { font-style: normal; color: var(--t1); font-weight: 600 }
.itot { display: flex; justify-content: space-between; align-items: center; margin-top: 12px; padding: 12px 4px; font-size: 14px }
.itot b { font-size: 18px }
</style>
