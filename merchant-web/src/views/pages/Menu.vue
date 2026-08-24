<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchDishes } from '@/api/mock'
import type { Dish } from '@/types'

const dishes = ref<Dish[]>([])
const cat = ref('全部分类')
const kw = ref('')
const loading = ref(false)
const modalOpen = ref(false)
const isNew = ref(false)

// 表单字段
const form = ref<Dish>({ id: 0, name: '', category: '热菜', price: 0, unit: '份', itemType: 'dish', status: 'on', soldOut: false, station: '热菜档', specs: [] })
const specText = ref('')

const STATIONS = ['热菜档', '凉菜档', '汤羹档', '主食档', '吧台']

onMounted(load)
async function load() {
  loading.value = true
  try {
    dishes.value = await fetchDishes()
  } catch {
    message.error('加载菜品失败')
  } finally {
    loading.value = false
  }
}

const cats = computed(() => ['全部分类', ...new Set(dishes.value.map((d) => d.category))])
const filtered = computed(() =>
  dishes.value.filter((d) => {
    if (cat.value !== '全部分类' && d.category !== cat.value) return false
    if (kw.value.trim() && !d.name.includes(kw.value.trim())) return false
    return true
  }),
)
const counts = computed(() => ({
  on: dishes.value.filter((d) => d.status !== 'off').length,
  soldOut: dishes.value.filter((d) => d.soldOut).length,
}))

const columns = [
  { title: '菜品', dataIndex: 'name', width: 180 },
  { title: '分类', dataIndex: 'category', width: 90 },
  { title: '价格', dataIndex: 'price', width: 90, align: 'right' as const },
  { title: '单位', dataIndex: 'unit', width: 70 },
  { title: '档口', dataIndex: 'station', width: 90 },
  { title: '计费', key: 'itemType', width: 80 },
  { title: '状态', key: 'status', width: 130 },
  { title: '操作', key: 'op', width: 200 },
]

function openNew() {
  isNew.value = true
  form.value = { id: 0, name: '', category: cat.value === '全部分类' ? '热菜' : cat.value, price: 0, unit: '份', itemType: 'dish', status: 'on', soldOut: false, station: '热菜档', specs: [] }
  specText.value = ''
  modalOpen.value = true
}
function openEdit(d: Dish) {
  isNew.value = false
  form.value = { ...d, specs: d.specs ?? [] }
  specText.value = (d.specs ?? []).join('、')
  modalOpen.value = true
}

function save() {
  if (!form.value.name.trim()) { message.warning('请输入菜品名'); return }
  if (form.value.price <= 0) { message.warning('请输入正确的价格'); return }
  form.value.specs = specText.value.split(/[、,，]/).map((s) => s.trim()).filter(Boolean)
  if (isNew.value) {
    form.value.id = Date.now()
    dishes.value.push({ ...form.value })
    message.success(`已新增「${form.value.name}」`)
  } else {
    const hit = dishes.value.find((d) => d.id === form.value.id)
    if (hit) Object.assign(hit, { ...form.value })
    message.success(`已保存「${form.value.name}」`)
  }
  modalOpen.value = false
}

function toggleOnOff(d: Dish) {
  d.status = d.status === 'off' ? 'on' : 'off'
  message.success(`「${d.name}」已${d.status === 'off' ? '下架（顾客端不可见）' : '上架'}`)
}
function toggleSoldOut(d: Dish) {
  d.soldOut = !d.soldOut
  message.success(`「${d.name}」已${d.soldOut ? '沽清（当日生效）' : '恢复可售'}`)
}
function remove(d: Dish) {
  Modal.confirm({
    title: `删除「${d.name}」`,
    content: '删除后顾客端不可见；历史订单仍按原记录保留。',
    okText: '删除', okButtonProps: { danger: true },
    onOk() {
      dishes.value = dishes.value.filter((x) => x.id !== d.id)
      message.success('已删除')
    },
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>分类与菜品</h1>
        <p>在售 {{ counts.on }} · 当日沽清 {{ counts.soldOut }} · 上下架实时同步顾客端</p>
      </div>
      <div class="page-head-right">
        <a-input v-model:value="kw" placeholder="搜菜品名" allow-clear style="width:160px" />
        <a-button @click="load">↻ 刷新</a-button>
        <a-button type="primary" @click="openNew">＋ 新增菜品</a-button>
      </div>
    </div>

    <div class="menu-grid">
      <!-- 左：分类 -->
      <div class="card">
        <div class="card-head"><div><h3>分类</h3><p>{{ dishes.length }} 道菜品</p></div></div>
        <div class="card-body" style="padding:0">
          <div v-for="c in cats" :key="c" class="cat-item" :class="{ on: cat === c }" @click="cat = c">
            <span>{{ c }}</span><em>{{ c === '全部分类' ? dishes.length : dishes.filter((d) => d.category === c).length }}</em>
          </div>
        </div>
      </div>

      <!-- 右：菜品表 -->
      <div class="card">
        <div class="card-body">
          <a-table :loading="loading" :data-source="filtered" :columns="columns" size="middle"
            :row-key="(r: Dish) => r.id" :pagination="{ pageSize: 10, showSizeChanger: false }">
            <template #bodyCell="{ column, record }">
              <template v-if="column.dataIndex === 'price'"><b class="num">¥{{ (record as Dish).price.toFixed(2) }}</b></template>
              <template v-else-if="column.key === 'itemType'"><a-tag color="purple" v-if="(record as Dish).itemType === 'per_head'">按人</a-tag><span v-else style="color:var(--t3)">按份</span></template>
              <template v-else-if="column.key === 'status'">
                <a-tag v-if="(record as Dish).status === 'off'" color="default">已下架</a-tag>
                <a-tag v-else-if="(record as Dish).soldOut" color="red">当日沽清</a-tag>
                <a-tag v-else color="green">在售</a-tag>
              </template>
              <template v-else-if="column.key === 'op'">
                <a-button size="small" type="link" @click="openEdit(record as Dish)">编辑</a-button>
                <a-button size="small" type="link" @click="toggleOnOff(record as Dish)">{{ (record as Dish).status === 'off' ? '上架' : '下架' }}</a-button>
                <a-button size="small" type="link" @click="toggleSoldOut(record as Dish)">{{ (record as Dish).soldOut ? '恢复' : '沽清' }}</a-button>
                <a-button size="small" type="link" danger @click="remove(record as Dish)">删除</a-button>
              </template>
            </template>
          </a-table>
        </div>
      </div>
    </div>

    <!-- 新增/编辑菜品 -->
    <a-modal v-model:open="modalOpen" :title="isNew ? '新增菜品' : `编辑 · ${form.name}`" :footer="null" width="520">
      <div class="form-grid">
        <div class="fg"><label>菜品名 *</label><a-input v-model:value="form.name" placeholder="如 宫保鸡丁" /></div>
        <div class="fg"><label>分类</label><a-select v-model:value="form.category" :options="cats.filter((c) => c !== '全部分类').map((c) => ({ label: c, value: c }))" style="width:100%" /></div>
        <div class="fg"><label>价格（元）*</label><a-input-number v-model:value="form.price" :min="0" :precision="2" style="width:100%" /></div>
        <div class="fg"><label>单位</label><a-select v-model:value="form.unit" :options="['份','碗','煲','瓶','杯','人'].map((u) => ({ label: u, value: u }))" style="width:100%" /></div>
        <div class="fg"><label>计费方式</label>
          <a-radio-group v-model:value="form.itemType">
            <a-radio-button value="dish">按份</a-radio-button>
            <a-radio-button value="per_head">按人</a-radio-button>
          </a-radio-group>
        </div>
        <div class="fg"><label>出品档口</label>
          <a-select v-model:value="form.station" :options="STATIONS.map((s) => ({ label: s, value: s }))" style="width:100%" />
        </div>
        <div class="fg wide"><label>规格（逗号分隔，如 大份/小份）</label><a-input v-model:value="specText" placeholder="不填则无规格" /></div>
        <div class="fg"><label>上架状态</label>
          <a-radio-group v-model:value="form.status">
            <a-radio-button value="on">上架</a-radio-button>
            <a-radio-button value="off">下架</a-radio-button>
          </a-radio-group>
        </div>
        <div class="fg"><label>当日沽清</label><a-switch v-model:checked="form.soldOut" /></div>
      </div>
      <div style="margin-top:20px;text-align:right">
        <a-button style="margin-right:8px" @click="modalOpen = false">取消</a-button>
        <a-button type="primary" @click="save">保存</a-button>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.menu-grid { display: grid; grid-template-columns: 200px 1fr; gap: 16px; align-items: start }
.cat-item { padding: 12px 18px; font-size: 14px; color: var(--t2); cursor: pointer; display: flex; align-items: center;
  border-left: 3px solid transparent; transition: .12s }
.cat-item:hover { background: var(--hover); color: var(--t1) }
.cat-item.on { color: var(--brand); border-left-color: var(--brand); background: var(--brand-bg); font-weight: 500 }
.cat-item em { margin-left: auto; font-style: normal; font-size: 12px; color: var(--t3) }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px }
.fg { display: flex; flex-direction: column; gap: 6px }
.fg.wide { grid-column: 1 / -1 }
.fg label { font-size: 13px; color: var(--t2) }
</style>
