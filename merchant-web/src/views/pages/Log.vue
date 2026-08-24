<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { fetchOpLogs } from '@/api/mock'
import type { OpLog } from '@/types'

const list = ref<OpLog[]>([])
const level = ref('全部')
const kw = ref('')
const loading = ref(false)

const LV_META: Record<string, { label: string; color: string }> = {
  info: { label: '普通', color: 'default' },
  warn: { label: '敏感操作', color: 'gold' },
  danger: { label: '高危', color: 'red' },
}

const cols = [
  { title: '时间', dataIndex: 'time', width: 110 },
  { title: '操作人', dataIndex: 'operator', width: 100 },
  { title: '动作', dataIndex: 'action', width: 130 },
  { title: '对象', dataIndex: 'target', width: 140 },
  { title: '详情', dataIndex: 'detail', ellipsis: true },
  { title: '级别', dataIndex: 'level', width: 100 },
]

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = await fetchOpLogs()
  } catch {
    message.error('加载日志失败')
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  const s = kw.value.trim()
  return list.value.filter((l) => {
    if (level.value !== '全部' && l.level !== level.value) return false
    if (s && !(l.operator.includes(s) || l.action.includes(s) || l.detail.includes(s))) return false
    return true
  })
})
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>操作日志</h1>
        <p>登录 / 结账 / 退款 / 改价 / 降级开关等敏感操作留痕，不可删除（仅店长可导出）</p>
      </div>
      <div class="page-head-right">
        <div class="seg">
          <div v-for="v in ['全部', 'info', 'warn', 'danger']" :key="v" :class="{ on: level === v }" @click="level = v">
            {{ v === '全部' ? '全部' : LV_META[v].label }}
          </div>
        </div>
        <a-input v-model:value="kw" placeholder="搜操作人/动作/详情" allow-clear style="width:180px" />
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <div class="card">
      <div class="card-body">
        <a-table :loading="loading" :data-source="filtered" :columns="cols" size="middle"
          :row-key="(r: OpLog) => r.id" :pagination="{ pageSize: 12, showSizeChanger: false }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'level'">
              <a-tag :color="LV_META[(record as OpLog).level].color">{{ LV_META[(record as OpLog).level].label }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'action'">
              <b style="font-weight:500">{{ (record as OpLog).action }}</b>
            </template>
          </template>
        </a-table>
      </div>
    </div>
  </div>
</template>
