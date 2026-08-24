<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchDishes, fetchOfflineOrders, fetchTableAreas } from '@/api/mock'
import type { Dish, Order, PayChannel, PayMode, TableArea, TableInfo } from '@/types'

const dishes = ref<Dish[]>([])
const areas = ref<TableArea[]>([])
const offline = ref<Order[]>([])
const loading = ref(false)

const mode = ref<PayMode>('prepay')
const selTable = ref<TableInfo | null>(null)
const channel = ref<PayChannel>('cash')
const remark = ref('')
const draft = ref<Array<{ dish: Dish; qty: number }>>([])

onMounted(load)
async function load() {
  loading.value = true
  try {
    const [d, t, o] = await Promise.all([fetchDishes(), fetchTableAreas(), fetchOfflineOrders()])
    dishes.value = d
    areas.value = t
    offline.value = o
  } catch {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

const tableOptions = computed(() =>
  areas.value.flatMap((a) =>
    a.tables.map((t) => ({ label: `${a.name} ${t.tableNo}${t.status === 'occupied' ? '（就餐中）' : ''}${t.payMode === 'postpay' ? '·后付' : ''}`, value: t.id, area: a.name })),
  ),
)

const draftTotal = computed(() => draft.value.reduce((s, l) => s + l.dish.price * l.qty, 0))
const draftQty = computed(() => draft.value.reduce((s, l) => s + l.qty, 0))

function addDish(d: Dish) {
  if (d.soldOut) { message.warning(`「${d.name}」已沽清`); return }
  const hit = draft.value.find((l) => l.dish.id === d.id)
  if (hit) hit.qty += 1
  else draft.value.push({ dish: d, qty: 1 })
}
function minusDish(d: Dish) {
  const hit = draft.value.find((l) => l.dish.id === d.id)
  if (!hit) return
  hit.qty -= 1
  if (hit.qty <= 0) draft.value = draft.value.filter((l) => l.dish.id !== d.id)
}

function submit() {
  if (!draft.value.length) { message.warning('请先添加菜品'); return }
  if (!selTable.value && mode.value !== 'frontend') { message.warning('请选择桌台（或先切到「仅点单」）'); return }
  Modal.confirm({
    title: '确认补录人工单',
    content: `桌台 ${selTable.value ? `${selTable.value.area} ${selTable.value.tableNo}` : '外带/仅点单'} · ${mode.value === 'postpay' ? '后付挂账' : mode.value === 'frontend' ? '仅点单' : '先付'} · ¥${draftTotal.value.toFixed(2)}。补录单进入订单管理并触发打印。`,
    okText: '确认补录',
    onOk() {
      const tableNo = selTable.value ? selTable.value.tableNo : '外带'
      offline.value.unshift({
        id: Date.now(), no: `OD-MAN${String(offline.value.length + 1).padStart(4, '0')}`,
        tableNo, area: selTable.value?.area || '—', sessionId: 0,
        payMode: mode.value, source: 'offline', payChannel: mode.value === 'postpay' ? '' : channel.value,
        status: mode.value === 'postpay' ? 'unsettled' : 'paid',
        totalAmount: draftTotal.value, paidAmount: mode.value === 'postpay' ? 0 : draftTotal.value,
        createdAt: '刚刚', remark: remark.value || '补录人工单',
        items: draft.value.map((l, i) => ({ id: i + 1, name: l.dish.name, qty: l.qty, price: l.dish.price, itemType: l.dish.itemType, isServed: false, isRefunded: false })),
      })
      draft.value = []
      remark.value = ''
      message.success('补录成功（模拟）· 已进入订单管理并触发打印')
    },
  })
}

const offlineCols = [
  { title: '单号', dataIndex: 'no', width: 150 },
  { title: '桌台', dataIndex: 'tableNo', width: 90 },
  { title: '金额', dataIndex: 'totalAmount', width: 100, align: 'right' as const },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '渠道', dataIndex: 'payChannel', width: 80 },
  { title: '时间', dataIndex: 'createdAt', width: 90 },
]
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>补录人工单</h1>
        <p>降级模式 / 网络异常 / 现金收款时的补录入口 · 补录单与线上单统一进入订单流水与对账</p>
      </div>
      <div class="page-head-right">
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <div class="off-grid">
      <!-- 左：补录表单 -->
      <div class="card">
        <div class="card-head"><div><h3>补录单</h3><p>收款方式：后付挂账单在结账时再选，补录阶段不收款</p></div></div>
        <div class="card-body" style="display:flex;flex-direction:column;gap:14px">
          <div>
            <div class="f-l">结算模式</div>
            <a-radio-group v-model:value="mode" size="middle">
              <a-radio-button value="prepay">先付</a-radio-button>
              <a-radio-button value="postpay">后付挂账</a-radio-button>
              <a-radio-button value="frontend">仅点单</a-radio-button>
            </a-radio-group>
          </div>

          <div>
            <div class="f-l">桌台</div>
            <a-select
              v-model:value="selTable" :options="tableOptions" style="width:100%"
              :disabled="mode === 'frontend'"
              placeholder="选择桌台（仅点单可不选）"
              show-search option-filter-prop="label"
            />
          </div>

          <div>
            <div class="f-l">菜品</div>
            <div class="qgrid">
              <a-button v-for="d in dishes" :key="d.id" size="small" :disabled="d.soldOut" @click="addDish(d)">
                {{ d.name }}<em class="num">¥{{ d.price }}</em>
              </a-button>
            </div>
          </div>

          <div>
            <div class="f-l">已选（{{ draftQty }} 件）· 应收 <b class="num" style="color:var(--er)">¥{{ draftTotal.toFixed(2) }}</b></div>
            <div v-if="!draft.length" style="color:var(--t3);font-size:13px">未选菜品</div>
            <div v-for="l in draft" :key="l.dish.id" class="dl">
              <span>{{ l.dish.name }}</span>
              <div class="stepper">
                <a-button size="small" shape="circle" @click="minusDish(l.dish)">−</a-button>
                <b class="num">{{ l.qty }}</b>
                <a-button size="small" shape="circle" type="primary" @click="addDish(l.dish)">+</a-button>
              </div>
              <em class="num">¥{{ (l.dish.price * l.qty).toFixed(2) }}</em>
            </div>
          </div>

          <div>
            <div class="f-l">收款方式</div>
            <a-radio-group v-model:value="channel" :disabled="mode === 'postpay'" size="small">
              <a-radio-button v-for="(label, ch) in ({ cash: '现金', pos: 'POS', wechat: '微信', alipay: '支付宝' } as Record<string, string>)" :key="ch" :value="ch">{{ label }}</a-radio-button>
            </a-radio-group>
          </div>

          <div>
            <div class="f-l">备注</div>
            <a-input v-model:value="remark" placeholder="如：降级模式补录 / 顾客现金支付" />
          </div>

          <a-button type="primary" size="large" @click="submit">确认补录</a-button>
        </div>
      </div>

      <!-- 右：今日人工单 -->
      <div class="card">
        <div class="card-head"><div><h3>今日人工单</h3><p>补录 + 代录合计 {{ offline.length }} 单</p></div></div>
        <div class="card-body">
          <a-table :loading="loading" :data-source="offline" :columns="offlineCols" size="small"
            :row-key="(r: Order) => r.id" :pagination="false">
            <template #bodyCell="{ column, record }">
              <template v-if="column.dataIndex === 'totalAmount'"><b class="num">¥{{ (record as Order).totalAmount.toFixed(2) }}</b></template>
              <template v-else-if="column.dataIndex === 'status'"><a-tag :color="(record as Order).status === 'unsettled' ? 'red' : 'green'">{{ (record as Order).status === 'unsettled' ? '未结账' : '已收' }}</a-tag></template>
              <template v-else-if="column.dataIndex === 'payChannel'">{{ (record as Order).payChannel ? (({ cash: '现金', pos: 'POS', wechat: '微信', alipay: '支付宝' } as Record<string, string>)[(record as Order).payChannel] || (record as Order).payChannel) : '挂账' }}</template>
            </template>
          </a-table>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.off-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; align-items: start }
.f-l { font-size: 13px; color: var(--t2); margin-bottom: 7px; font-weight: 500 }
.qgrid { display: flex; flex-wrap: wrap; gap: 6px }
.qgrid .ant-btn { height: 28px; font-size: 12px }
.qgrid em { font-style: normal; color: var(--t3); margin-left: 6px }
.dl { display: flex; align-items: center; gap: 8px; padding: 7px 0; border-bottom: 1px dashed var(--split); font-size: 13px }
.dl:last-of-type { border-bottom: none }
.dl span { flex: 1 }
.dl em { font-style: normal; font-weight: 600; min-width: 58px; text-align: right }
.stepper { display: flex; align-items: center; gap: 6px }
.stepper b { min-width: 18px; text-align: center }
</style>
