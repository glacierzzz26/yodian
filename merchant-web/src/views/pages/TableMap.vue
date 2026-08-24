<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { fetchTableAreas } from '@/api/mock'
import { useAuthStore } from '@/stores/auth'
import type { TableArea, TableInfo } from '@/types'

const router = useRouter()
const auth = useAuthStore()
const areas = ref<TableArea[]>([])
const areaFilter = ref('全部区域')
const view = ref('grid')
const current = ref<TableInfo | null>(null)

// 列表视图列定义（与详情弹窗信息一致）
const tableCols = [
  { title: '桌号', dataIndex: 'tableNo', width: 80 },
  { title: '座位', dataIndex: 'seats', width: 60 },
  { title: '模式', dataIndex: 'payMode', width: 80, customRender: ({ text }: { text: string }) => (text === 'postpay' ? '后付' : text === 'frontend' ? '仅点单' : '先付') },
  { title: '状态', dataIndex: 'status', width: 90, customRender: ({ text }: { text: string }) => (text === 'occupied' ? '就餐中' : '空闲') },
  { title: '待收金额', dataIndex: ['session', 'unsettled'], width: 100, customRender: ({ text }: { text?: number }) => (text ? `¥${text.toFixed(2)}` : '—') },
]

onMounted(load)
async function load() {
  try {
    areas.value = await fetchTableAreas()
  } catch {
    message.error('加载桌台失败')
  }
}

const filtered = computed(() =>
  areas.value.filter((a) => areaFilter.value === '全部区域' || a.name === areaFilter.value),
)

const summary = computed(() => {
  const all = areas.value.flatMap((a) => a.tables)
  const occupied = all.filter((t) => t.status === 'occupied').length
  const unsettled = all.filter((t) => t.session && t.session.unsettled > 0).length
  const waitClean = all.filter((t) => t.waitClean).length
  const empty = all.length - occupied
  return { total: all.length, occupied, unsettled, empty, waitClean }
})

function cellClass(t: TableInfo) {
  if (t.status === 'empty') return 'top'
  if (t.session?.unsettled && t.session.unsettled > 0) return 'unsettled'
  if (t.waitClean) return 'waitclean'
  if (t.session?.merged) return 'merged'
  return 'occ'
}

function isOwner() {
  return auth.role === 'owner'
}

function openDetail(t: TableInfo) {
  current.value = t
}

function onSettle() {
  if (!current.value) return
  Modal.confirm({
    title: `对 ${current.value.tableNo} 发起结账`,
    content: `生成会话结账单，金额由服务端聚合计算。${isOwner() ? '可代客结账（现金/POS/扫顾客付款码）。' : ''}`,
    okText: '发起结账',
    cancelText: '取消',
    onOk() {
      message.success(`已对 ${current.value!.tableNo} 生成结账单（模拟）`)
      current.value = null
    },
  })
}

function onCloseTable() {
  if (!current.value) return
  const unsettled = current.value.session?.unsettled || 0
  if (unsettled > 0) {
    Modal.warning({
      title: '禁止直接清台',
      content: `本桌仍有未结账金额 ¥${unsettled.toFixed(2)}，须先完成结账，或由店长「跑单/免单核销」后清台。`,
    })
    return
  }
  Modal.confirm({ title: '清台', content: '确认结束会话并清台？', okText: '清台', onOk() { message.success('已清台（模拟）'); current.value = null } })
}

function onWriteOff() {
  if (!current.value) return
  Modal.confirm({
    title: '跑单/免单核销',
    content: '选择原因将留痕（记录操作人 + 原因，计入核销报表）。',
    okText: '确认核销',
    okButtonProps: { danger: true },
    onOk() { message.success('已核销并清台（模拟）'); current.value = null },
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>桌台图</h1>
        <p v-if="areas.length">共 {{ summary.total }} 桌 · 就餐中 {{ summary.occupied }} · 未结账 {{ summary.unsettled }} · 空闲 {{ summary.empty }} · 待清台 {{ summary.waitClean }}</p>
      </div>
      <div class="page-head-right">
        <div class="seg">
          <div v-for="a in ['全部区域', ...new Set(areas.map((x) => x.name))]" :key="a" :class="{ on: areaFilter === a }" @click="areaFilter = a">{{ a }}</div>
        </div>
        <a-button @click="view = view === 'grid' ? 'list' : 'grid'">☰ {{ view === 'grid' ? '列表视图' : '卡片视图' }}</a-button>
        <a-button type="primary" @click="router.push('/pos')">✎ 代客点单</a-button>
      </div>
    </div>

    <div class="card" style="margin-bottom:16px">
      <div class="card-body" style="padding:13px 20px">
        <div class="tlgd">
          <div><i style="background:var(--bd)"></i>空闲</div>
          <div><i style="background:var(--brand)"></i>就餐中（已结账/先付）</div>
          <div><i style="background:var(--er)"></i>未结账（后付挂账）</div>
          <div><i style="background:var(--wn)"></i>待清台</div>
          <div><i style="background:var(--cy)"></i>并桌</div>
          <div style="margin-left:auto;color:var(--t3)">🔒 = 结账中锁定，禁止加菜</div>
        </div>
      </div>
    </div>

    <div v-for="a in filtered" :key="a.id" class="zone">
      <div class="zone-head">
        <h3>{{ a.name }}</h3>
        <span class="zl">{{ a.tables.length }} 桌</span>
        <div class="zr">
          <span class="tag" :class="a.defaultPayMode === 'postpay' ? 'p' : 'd'">默认{{ a.defaultPayMode === 'postpay' ? '后付' : '先付' }}</span>
        </div>
      </div>

      <template v-if="view === 'grid'">
        <div class="tgrid">
          <div v-for="t in a.tables" :key="t.id" class="tcell" :class="cellClass(t)" @click="openDetail(t)">
            <div class="tno">
              {{ t.tableNo }}
              <span v-if="t.session?.lockedForBill" class="lock" title="结账中锁定，禁止加菜">🔒</span>
              <span v-if="t.session?.merged" class="lock" style="color:var(--cy)" title="并桌">⊕</span>
            </div>
            <div class="tmeta">{{ t.seats }} 位 · {{ t.payMode === 'postpay' ? '后付' : t.payMode === 'frontend' ? '仅点单' : '先付' }}{{ t.session ? ` · ${t.session.pax} 人` : '' }}</div>
            <div class="tamt">
              <span>{{ t.status === 'empty' ? '空台' : t.session?.unsettled ? '待收' : '就餐中' }}</span>
              <b v-if="t.session?.unsettled" class="num">¥{{ t.session.unsettled.toFixed(2) }}</b>
            </div>
          </div>
        </div>
      </template>

      <div v-else class="card">
        <a-table size="small" :columns="tableCols" :data-source="a.tables" :pagination="false" :row-key="(r: TableInfo) => r.id" @row-click="openDetail" />
      </div>
    </div>

    <!-- 桌台详情弹窗 -->
    <a-modal :open="!!current" :title="`${current?.tableNo} · 桌台详情`" :footer="null" @cancel="current = null" width="420">
      <template v-if="current">
        <a-descriptions :column="1" size="small" bordered>
          <a-descriptions-item label="区域">{{ current.area }}</a-descriptions-item>
          <a-descriptions-item label="座位数">{{ current.seats }}</a-descriptions-item>
          <a-descriptions-item label="结账模式">
            <a-tag :color="current.payMode === 'postpay' ? 'purple' : 'blue'">{{ current.payMode === 'postpay' ? '后付统一结账' : current.payMode === 'frontend' ? '仅点单前台结' : '先付后吃' }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="current.status === 'occupied' ? 'blue' : 'default'">{{ current.status === 'occupied' ? '就餐中' : '空闲' }}</a-tag>
          </a-descriptions-item>
          <template v-if="current.session">
            <a-descriptions-item label="会话 #">{{ current.session.sessionId }}</a-descriptions-item>
            <a-descriptions-item label="就餐人数">{{ current.session.pax }} 人</a-descriptions-item>
            <a-descriptions-item label="未结账金额"><b style="color:var(--er)" class="num">¥{{ current.session.unsettled.toFixed(2) }}</b></a-descriptions-item>
            <a-descriptions-item label="结账中锁单">{{ current.session.lockedForBill ? '是' : '否' }}</a-descriptions-item>
          </template>
        </a-descriptions>

        <div style="margin-top:16px;display:flex;gap:8px;justify-content:flex-end;flex-wrap:wrap">
          <a-button v-if="current.session" type="primary" @click="onSettle">结账</a-button>
          <a-button v-if="isOwner()" @click="onWriteOff" danger>跑单/免单核销</a-button>
          <a-button @click="onCloseTable">清台</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<!-- 桌台图样式由 global.css「桌台图」段统一持有（.zone/.tgrid/.tcell 各状态） -->
<style scoped></style>
