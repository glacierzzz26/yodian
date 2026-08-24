<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { visibleGroups } from '@/router/menu'
import { fetchDashboard, fetchPrintTasks } from '@/api/mock'
import type { Role } from '@/types'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()

const groups = computed(() => visibleGroups(auth.role as Role))
const currentTitle = computed(() => {
  const path = route.path
  for (const g of groups.value) {
    const it = g.items.find((i) => i.path === path)
    if (it) return it.label
  }
  return '工作台'
})

const badges = ref<Record<string, number>>({})
onMounted(async () => {
  try {
    const dash = await fetchDashboard()
    const prints = await fetchPrintTasks()
    badges.value = {
      tables: dash.unsettledTables,
      manual: dash.manualPending,
      print: prints.filter((p) => p.status === 'failed').length,
    }
  } catch { /* 徽标加载失败不阻塞界面 */ }
})

function onMenuClick(path: string) {
  router.push(path)
}

function onDegradeToggle() {
  if (auth.role !== 'owner') return
  Modal.confirm({
    title: app.degrade ? '关闭降级模式' : '开启降级模式',
    content: app.degrade
      ? '关闭后顾客扫码下单入口恢复。'
      : '开启后顾客扫码下单入口关闭，新单请在「补录人工单」录入，收款走现金/POS。',
    okText: '确认',
    cancelText: '取消',
    onOk() {
      app.degrade = !app.degrade
      message[app.degrade ? 'warning' : 'success'](app.degrade ? '已开启降级模式' : '已关闭降级模式')
    },
  })
}

function onLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <a-layout style="min-height:100vh">
    <a-layout-sider
      :collapsed="app.siderCollapsed"
      :width="216"
      :collapsed-width="64"
      theme="dark"
      style="background:#001529"
    >
      <div class="brand">
        <i>◧</i>
        <b v-show="!app.siderCollapsed">悦点 · 商家端</b>
      </div>
      <div class="menu">
        <template v-for="g in groups" :key="g.title">
          <div class="mg-t" v-show="!app.siderCollapsed">{{ g.title }}</div>
          <div
            v-for="it in g.items" :key="it.key"
            class="mi" :class="{ on: route.path === it.path }"
            @click="onMenuClick(it.path)"
            :title="it.label"
          >
            <u>{{ it.icon }}</u>
            <span>{{ it.label }}</span>
            <em v-if="badges[it.key]">{{ badges[it.key] }}</em>
          </div>
        </template>
      </div>
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="hd">
        <div class="hd-ic" @click="app.toggleSider()" title="收起/展开侧边栏">☰</div>
        <div class="crumb">商家端 / <b>{{ currentTitle }}</b></div>
        <div class="hd-r">
          <div class="biz" title="营业时段见系统设置">
            <i></i><span>晚市营业中 · 17:00–21:30</span>
          </div>
          <div v-if="auth.role === 'owner'" class="hd-ic" @click="onDegradeToggle()" title="降级模式开关" :class="{ active: app.degrade }">⚡</div>
          <div class="hd-ic" @click="app.toggleTheme()" title="亮色/暗色切换">{{ app.isDark ? '☀' : '☾' }}</div>
          <div class="who" @click="onLogout()" title="退出登录">
            <div class="avt">{{ (auth.operator?.name || '?').charAt(0) }}</div>
            <div>
              <span>{{ auth.operator?.name }}</span>
              <small>{{ auth.operator?.role === 'owner' ? '店长' : auth.operator?.role === 'cashier' ? '收银员' : '后厨' }} · 工号{{ auth.operator?.employeeNo }}</small>
            </div>
          </div>
        </div>
      </a-layout-header>

      <div v-if="app.degrade" class="degrade-banner">
        <span>⚡</span>
        <div><b>降级模式已开启</b> · 顾客扫码下单入口已关闭，新单请在「补录人工单」录入，收款走现金/POS</div>
        <div class="br">
          <a-button size="small" @click="router.push('/offline')">去补录</a-button>
          <a-button size="small" danger @click="onDegradeToggle()">关闭降级</a-button>
        </div>
      </div>

      <a-layout-content>
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.brand { height: 56px; display: flex; align-items: center; gap: 10px; padding: 0 18px; flex-shrink: 0;
  border-bottom: 1px solid rgba(255,255,255,.08); overflow: hidden; }
.brand i { width: 30px; height: 30px; border-radius: 7px; background: linear-gradient(135deg,#1677ff,#0958d9);
  display: grid; place-items: center; font-size: 16px; font-style: normal; color: #fff; flex-shrink: 0; }
.brand b { color: #fff; font-size: 14px; font-weight: 600; white-space: nowrap; letter-spacing: .3px; }
.menu { overflow-y: auto; padding: 8px 0 20px; height: calc(100vh - 56px); }
.menu::-webkit-scrollbar { width: 4px }
.mg-t { padding: 14px 20px 6px; font-size: 11px; color: rgba(255,255,255,.32); letter-spacing: 1px; white-space: nowrap; }
.mi { display: flex; align-items: center; gap: 10px; height: 40px; padding: 0 20px; color: rgba(255,255,255,.65);
  cursor: pointer; font-size: 14px; position: relative; transition: .15s; white-space: nowrap; }
.mi:hover { color: #fff; background: rgba(255,255,255,.06) }
.mi.on { color: #fff; background: #1677ff }
.mi.on::before { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 3px; background: #fff }
.mi u { width: 18px; text-align: center; font-size: 15px; text-decoration: none; flex-shrink: 0; }
.mi span { flex: 1; overflow: hidden; text-overflow: ellipsis; }
.mi em { margin-left: auto; font-style: normal; font-size: 11px; min-width: 18px; height: 18px; padding: 0 5px;
  border-radius: 9px; background: #ff4d4f; color: #fff; display: grid; place-items: center; font-weight: 600; }
.mi.on em { background: #fff; color: #1677ff }

.hd { position: sticky; top: 0; z-index: 70; height: 56px; background: var(--card); border-bottom: 1px solid var(--split);
  display: flex; align-items: center; gap: 14px; padding: 0 20px; line-height: normal; }
.hd-ic { width: 32px; height: 32px; border-radius: var(--r); display: grid; place-items: center; cursor: pointer;
  color: var(--t2); font-size: 16px; transition: .15s; }
.hd-ic:hover { background: var(--hover); color: var(--t1) }
.hd-ic.active { color: #faad14; background: var(--wn-bg) }
.crumb { font-size: 14px; color: var(--t3) }
.crumb b { color: var(--t1); font-weight: 500 }
.hd-r { margin-left: auto; display: flex; align-items: center; gap: 12px }
.biz { display: flex; align-items: center; gap: 6px; height: 28px; padding: 0 10px; border-radius: 14px;
  background: var(--ok-bg); border: 1px solid var(--ok-bd); font-size: 12px; color: var(--ok); font-weight: 500 }
.biz i { width: 6px; height: 6px; border-radius: 50%; background: var(--ok); animation: bp 1.6s infinite }
@keyframes bp { 0%,100% { opacity: 1 } 50% { opacity: .35 } }
.who { display: flex; align-items: center; gap: 8px; padding: 4px 10px 4px 4px; border-radius: 16px; cursor: pointer; }
.who:hover { background: var(--hover) }
.avt { width: 28px; height: 28px; border-radius: 50%; background: #1677ff; color: #fff; display: grid;
  place-items: center; font-size: 12px; font-weight: 600 }
.who span { font-size: 13px; color: var(--t1) }
.who small { font-size: 11px; color: var(--t3); display: block; line-height: 1.2 }
</style>
