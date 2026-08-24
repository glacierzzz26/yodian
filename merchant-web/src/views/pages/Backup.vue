<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchBackups } from '@/api/mock'
import type { BackupTask } from '@/types'

const list = ref<BackupTask[]>([])
const loading = ref(false)

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = await fetchBackups()
  } catch {
    message.error('加载备份列表失败')
  } finally {
    loading.value = false
  }
}

const successCount = computed(() => list.value.filter((b) => b.status === 'success').length)
const lastTime = computed(() => list.value.find((b) => b.type === 'full' && b.status === 'success')?.time || '—')

const cols = [
  { title: '时间', dataIndex: 'time', width: 150 },
  { title: '类型', dataIndex: 'type', width: 90 },
  { title: '大小', dataIndex: 'size', width: 90 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '说明', dataIndex: 'note', ellipsis: true },
  { title: '操作', key: 'op', width: 140 },
]

function doBackup() {
  Modal.confirm({
    title: '立即全量备份',
    content: '备份耗时取决于数据量，期间正常营业不受影响。备份写入本机 + 同步对象存储（异地容灾）。',
    okText: '开始备份',
    onOk() {
      list.value.unshift({ id: Date.now(), time: '刚刚', type: 'manual', size: '计算中…', status: 'running', note: '手动全量备份 · 进行中' })
      setTimeout(() => {
        const hit = list.value[0]
        if (hit) { hit.status = 'success'; hit.size = '128.5MB'; hit.note = '手动全量备份 · 完成' }
        message.success('备份完成（mock）· 已同步对象存储')
      }, 800)
    },
  })
}

function restore(b: BackupTask) {
  if (b.status === 'failed') { message.warning('该备份失败不可用'); return }
  Modal.confirm({
    title: `恢复到该备份（${b.time}）`,
    content: '恢复将覆盖当前数据库并短暂中断服务，请在低峰操作。恢复前系统自动再做一次全量备份。',
    okText: '确认恢复', okButtonProps: { danger: true },
    onOk() { message.success('恢复任务已提交（mock）· 完成后自动重启服务并校验六口径') },
  })
}

function remove(b: BackupTask) {
  Modal.confirm({
    title: '删除该备份',
    content: '删除后不可恢复；请确认已存在至少一份可用全量备份。',
    okText: '删除', okButtonProps: { danger: true },
    onOk() {
      list.value = list.value.filter((x) => x.id !== b.id)
      message.success('已删除')
    },
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>备份与恢复</h1>
        <p>策略：每日 03:00 全量 + 每小时 WAL + 异地对象存储（设计 12 章）· 最近全量 {{ lastTime }}</p>
      </div>
      <div class="page-head-right">
        <a-button @click="load">↻ 刷新</a-button>
        <a-button type="primary" @click="doBackup">⚡ 立即备份</a-button>
      </div>
    </div>

    <a-alert type="info" show-icon style="margin-bottom:16px">
      <template #message>
        备份可靠性检查：本机 {{ successCount }} 份可用 · 异地对象存储已启用 · 恢复演练每季度一次（上次 2026-08-10，RPO ≤ 1h，RTO ≤ 30min）。
        <b>凭据备份与数据库备份分开存放（12.6）。</b>
      </template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <a-table :loading="loading" :data-source="list" :columns="cols" size="middle"
          :row-key="(r: BackupTask) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'type'">
              <a-tag :color="(record as BackupTask).type === 'full' ? 'blue' : (record as BackupTask).type === 'wal' ? 'default' : 'purple'">
                {{ (record as BackupTask).type === 'full' ? '全量' : (record as BackupTask).type === 'wal' ? 'WAL' : '手动' }}
              </a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'status'">
              <a-tag :color="(record as BackupTask).status === 'success' ? 'green' : (record as BackupTask).status === 'running' ? 'blue' : 'red'">
                {{ (record as BackupTask).status === 'success' ? '成功' : (record as BackupTask).status === 'running' ? '进行中' : '失败' }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'op'">
              <a-button size="small" type="link" @click="restore(record as BackupTask)">恢复</a-button>
              <a-button size="small" type="link" danger @click="remove(record as BackupTask)">删除</a-button>
            </template>
          </template>
        </a-table>
      </div>
    </div>
  </div>
</template>
