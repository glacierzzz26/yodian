<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchRecon } from '@/api/mock'
import type { ReconRow } from '@/types'

const list = ref<ReconRow[]>([])
const loading = ref(false)
const scope = ref('全部口径')

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = await fetchRecon()
  } catch {
    message.error('加载对账数据失败')
  } finally {
    loading.value = false
  }
}

const scopes = computed(() => ['全部口径', ...new Set(list.value.map((r) => r.scope))])
const filtered = computed(() => (scope.value === '全部口径' ? list.value : list.value.filter((r) => r.scope === scope.value)))
const diffCount = computed(() => list.value.filter((r) => r.diff !== 0).length)
const totalDiff = computed(() => list.value.reduce((s, r) => s + Math.abs(r.diff), 0))

function diffText(r: ReconRow): string {
  return (r.diff > 0 ? '+' : '') + r.diff.toFixed(2)
}

const cols = [
  { title: '口径', dataIndex: 'scope', width: 110 },
  { title: '对账项目', dataIndex: 'name', ellipsis: true },
  { title: '账上（应收）', dataIndex: 'expected', width: 120, align: 'right' as const },
  { title: '对到（实收）', dataIndex: 'actual', width: 120, align: 'right' as const },
  { title: '差额', dataIndex: 'diff', width: 110, align: 'right' as const },
  { title: '结果', key: 'status', width: 90 },
  { title: '说明', dataIndex: 'note', ellipsis: true },
]

function explain(row: ReconRow) {
  Modal.info({
    title: `差异说明 · ${row.name}`,
    content: () => [
      h('p', null, ['账上应收 ', h('b', null, `¥${row.expected.toFixed(2)}`), '，实际对到 ', h('b', null, `¥${row.actual.toFixed(2)}`), '，差额 ', h('b', { style: 'color:var(--er)' }, `¥${row.diff.toFixed(2)}`), '。']),
      h('p', null, '处理建议：核对该渠道当班流水与收银记录；现金短款需班次清点确认并留存说明，不计入跑单。'),
      h('p', null, '确认后在「退款与核销」或人工单中冲正，系统将把该差异归入对账差异报表。'),
    ],
  })
}

function exportRecon() {
  message.success('对账报告已导出（mock）：含六口径汇总 + 差异明细 + 责任人待办')
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>多来源对账</h1>
        <p>六口径：订单应收 / 支付网关 / 渠道到账 / 财务冲减 · 有差异 {{ diffCount }} 项，合计 ¥{{ totalDiff.toFixed(2) }}</p>
      </div>
      <div class="page-head-right">
        <div class="seg">
          <div v-for="s in scopes" :key="s" :class="{ on: scope === s }" @click="scope = s">{{ s }}</div>
        </div>
        <a-button @click="exportRecon">⬇ 导出对账报告</a-button>
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <a-alert :type="diffCount ? 'warning' : 'success'" show-icon style="margin-bottom:16px">
      <template #message>
        <template v-if="diffCount">
          <b>{{ diffCount }} 项差异未闭环</b>：现金短款 ¥4.85 待班次清点确认；请在当日打烊对账前处理完毕，逾期自动置入次日对账差异报表。
        </template>
        <template v-else>全部口径已对平，无差异。</template>
      </template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <a-table :loading="loading" :data-source="filtered" :columns="cols" size="middle"
          :row-key="(r: ReconRow) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'expected'"><span class="num">{{ (record as ReconRow).expected.toFixed(2) }}</span></template>
            <template v-else-if="column.dataIndex === 'actual'"><b class="num" :style="{ color: (record as ReconRow).diff !== 0 ? 'var(--er)' : 'inherit' }">{{ (record as ReconRow).actual.toFixed(2) }}</b></template>
            <template v-else-if="column.dataIndex === 'diff'">
              <b class="num" :style="{ color: (record as ReconRow).diff !== 0 ? 'var(--er)' : 'var(--ok)' }">{{ diffText(record as ReconRow) }}</b>
            </template>
            <template v-else-if="column.key === 'status'">
              <a-tag :color="(record as ReconRow).diff === 0 ? 'green' : 'red'">{{ (record as ReconRow).diff === 0 ? '已对平' : '有差异' }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'note'">
              <span style="color:var(--t3)">{{ (record as ReconRow).note }}</span>
              <a v-if="(record as ReconRow).diff !== 0" style="margin-left:6px" @click="explain(record as ReconRow)">处理</a>
            </template>
          </template>
        </a-table>
      </div>
    </div>

    <div style="margin-top:14px;font-size:12px;color:var(--t3)">
      六口径定义见设计文档 5 章：订单表应收（线上/人工）、支付网关账单（微信/支付宝）、渠道结算账户、收银现金清点、财务冲减（退款/核销）。所有差异项处理均留痕。
    </div>
  </div>
</template>
