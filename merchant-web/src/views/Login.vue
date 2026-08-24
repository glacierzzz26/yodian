<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { useAuthStore } from '@/stores/auth'
import { firstPath } from '@/router/menu'
import type { Role } from '@/types'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const rolePick = ref<Role>('owner')
const employeeNo = ref('1001')
const password = ref('123456')
const loading = ref(false)

const ROLE_LABEL: Record<Role, { name: string; tip: string }> = {
  owner: { name: '店长', tip: '全部权限' },
  cashier: { name: '收银员', tip: '收银台' },
  kitchen: { name: '后厨', tip: 'KDS' },
}

async function doLogin() {
  if (!employeeNo.value || !password.value) {
    message.warning('请输入工号和密码')
    return
  }
  loading.value = true
  try {
    await auth.mockLogin(employeeNo.value, password.value, rolePick.value)
    message.success('登录成功')
    const redirect = (route.query.redirect as string) || firstPath(rolePick.value)
    router.push(redirect)
  } catch (e) {
    message.error((e as Error).message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login">
    <div class="lg-card">
      <div class="lg-logo">◧</div>
      <div class="lg-t">悦点 · 商家端</div>
      <div class="lg-s">扫码点餐管理系统 v1.0</div>

      <div class="lg-role">
        <div
          v-for="r in (['owner','cashier','kitchen'] as Role[])" :key="r"
          :class="{ on: rolePick === r }"
          @click="rolePick = r"
        >
          <b>{{ ROLE_LABEL[r].name }}</b><small>{{ ROLE_LABEL[r].tip }}</small>
        </div>
      </div>

      <div class="lg-f"><label>工号</label>
        <input v-model="employeeNo" class="lg-in" placeholder="如 1001" /></div>
      <div class="lg-f"><label>密码</label>
        <input v-model="password" class="lg-in" type="password" placeholder="默认 123456" @keyup.enter="doLogin" /></div>

      <button class="lg-btn" :disabled="loading" @click="doLogin">{{ loading ? '登录中…' : '登 录' }}</button>

      <div class="lg-tip"><b>开发期说明</b>：当前为 mock 登录（任意工号 + 123456），选择角色决定可见菜单（店长可见全部）。后端 /auth/login 就绪后切换真实鉴权。</div>
      <div class="lg-fs">原型构建版本：2026-08-24 · v1.8 设计基线</div>
    </div>
  </div>
</template>

<style scoped>
.login { position: fixed; inset: 0; z-index: 900; display: flex; align-items: center; justify-content: center;
  background: linear-gradient(135deg,#001529 0%,#003a70 55%,#0958d9 100%) }
.login::after { content: ''; position: absolute; inset: 0;
  background: radial-gradient(circle at 18% 22%, rgba(255,255,255,.09), transparent 42%),
    radial-gradient(circle at 82% 78%, rgba(255,255,255,.07), transparent 40%) }
.lg-card { position: relative; z-index: 2; width: 400px; background: #fff; border-radius: 12px; padding: 40px 36px;
  box-shadow: 0 12px 48px rgba(0,0,0,.28) }
.lg-logo { width: 52px; height: 52px; border-radius: 12px; background: linear-gradient(135deg,#1677ff,#0958d9);
  display: grid; place-items: center; color: #fff; font-size: 26px; margin: 0 auto 16px }
.lg-t { text-align: center; font-size: 22px; font-weight: 600; color: rgba(0,0,0,.88); letter-spacing: .5px }
.lg-s { text-align: center; font-size: 13px; color: rgba(0,0,0,.45); margin-top: 6px; margin-bottom: 28px }
.lg-role { display: flex; gap: 8px; margin-bottom: 18px }
.lg-role > div { flex: 1; height: 48px; display: grid; place-items: center; align-content: center; gap: 1px;
  border: 1px solid #d9d9d9; border-radius: 6px; font-size: 13px; color: rgba(0,0,0,.65); cursor: pointer; transition: .15s }
.lg-role > div small { font-size: 11px; color: rgba(0,0,0,.35) }
.lg-role > div.on { border-color: #1677ff; color: #1677ff; background: #e6f4ff }
.lg-role > div.on small { color: #1677ff }
.lg-f { margin-bottom: 16px }
.lg-f label { display: block; font-size: 13px; color: rgba(0,0,0,.65); margin-bottom: 6px }
.lg-in { width: 100%; height: 40px; padding: 0 12px; border: 1px solid #d9d9d9; border-radius: 6px;
  color: rgba(0,0,0,.88); background: #fff; transition: .2s; outline: none; font-size: 14px }
.lg-in:focus { border-color: #1677ff; box-shadow: 0 0 0 2px rgba(22,119,255,.1) }
.lg-btn { width: 100%; height: 42px; margin-top: 10px; border: none; border-radius: 6px; cursor: pointer;
  background: #1677ff; color: #fff; font-size: 15px; font-weight: 500; transition: .2s }
.lg-btn:hover:not(:disabled) { background: #4096ff; box-shadow: 0 4px 12px rgba(22,119,255,.35) }
.lg-btn:disabled { opacity: .6; cursor: not-allowed }
.lg-tip { margin-top: 20px; padding: 10px 12px; background: #e6f4ff; border: 1px solid #91caff; border-radius: 6px;
  font-size: 12px; color: #0958d9; line-height: 1.7 }
.lg-fs { font-size: 11px; color: rgba(255,255,255,.45); margin-top: 10px; text-align: center }
</style>
