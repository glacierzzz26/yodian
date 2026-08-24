<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchPrintTasks, reprint } from '@/api/mock'
import { useAuthStore } from '@/stores/auth'
import type { PrintTask } from '@/types'

const auth = useAuthStore()
const list = ref<PrintTask[]>([])
const status = ref('全部')
const loading = ref(false)
const addOpen = ref(false)
const newKind = ref('下单小票')
const newStation = ref('热菜档')

const ST_META: Record<string, { label: string; color: string }> = {
  pending: { label: '待发送', color: 'default' },
  sent: { label: '已发送', color: 'blue' },
  failed: { label: '发送失败', color: 'red' },
}
const KIND_OPTS = ['下单小票', '结账小票', '预结账单']
const STATION_OPTS = ['热菜档', '凉菜档', '汤羹档', '主食档', '吧台', '收银台']

const columns = [
  { title: '档口', dataIndex: 'station', width: 110 },
  { title: '打印机', dataIndex: 'printerSn', width: 130 },
  { title: '内容', dataIndex: 'kind', ellipsis: true },
  { title: '状态', dataIndex: 'status', width: 100 },
  { title: '重试', dataIndex: 'retryCount', width: 64, align: 'center' as const },
  { title: '时间', dataIndex: 'createdAt', width: 90 },
  { title: '操作', key: 'op', width: 140 },
]

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = await fetchPrintTasks()
  } catch {
    message.error('加载打印任务失败')
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => (status.value === '全部' ? list.value : list.value.filter((p) => p.status === status.value)))
const counts = computed(() => {
  const c: Record<string, number> = { 全部: list.value.length, pending: 0, sent: 0, failed: 0 }
  for (const p of list.value) c[p.status]++
  return c
})

function doReprint(p: PrintTask) {
  Modal.confirm({
    title: `补打「${p.kind}」`,
    content: `向 ${p.station}（${p.printerSn}）重新发送打印任务。`,
    okText: '补打',
    onOk: async () => {
      await reprint(p.id)
      p.status = 'sent'
      p.retryCount += 1
      message.success('已重新发送（模拟）')
    },
  })
}

function doRetry(p: PrintTask) {
  Modal.confirm({ title: '重试发送', content: `重新尝试向 ${p.printerSn} 发送「${p.kind}」。`, okText: '重试', onOk: () => { p.status = 'sent'; message.success('已重试（模拟）') } })
}

function onAdd() {
  addOpen.value = false
  list.value.unshift({
    id: Date.now(), station: newStation.value, printerSn: 'FE-8021', kind: `${newKind.value} · 手动`,
    status: 'sent', retryCount: 0, createdAt: '刚刚',
  })
  message.success('打印任务已创建（模拟）')
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>打印任务</h1>
        <p>订单/结账小票下发记录 · 待发送 {{ counts.pending }} · 失败 {{ counts.failed }}</p>
      </div>
      <div class="page-head-right">
        <a-button v-if="auth.role === 'owner'" @click="addOpen = true">＋ 新增打印任务</a-button>
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <a-alert v-if="counts.failed" type="error" show-icon style="margin-bottom:16px">
      <template #message><b>{{ counts.failed }} 个打印任务发送失败</b>，请检查对应打印机（离线/缺纸），修复后点击「补打」。</template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <div class="fchips">
          <span :class="{ on: status === '全部' }" @click="status = '全部'">全部 {{ counts['全部'] }}</span>
          <span v-for="(meta, k) in ST_META" :key="k" :class="{ on: status === k }" @click="status = k">{{ meta.label }} {{ counts[k] || 0 }}</span>
        </div>

        <a-table :loading="loading" :data-source="filtered" :columns="columns" size="middle"
          :row-key="(r: PrintTask) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="ST_META[(record as PrintTask).status].color">{{ ST_META[(record as PrintTask).status].label }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'retryCount'">
              <span :style="{ color: (record as PrintTask).retryCount > 0 ? 'var(--wn)' : 'var(--t3)' }">{{ (record as PrintTask).retryCount }}</span>
            </template>
            <template v-else-if="column.key === 'op'">
              <a-button size="small" type="link" @click="doReprint(record as PrintTask)">补打</a-button>
              <a-button v-if="(record as PrintTask).status === 'pending'" size="small" type="link" @click="doRetry(record as PrintTask)">重试</a-button>
            </template>
          </template>
        </a-table>
      </div>
    </div>

    <div style="margin-top:14px;font-size:12px;color:var(--t3)">
      打印机离线/缺纸检测与提醒阈值在 <a @click.prevent="undefined">系统设置 · 打印机</a> 配置（design 8 章）。补打按原单内容原样重发，不计入重复出单。
    </div>

    <!-- 新增打印任务 -->
    <a-modal v-model:open="addOpen" title="新增打印任务" :footer="null" width="400">
      <div class="f-l">任务内容</div>
      <a-radio-group v-model:value="newKind" style="margin-bottom:14px">
        <a-radio-button v-for="k in KIND_OPTS" :key="k" :value="k">{{ k }}</a-radio-button>
      </a-radio-group>
      <div class="f-l">目标档口 / 打印机</div>
      <a-select v-model:value="newStation" :options="STATION_OPTS.map((s) => ({ label: s, value: s }))" style="width:100%;margin-bottom:18px" />
      <div style="text-align:right">
        <a-button style="margin-right:8px" @click="addOpen = false">取消</a-button>
        <a-button type="primary" @click="onAdd">创建</a-button>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.fchips { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 14px }
.fchips span { height: 28px; padding: 0 12px; border-radius: 14px; border: 1px solid var(--split);
  font-size: 13px; color: var(--t2); cursor: pointer; background: var(--card) }
.fchips span.on { border-color: var(--brand); color: var(--brand); background: var(--brand-bg) }
.f-l { font-size: 13px; color: var(--t2); margin-bottom: 7px; font-weight: 500 }
</style>
