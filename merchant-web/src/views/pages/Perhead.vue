<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { fetchDishes } from '@/api/mock'
import type { Dish } from '@/types'

const list = ref<Dish[]>([])
const loading = ref(false)
const editing = ref<Dish | null>(null)
const modalOpen = ref(false)

// 按人项扩展设置（仅本地状态，mock 阶段不落库）
const ex = ref<Record<number, { tea: boolean; kidHalf: boolean; minPax: number; freeDrink: boolean }>>({})

onMounted(load)
async function load() {
  loading.value = true
  try {
    const all = await fetchDishes()
    list.value = all.filter((d) => d.itemType === 'per_head')
    for (const d of list.value) {
      if (!ex.value[d.id]) ex.value[d.id] = { tea: true, kidHalf: false, minPax: 2, freeDrink: false }
    }
  } catch {
    message.error('加载按人项失败')
  } finally {
    loading.value = false
  }
}

const columns = [
  { title: '项目', dataIndex: 'name', width: 140 },
  { title: '单价（元/人）', dataIndex: 'price', width: 110, align: 'right' as const },
  { title: '含茶水', key: 'tea', width: 90 },
  { title: '儿童半价', key: 'kid', width: 100 },
  { title: '最低起收', key: 'min', width: 100 },
  { title: '酒水畅饮', key: 'drink', width: 100 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '操作', key: 'op', width: 90 },
]

function openEdit(d: Dish) {
  editing.value = d
  modalOpen.value = true
}

function save() {
  if (!editing.value) return
  message.success(`已保存「${editing.value.name}」按人项设置`)
  modalOpen.value = false
}

function toggleOnOff(d: Dish) {
  d.status = d.status === 'off' ? 'on' : 'off'
  message.success(`「${d.name}」已${d.status === 'off' ? '下架' : '上架'}`)
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>按人收费项</h1>
        <p>自助/按位计费项目 · 结算时按就餐人数 × 单价，并叠加折扣规则</p>
      </div>
      <div class="page-head-right">
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <a-alert type="info" show-icon style="margin-bottom:16px">
      <template #message>按人项按<b>会话人数</b>计费（结账时以就餐人数为准）；儿童半价与最低起收由本页配置，变更即时生效。</template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <a-table :loading="loading" :data-source="list" :columns="columns" size="middle"
          :row-key="(r: Dish) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'price'"><b class="num">¥{{ (record as Dish).price.toFixed(2) }}</b></template>
            <template v-else-if="column.key === 'tea'"><a-switch :checked="ex[(record as Dish).id]?.tea" size="small" /></template>
            <template v-else-if="column.key === 'kid'"><a-switch :checked="ex[(record as Dish).id]?.kidHalf" size="small" /></template>
            <template v-else-if="column.key === 'min'"><span class="num">{{ ex[(record as Dish).id]?.minPax ?? '—' }} 人起</span></template>
            <template v-else-if="column.key === 'drink'"><a-switch :checked="ex[(record as Dish).id]?.freeDrink" size="small" /></template>
            <template v-else-if="column.dataIndex === 'status'">
              <a-tag :color="(record as Dish).status === 'off' ? 'default' : 'green'">{{ (record as Dish).status === 'off' ? '已下架' : '在售' }}</a-tag>
            </template>
            <template v-else-if="column.key === 'op'">
              <a-button size="small" type="link" @click="openEdit(record as Dish)">设置</a-button>
              <a-button size="small" type="link" @click="toggleOnOff(record as Dish)">{{ (record as Dish).status === 'off' ? '上架' : '下架' }}</a-button>
            </template>
          </template>
        </a-table>
      </div>
    </div>

    <a-modal v-model:open="modalOpen" :title="`按人项设置 · ${editing?.name}`" :footer="null" width="460">
      <div v-if="editing" class="form-grid">
        <div class="fg"><label>单价（元/人）</label><a-input-number v-model:value="editing.price" :min="0" :precision="2" style="width:100%" /></div>
        <div class="fg"><label>最低起收（人）</label><a-input-number v-model:value="ex[editing.id].minPax" :min="1" style="width:100%" /></div>
        <div class="fg"><label>含茶水</label><a-switch v-model:checked="ex[editing.id].tea" /></div>
        <div class="fg"><label>儿童半价（身高 < 1.2m）</label><a-switch v-model:checked="ex[editing.id].kidHalf" /></div>
        <div class="fg wide"><label>酒水畅饮</label><a-switch v-model:checked="ex[editing.id].freeDrink" /></div>
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
.fg.wide { grid-column: 1 / -1 }
.fg label { font-size: 13px; color: var(--t2) }
</style>
