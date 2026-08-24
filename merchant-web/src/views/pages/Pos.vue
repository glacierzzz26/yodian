<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchDishes, fetchTableAreas } from '@/api/mock'
import type { Dish, PayChannel, TableArea, TableInfo } from '@/types'

const dishes = ref<Dish[]>([])
const areas = ref<TableArea[]>([])
const cat = ref('全部')
const selKey = ref<'takeout' | number>('takeout') // 'takeout' = 外带自取
const cart = ref<CartLine[]>([])
const payOpen = ref(false)
const tableOpen = ref(false)
const payChannel = ref<PayChannel>('wechat')

interface CartLine { dish: Dish; qty: number; remark?: string }

const allTables = computed<TableInfo[]>(() => areas.value.flatMap((a) => a.tables))
const selTable = computed<TableInfo | null>(() => (selKey.value === 'takeout' ? null : allTables.value.find((t) => t.id === selKey.value) || null))

onMounted(async () => {
  const [d, t] = await Promise.all([fetchDishes(), fetchTableAreas()])
  dishes.value = d
  areas.value = t
})

const cats = computed(() => ['全部', ...new Set(dishes.value.map((d) => d.category))])
const dishList = computed(() => (cat.value === '全部' ? dishes.value : dishes.value.filter((d) => d.category === cat.value)))

// 三种结账模式：外带/先付桌 → prepay；后付桌 → postpay；仅点单档口 → frontend
const payMode = computed<'prepay' | 'postpay' | 'frontend'>(() => (selTable.value ? selTable.value.payMode : 'prepay'))
const modeLabel = computed(() => (payMode.value === 'postpay' ? '后付挂账' : payMode.value === 'frontend' ? '仅点单' : '先付即结'))

const total = computed(() => cart.value.reduce((s, l) => s + l.dish.price * l.qty, 0))
const totalQty = computed(() => cart.value.reduce((s, l) => s + l.qty, 0))
const qtyOf = (dishId: number) => cart.value.find((l) => l.dish.id === dishId)?.qty ?? 0

function add(d: Dish) {
  if (d.soldOut) { message.warning(`「${d.name}」已沽清`); return }
  if (d.itemType === 'per_head') { promptPax(d); return }
  const hit = cart.value.find((l) => l.dish.id === d.id)
  if (hit) hit.qty += 1
  else cart.value.push({ dish: d, qty: 1 })
}
function minus(d: Dish) {
  const hit = cart.value.find((l) => l.dish.id === d.id)
  if (!hit) return
  hit.qty -= 1
  if (hit.qty <= 0) cart.value = cart.value.filter((l) => l.dish.id !== d.id)
}

function promptPax(d: Dish) {
  let pax = 2
  Modal.confirm({
    title: `「${d.name}」按人计费（¥${d.price}/人）`,
    content: () => h('input', {
      value: pax, type: 'number', min: 1,
      style: 'width:100%;height:36px;padding:0 10px;border:1px solid #d9d9d9;border-radius:6px;outline:none;font-size:14px',
      onInput: (e: Event) => { pax = Math.max(1, Number((e.target as HTMLInputElement).value) || 1) },
    }),
    okText: '加入清单', cancelText: '取消',
    onOk() {
      const hit = cart.value.find((l) => l.dish.id === d.id)
      if (hit) hit.qty += pax
      else cart.value.push({ dish: d, qty: pax })
    },
  })
}

function editRemark(line: CartLine) {
  let text = line.remark || ''
  Modal.confirm({
    title: `「${line.dish.name}」备注`,
    content: () => h('input', {
      value: text, placeholder: '如：少辣 / 免香菜',
      style: 'width:100%;height:36px;padding:0 10px;border:1px solid #d9d9d9;border-radius:6px;outline:none;font-size:14px',
      onInput: (e: Event) => { text = (e.target as HTMLInputElement).value },
    }),
    okText: '保存', cancelText: '取消',
    onOk() { line.remark = text.trim() || undefined },
  })
}

function openPayModal() {
  if (!total.value) { message.warning('请先选择菜品'); return }
  payOpen.value = true
}

function submitOrder() {
  const lineDesc = cart.value.map((l) => `${l.dish.name}×${l.qty}`).join('、')
  const where = selTable.value ? `${selTable.value.tableNo}（${selTable.value.area}）` : '外带'
  payOpen.value = false
  message.success(
    `${where} ${modeLabel.value}下单成功（模拟）· ${lineDesc} · ¥${total.value.toFixed(2)}` +
    (payMode.value === 'postpay' ? ' · 已挂账，清台前统一结账' : ` · 已收 ${payChannel.value === 'wechat' ? '微信支付' : payChannel.value === 'alipay' ? '支付宝' : payChannel.value === 'cash' ? '现金' : 'POS'} ¥${total.value.toFixed(2)}`),
  )
  cart.value = []
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>代客点单</h1>
        <p>收银台代客下单 · 三种结账模式同屏切换（先付/后付/仅点单）</p>
      </div>
      <div class="page-head-right">
        <a-button @click="tableOpen = true">
          {{ selTable ? `${selTable.tableNo} · ${selTable.area}` : '外带自取' }}
        </a-button>
        <a-button @click="cart = []" :disabled="!totalQty">清空</a-button>
        <a-button type="primary" @click="openPayModal()">{{ modeLabel }}并提交</a-button>
      </div>
    </div>

    <div class="pos-grid">
      <!-- 左：分类 -->
      <div class="card col-cat">
        <div class="card-body" style="padding:0">
          <div v-for="c in cats" :key="c" class="cat-item" :class="{ on: cat === c }" @click="cat = c">{{ c }}</div>
        </div>
      </div>

      <!-- 中：菜品 -->
      <div class="card col-dish">
        <div class="card-body">
          <div class="dish-grid">
            <div v-for="d in dishList" :key="d.id" class="dish" :class="{ off: d.soldOut }" @click="add(d)">
              <div class="d-name">{{ d.name }}</div>
              <div class="d-meta">{{ d.itemType === 'per_head' ? '按人计费' : d.unit }} · {{ d.station }}</div>
              <div class="d-price"><b class="num">¥{{ d.price }}</b><small>/{{ d.unit }}</small></div>
              <span v-if="d.soldOut" class="soldout">沽清</span>
              <span v-if="qtyOf(d.id) > 0" class="dq num">{{ qtyOf(d.id) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 右：清单 -->
      <div class="card col-cart">
        <div class="card-head">
          <div><h3>已点清单</h3><p>{{ selTable ? selTable.tableNo : '外带' }} · {{ modeLabel }}</p></div>
          <span class="tag g">{{ totalQty }} 件</span>
        </div>
        <div class="card-body">
          <div v-if="!cart.length" class="empty">点击菜品加入清单</div>
          <div v-for="l in cart" :key="l.dish.id" class="cline">
            <div class="cname">
              <b>{{ l.dish.name }}</b>
              <small v-if="l.remark">「{{ l.remark }}」</small>
            </div>
            <a-button size="small" type="text" @click="editRemark(l)">备注</a-button>
            <div class="stepper">
              <a-button size="small" shape="circle" @click="minus(l.dish)">−</a-button>
              <b class="num">{{ l.qty }}</b>
              <a-button size="small" shape="circle" type="primary" @click="add(l.dish)">+</a-button>
            </div>
            <em class="num">¥{{ (l.dish.price * l.qty).toFixed(2) }}</em>
          </div>
        </div>
        <div class="card-foot">
          <div class="tot"><span>应收</span><b class="num" style="color:var(--er)">¥{{ total.toFixed(2) }}</b></div>
          <div class="tot-op">
            <a-button size="large" @click="openPayModal()">仅下单</a-button>
            <a-button size="large" type="primary" @click="openPayModal()">下单并收款</a-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 选桌 / 外带 -->
    <a-modal v-model:open="tableOpen" title="选择桌台 / 外带" :footer="null">
      <a-radio-group v-model:value="selKey" style="width:100%">
        <a-radio :value="'takeout'" style="display:block;margin-bottom:6px"><b>外带自取</b><small style="color:var(--t3)"> 先付</small></a-radio>
        <div v-for="a in areas" :key="a.id">
          <div style="margin:12px 0 6px;font-weight:600;color:var(--t2)">{{ a.name }}<small style="color:var(--t3);font-weight:400"> · 默认{{ a.defaultPayMode === 'postpay' ? '后付' : '先付' }}</small></div>
          <a-space wrap>
            <a-radio v-for="t in a.tables" :key="t.id" :value="t.id" :disabled="!!t.session?.lockedForBill">
              {{ t.tableNo }}<small style="color:var(--t3)">{{ t.status === 'occupied' ? '·就餐中' : '·空闲' }}{{ t.payMode === 'postpay' ? '·后付' : '' }}</small>
            </a-radio>
          </a-space>
        </div>
      </a-radio-group>
      <div style="margin-top:18px;text-align:right">
        <a-button type="primary" @click="tableOpen = false">确定</a-button>
      </div>
    </a-modal>

    <!-- 收款确认 -->
    <a-modal v-model:open="payOpen" title="收款确认" :footer="null" width="400">
      <div class="pay-card">
        <div class="pay-t">应收金额</div>
        <div class="pay-am num">¥{{ total.toFixed(2) }}</div>
        <span class="tag" :class="payMode === 'postpay' ? 'p' : 'd'">{{ modeLabel }}</span>
      </div>
      <div style="margin-top:16px">
        <div class="pay-l">收款方式</div>
        <a-radio-group v-model:value="payChannel" style="display:flex;flex-wrap:wrap;gap:8px">
          <a-radio-button v-for="(label, ch) in ({ wechat: '微信支付', alipay: '支付宝', cash: '现金', pos: 'POS' } as Record<string, string>)" :key="ch" :value="ch" :disabled="payMode === 'postpay'">{{ label }}</a-radio-button>
        </a-radio-group>
      </div>
      <div v-if="payMode === 'postpay'" class="pay-note">后付桌台：本单挂账到会话，清台前统一结账，收款方式在结账时选择。</div>
      <div style="margin-top:20px;text-align:right">
        <a-button style="margin-right:8px" @click="payOpen = false">取消</a-button>
        <a-button type="primary" @click="submitOrder">确认{{ payMode === 'postpay' ? '挂账下单' : '收款下单' }}</a-button>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.pos-grid { display: grid; grid-template-columns: 150px 1fr 340px; gap: 16px; align-items: start }
.cat-item { padding: 13px 16px; font-size: 14px; color: var(--t2); cursor: pointer; border-left: 3px solid transparent; transition: .12s }
.cat-item:hover { background: var(--hover); color: var(--t1) }
.cat-item.on { color: var(--brand); border-left-color: var(--brand); background: var(--brand-bg); font-weight: 500 }
.dish-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 10px }
.dish { position: relative; border: 1px solid var(--split); border-radius: 8px; padding: 10px; cursor: pointer; transition: .12s; background: var(--card) }
.dish:hover { border-color: var(--brand); box-shadow: var(--sh) }
.dish.off { opacity: .45; cursor: not-allowed }
.d-name { font-size: 14px; font-weight: 500 }
.d-meta { font-size: 11px; color: var(--t3); margin: 2px 0 6px }
.d-price b { font-size: 15px; color: var(--er) }
.d-price small { font-size: 11px; color: var(--t3) }
.soldout { position: absolute; top: 6px; right: 6px; font-size: 10px; color: #fff; background: var(--er); border-radius: 3px; padding: 1px 5px }
.dq { position: absolute; top: -6px; right: -6px; min-width: 20px; height: 20px; border-radius: 10px; background: var(--brand); color: #fff; font-size: 12px; display: grid; place-items: center; padding: 0 5px }
.empty { color: var(--t3); font-size: 13px; text-align: center; padding: 40px 0 }
.cline { display: flex; align-items: center; gap: 8px; padding: 8px 0; border-bottom: 1px dashed var(--split); font-size: 13px }
.cline:last-of-type { border-bottom: none }
.cname { flex: 1; min-width: 0 }
.cname b { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap }
.cname small { color: var(--wn); font-size: 11px }
.stepper { display: flex; align-items: center; gap: 6px }
.stepper b { min-width: 18px; text-align: center }
.cline em { font-style: normal; font-weight: 600; min-width: 58px; text-align: right }
.card-foot { border-top: 1px solid var(--split); padding: 14px 20px }
.tot { display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 10px }
.tot b { font-size: 22px }
.tot-op { display: flex; gap: 8px }
.tot-op .ant-btn { flex: 1 }
.pay-card { background: var(--brand-bg); border-radius: 10px; padding: 18px; text-align: center }
.pay-t { font-size: 13px; color: var(--t3) }
.pay-am { font-size: 34px; font-weight: 700; color: var(--er); margin: 4px 0 8px }
.pay-l { font-size: 13px; color: var(--t2); margin-bottom: 8px }
.pay-note { margin-top: 12px; padding: 8px 10px; background: var(--wn-bg); border-radius: 6px; font-size: 12px; color: var(--wn) }
</style>
