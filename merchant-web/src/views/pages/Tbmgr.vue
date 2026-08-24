<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchTableAreas } from '@/api/mock'
import type { TableArea, TableInfo, PayMode } from '@/types'

interface FlatRow { area: TableArea; table: TableInfo }

const areas = ref<TableArea[]>([])
const loading = ref(false)
const modalOpen = ref(false)
const editing = ref<{ areaName: string; table: TableInfo } | null>(null)

const form = ref({ tableNo: '', area: '大厅', seats: 4, payMode: 'prepay' as PayMode })

onMounted(load)
async function load() {
  loading.value = true
  try {
    areas.value = await fetchTableAreas()
  } catch {
    message.error('加载桌台失败')
  } finally {
    loading.value = false
  }
}

const flat = computed<FlatRow[]>(() => areas.value.flatMap((a) => a.tables.map((t) => ({ area: a, table: t }))))
const summary = computed(() => ({
  total: flat.value.length,
  occ: flat.value.filter((x) => x.table.status === 'occupied').length,
  postpay: flat.value.filter((x) => x.table.payMode === 'postpay').length,
}))

const cols = [
  { title: '区域', dataIndex: ['area', 'name'], width: 120 },
  { title: '桌号', dataIndex: ['table', 'tableNo'], width: 100 },
  { title: '座位', dataIndex: ['table', 'seats'], width: 70, align: 'center' as const },
  { title: '结账模式', dataIndex: ['table', 'payMode'], width: 100 },
  { title: '当前状态', dataIndex: ['table', 'status'], width: 100 },
  { title: '关联会话', key: 'session', width: 130 },
  { title: '操作', key: 'op', width: 140 },
]

function openAdd() {
  editing.value = null
  form.value = { tableNo: '', area: '大厅', seats: 4, payMode: 'prepay' }
  modalOpen.value = true
}
function openEdit(row: { area: TableArea; table: TableInfo }) {
  editing.value = { areaName: row.area.name, table: row.table }
  form.value = { tableNo: row.table.tableNo, area: row.area.name, seats: row.table.seats, payMode: row.table.payMode }
  modalOpen.value = true
}

function save() {
  if (!form.value.tableNo.trim()) { message.warning('请输入桌号'); return }
  const hit = flat.value.find((x) => x.table.tableNo === form.value.tableNo && x.area.name !== form.value.area)
  if (hit) { message.warning('该桌号在其他区域已存在'); return }
  if (editing.value) {
    editing.value.table.tableNo = form.value.tableNo
    editing.value.table.seats = form.value.seats
    editing.value.table.payMode = form.value.payMode
    message.success('桌台已更新（桌台状态为派生字段，随会话自动变化）')
  } else {
    const area = areas.value.find((a) => a.name === form.value.area) || areas.value[0]
    area.tables.push({
      id: Date.now(), tableNo: form.value.tableNo, area: area.name, zone: area.name, seats: form.value.seats,
      payMode: form.value.payMode, status: 'empty',
    })
    message.success('桌台已新增')
  }
  modalOpen.value = false
}

function sessionText(r: FlatRow): string {
  const s = r.table.session
  if (!s) return '—'
  return `#${s.sessionId}` + (s.unsettled ? ` · 挂账 ¥${s.unsettled.toFixed(2)}` : '')
}

function remove(row: { area: TableArea; table: TableInfo }) {
  if (row.table.status === 'occupied') { message.warning('就餐中/未结账桌台不可删除'); return }
  Modal.confirm({
    title: `删除桌台 ${row.table.tableNo}`,
    content: '删除后顾客端二维码立即失效；历史订单仍保留。',
    okText: '删除', okButtonProps: { danger: true },
    onOk() {
      row.area.tables = row.area.tables.filter((t) => t.id !== row.table.id)
      message.success('已删除')
    },
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>桌台管理</h1>
        <p>共 {{ summary.total }} 桌 · 就餐中 {{ summary.occ }} · 后付桌 {{ summary.postpay }}</p>
      </div>
      <div class="page-head-right">
        <a-button @click="load">↻ 刷新</a-button>
        <a-button type="primary" @click="openAdd">＋ 新增桌台</a-button>
      </div>
    </div>

    <a-alert type="info" show-icon style="margin-bottom:16px">
      <template #message>桌台「当前状态」为<b>派生字段</b>（由会话/结账状态计算），不在本页维护；本页仅维护静态属性（区域/座位/结账模式）。</template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <a-table :loading="loading" :data-source="flat" :columns="cols" size="middle"
          :row-key="(r: { area: TableArea; table: TableInfo }) => r.table.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex[1] === 'payMode'">
              <a-tag :color="(record as FlatRow).table.payMode === 'postpay' ? 'purple' : 'blue'">{{ (record as FlatRow).table.payMode === 'postpay' ? '后付' : '先付' }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex[1] === 'status'">
              <a-tag :color="(record as FlatRow).table.status === 'occupied' ? 'blue' : 'default'">{{ (record as FlatRow).table.status === 'occupied' ? '就餐中' : '空闲' }}</a-tag>
            </template>
            <template v-else-if="column.key === 'session'">
              <span style="font-size:12px;color:var(--t3)">{{ sessionText(record as FlatRow) }}</span>
            </template>
            <template v-else-if="column.key === 'op'">
              <a-button size="small" type="link" @click="openEdit(record as FlatRow)">编辑</a-button>
              <a-button size="small" type="link" danger @click="remove(record as FlatRow)">删除</a-button>
            </template>
          </template>
        </a-table>
      </div>
    </div>

    <a-modal v-model:open="modalOpen" :title="editing ? `编辑桌台 ${editing.table?.tableNo}` : '新增桌台'" :footer="null" width="420">
      <div class="form-grid">
        <div class="fg"><label>桌号 *</label><a-input v-model:value="form.tableNo" placeholder="如 A11" /></div>
        <div class="fg"><label>区域</label>
          <a-select v-model:value="form.area" :options="areas.map((a) => ({ label: a.name, value: a.name }))" style="width:100%" />
        </div>
        <div class="fg"><label>座位数</label><a-input-number v-model:value="form.seats" :min="1" style="width:100%" /></div>
        <div class="fg"><label>默认结账模式</label>
          <a-radio-group v-model:value="form.payMode">
            <a-radio-button value="prepay">先付</a-radio-button>
            <a-radio-button value="postpay">后付</a-radio-button>
          </a-radio-group>
        </div>
      </div>
      <div style="margin-top:20px;text-align:right">
        <a-button style="margin-right:8px" @click="modalOpen = false">取消</a-button>
        <a-button type="primary" @click="save">保存</a-button>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px }
.fg { display: flex; flex-direction: column; gap: 6px }
.fg label { font-size: 13px; color: var(--t2) }
</style>
