<template>
  <view class="orders">
    <!-- 桌台胶囊 + 会话状态 -->
    <view class="hd">
      <view class="pill">
        <text class="tno">{{ tableNo }} 号桌</text>
        <text class="tsep">·</text>
        <text class="tmode">{{ payModeLabel }}</text>
        <text class="tsep">·</text>
        <text class="tstat" :class="{ settled: sessionStatus === 'settled' }">{{ sessionStatus === 'settled' ? '已结账' : '就餐中' }}</text>
      </view>
      <view class="refresh" @click="load">刷新</view>
    </view>
    <view class="pax" v-if="data">共 {{ data.pax }} 人 · {{ data.orders.length }} 笔订单</view>

    <scroll-view class="list" scroll-y>
      <template v-if="data && data.orders.length">
        <view v-for="o in data.orders" :key="o.id" class="card">
          <view class="o-hd">
            <text class="oid">单 {{ o.id }}</text>
            <text class="ostatus" :class="'s-' + o.status">{{ orderStatusLabel(o.status) }}</text>
            <text v-if="o.channel" class="och">{{ channelLabel(o.channel) }}</text>
          </view>
          <view v-for="(it, i) in o.items" :key="i" class="line">
            <view class="l-info">
              <view class="l-name">{{ it.name }}<text v-if="it.item_type === 'per_head'" class="l-tag">按人</text></view>
              <view v-if="specText(it)" class="l-spec">{{ specText(it) }}</view>
            </view>
            <view class="l-right">
              <text class="l-price">¥{{ fen2yuan(it.price) }}</text>
              <text class="l-qty">× {{ it.qty }}</text>
            </view>
          </view>
          <view class="o-ft">
            <text class="l-sum">合计 ¥{{ fen2yuan(o.total_amount) }}</text>
            <wd-button v-if="o.customer_id === auth.customerId && o.status === 'pending'" size="small" type="primary" custom-class="paybtn" @click="pay(o.id)">去支付</wd-button>
          </view>
        </view>
      </template>
      <view v-else-if="data" class="empty">本桌暂无订单</view>
    </scroll-view>

    <!-- 后付/前台结：统一结账入口 -->
    <view v-if="canCheckout" class="footer">
      <wd-button block type="primary" custom-class="co" @click="goCheckout">去结账</wd-button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useSessionStore } from '@/stores/session'
import { useAuthStore } from '@/stores/auth'
import { request } from '@/utils/request'
import { fen2yuan } from '@/utils/money'
import { orderStatusLabel, channelLabel } from '@/utils/status'
import type { SessionOrdersResp } from '@/types/api'

const session = useSessionStore()
const auth = useAuthStore()

const data = ref<SessionOrdersResp | null>(null)

const tableNo = computed(() => session.table?.table_no || '—')
const payModeLabel = computed(() => {
  const m = session.table?.pay_mode
  return m === 'prepay' ? '先付' : m === 'postpay' ? '后付' : m === 'frontend' ? '前台结' : '—'
})
const sessionStatus = computed(() => data.value?.status || 'active')
const canCheckout = computed(
  () => !!session.sid && data.value?.status === 'active' && data.value.pay_mode !== 'prepay',
)

function specText(it: { specs?: unknown }) {
  const s = it.specs
  if (!s) return ''
  if (typeof s === 'string') return s
  try {
    const obj = s as Record<string, string>
    if (obj && typeof obj === 'object') return Object.values(obj).join(' · ')
  } catch {
    return ''
  }
  return ''
}

async function load() {
  if (!session.sid) return
  try {
    data.value = await request<SessionOrdersResp>({ url: `/sessions/${session.sid}/orders` })
  } catch {
    // 服务端已 toast
  }
}

function pay(orderId: number) {
  uni.navigateTo({ url: `/pages/pay/index?order_id=${orderId}` })
}
function goCheckout() {
  uni.navigateTo({ url: '/pages/checkout/index' })
}

onLoad(() => {
  if (!session.sid) {
    uni.showToast({ title: '下单后即可查看本桌订单', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 600)
    return
  }
  load()
})
onShow(() => {
  // 从支付页返回后刷新状态（弱网补单 + 结账后回显）
  if (session.sid) load()
})
</script>

<style lang="scss" scoped>
.orders {
  min-height: 100vh;
  background: $bg;
  padding: 24rpx 24rpx calc(40rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
}
.hd {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.pill {
  display: flex;
  align-items: center;
  padding: 12rpx 28rpx;
  border-radius: 999rpx;
  background: #fff;
  border: 2rpx solid #f0e6de;
}
.tno {
  font-size: 30rpx;
  font-weight: 700;
  color: $ink;
}
.tsep {
  margin: 0 12rpx;
  color: #d1d5db;
}
.tmode {
  font-size: 24rpx;
  color: $brand;
  background: $brand-light;
  padding: 4rpx 14rpx;
  border-radius: 999rpx;
}
.tstat {
  font-size: 24rpx;
  color: #16a34a;
}
.tstat.settled {
  color: #9ca3af;
}
.refresh {
  font-size: 26rpx;
  color: $brand;
  padding: 8rpx 16rpx;
}
.pax {
  margin: 16rpx 8rpx 4rpx;
  font-size: 24rpx;
  color: #9ca3af;
}
.list {
  height: calc(100vh - 320rpx);
}
.card {
  background: #fff;
  border-radius: 20rpx;
  padding: 24rpx 28rpx;
  margin-top: 20rpx;
}
.o-hd {
  display: flex;
  align-items: center;
  border-bottom: 2rpx solid #f5efe9;
  padding-bottom: 16rpx;
}
.oid {
  font-size: 28rpx;
  font-weight: 600;
  color: $ink;
}
.ostatus {
  margin-left: 16rpx;
  font-size: 22rpx;
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
  background: #f4f4f5;
  color: #71717a;
}
.ostatus.s-pending {
  background: $brand-light;
  color: $brand;
}
.ostatus.s-paid,
.ostatus.s-done {
  background: #f0fdf4;
  color: #16a34a;
}
.ostatus.s-preparing,
.ostatus.s-served {
  background: #eff6ff;
  color: #2563eb;
}
.och {
  margin-left: auto;
  font-size: 22rpx;
  color: $sub;
}
.line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14rpx 0;
}
.l-name {
  font-size: 28rpx;
  color: $ink;
}
.l-tag {
  margin-left: 12rpx;
  font-size: 20rpx;
  color: $sub;
  background: $bg;
  padding: 2rpx 12rpx;
  border-radius: 999rpx;
}
.l-spec {
  font-size: 22rpx;
  color: #9ca3af;
  margin-top: 4rpx;
}
.l-right {
  display: flex;
  align-items: baseline;
}
.l-price {
  font-size: 26rpx;
  color: $ink;
}
.l-qty {
  margin-left: 12rpx;
  font-size: 24rpx;
  color: #9ca3af;
}
.o-ft {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 2rpx solid #f5efe9;
  padding-top: 16rpx;
  margin-top: 4rpx;
}
.l-sum {
  font-size: 30rpx;
  font-weight: 700;
  color: $ink;
}
.paybtn {
  border-radius: 999rpx;
}
.empty {
  padding: 100rpx 0;
  text-align: center;
  color: #9ca3af;
  font-size: 26rpx;
}
.footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 16rpx 24rpx calc(16rpx + env(safe-area-inset-bottom));
  background: rgba(250, 246, 242, 0.96);
}
.co {
  border-radius: 999rpx;
}
</style>
