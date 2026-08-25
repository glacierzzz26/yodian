<template>
  <view class="page">
    <!-- 加载态：静默登录 + 桌码签名校验 -->
    <view v-if="state === 'loading'" class="body">
      <view class="logo">悦点</view>
      <text class="title">正在进入桌台…</text>
      <text v-if="devInfo" class="dev">已登录 customer #{{ devInfo }}</text>
    </view>

    <!-- 二维码无效：缺参 / sig 校验失败（后端 20001） -->
    <view v-else-if="state === 'invalid'" class="body">
      <view class="logo bad">?</view>
      <text class="title">二维码无效</text>
      <text class="sub">桌贴二维码不完整或已失效，请联系服务员协助处理。</text>
      <wd-button size="small" plain type="primary" custom-class="btn" @click="showHint">重新扫码</wd-button>
    </view>

    <!-- 网络异常：登录/校验请求失败（非业务码） -->
    <view v-else class="body">
      <view class="logo bad">!</view>
      <text class="title">网络异常</text>
      <text class="sub">无法连接服务，请检查网络后重试。</text>
      <wd-button size="small" type="primary" custom-class="btn" @click="retry">重试</wd-button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { parseEntry } from '@/utils/entry'
import { useAuthStore } from '@/stores/auth'
import { useSessionStore } from '@/stores/session'
import type { BizError } from '@/utils/request'

const auth = useAuthStore()
const session = useSessionStore()
const state = ref<'loading' | 'invalid' | 'error'>('loading')
const devInfo = ref('')

async function boot() {
  state.value = 'loading'
  devInfo.value = ''
  try {
    // 1. 静默登录（不索取昵称/头像，9.4；401 由 request 层静默重登）
    if (!auth.token) await auth.login()
    devInfo.value = String(auth.customerId ?? '')
    // 2. 校验桌码签名并拉取桌台/菜单（sig 无效 → 20001 BizError → 无效屏）
    await session.fetchMenu()
    // 3. 校验通过 → 进菜单（阶段 3.4 实现点餐主链路）
    uni.redirectTo({ url: '/pages/menu/index' })
  } catch (e) {
    const biz = e as BizError
    state.value = biz && biz.__biz ? 'invalid' : 'error'
  }
}

onLoad((options) => {
  const p = parseEntry(options || {})
  if (!p.tid || !p.sig) {
    state.value = 'invalid'
    return
  }
  session.setEntry(Number(p.tid), p.sig)
  boot()
})

function retry() {
  boot()
}
function showHint() {
  uni.showToast({ title: '请在微信或支付宝中重新扫描桌贴二维码', icon: 'none' })
}
</script>

<style lang="scss" scoped>
.page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: $bg;
}
.body {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 48rpx;
}
.logo {
  width: 128rpx;
  height: 128rpx;
  border-radius: 32rpx;
  background: $brand;
  color: #fff;
  font-size: 40rpx;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  letter-spacing: 4rpx;
  margin-bottom: 32rpx;
}
.logo.bad {
  background: #f4f4f5;
  color: #a1a1aa;
}
.title {
  font-size: 36rpx;
  font-weight: 600;
  color: $ink;
}
.sub {
  margin-top: 16rpx;
  font-size: 26rpx;
  line-height: 1.6;
  color: $sub;
  text-align: center;
}
.dev {
  margin-top: 24rpx;
  font-size: 22rpx;
  color: #9ca3af;
}
.btn {
  margin-top: 40rpx;
}
</style>
