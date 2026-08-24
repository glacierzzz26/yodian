<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { fetchDishes } from '@/api/mock'
import type { Dish } from '@/types'

const dishes = ref<Dish[]>([])
const cat = ref('全部')
const loading = ref(false)

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

const cats = computed(() => ['全部', ...new Set(dishes.value.map((d) => d.category))])
const shown = computed(() => (cat.value === '全部' ? dishes.value : dishes.value.filter((d) => d.category === cat.value)))
const soldCount = computed(() => dishes.value.filter((d) => d.soldOut).length)

function toggle(d: Dish) {
  d.soldOut = !d.soldOut
  message.success(`「${d.name}」已${d.soldOut ? '沽清（顾客端立即显示「今日已售罄」）' : '恢复可售'}`)
}

function clearAll() {
  dishes.value.forEach((d) => { d.soldOut = false })
  message.success('已恢复全部菜品为可售')
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>沽清管理</h1>
        <p>点击切换当日沽清 · 当前沽清 {{ soldCount }} 道 · 即时同步顾客端与 KDS</p>
      </div>
      <div class="page-head-right">
        <div class="seg">
          <div v-for="c in cats" :key="c" :class="{ on: cat === c }" @click="cat = c">{{ c }}</div>
        </div>
        <a-button :disabled="!soldCount" @click="clearAll">全部恢复可售</a-button>
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <div class="card">
      <div class="card-body">
        <div class="sg-grid">
          <div v-for="d in shown" :key="d.id" class="sg" :class="{ off: d.status === 'off', so: d.soldOut }" @click="toggle(d)">
            <div class="sg-name">{{ d.name }}</div>
            <div class="sg-meta">{{ d.category }} · ¥{{ d.price }}{{ d.itemType === 'per_head' ? '/人' : '/' + d.unit }}</div>
            <span v-if="d.soldOut" class="sg-badge">今日已售罄 · 点击恢复</span>
            <span v-else-if="d.status === 'off'" class="sg-badge off">已下架（不可沽清）</span>
            <span v-else class="sg-badge on">可售 · 点击沽清</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sg-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(170px, 1fr)); gap: 12px }
.sg { position: relative; border: 1px solid var(--split); border-radius: 8px; padding: 14px; cursor: pointer; transition: .15s; background: var(--card) }
.sg:hover { box-shadow: var(--sh) }
.sg.so { border-color: var(--er); background: var(--er-bg) }
.sg.off { opacity: .5 }
.sg-name { font-size: 14px; font-weight: 600 }
.sg-meta { font-size: 12px; color: var(--t3); margin: 2px 0 10px }
.sg-badge { display: inline-block; font-size: 12px; padding: 3px 10px; border-radius: 4px }
.sg-badge.on { color: var(--ok); background: var(--ok-bg); border: 1px solid var(--ok-bd) }
.sg-badge.so { color: var(--er); background: var(--card); border: 1px solid var(--er) }
.sg-badge.off { color: var(--t3); background: var(--bg); border: 1px solid var(--bd) }
</style>
