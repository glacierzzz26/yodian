<template>
  <view class="pay">
    <!-- 加载/预下单 -->
    <view v-if="state === 'loading'" class="body">
      <view class="spin"></view>
      <text class="tip">正在发起支付…</text>
    </view>

    <!-- 待支付：mock 直接给「模拟支付成功」；真实渠道则调 SDK -->
    <view v-else-if="state === 'ready'" class="body">
      <view class="amt"><text class="cur">¥</text>{{ fen2yuan(amount) }}</view>
      <text class="label">待支付金额{{ mode === 'bill' ? '（本桌结账单）' : '' }}</text>

      <view v-if="isMock" class="mock-note">当前为开发模拟支付（MOCK 网关），点击下方按钮模拟支付成功回调。</view>

      <view class="btns">
        <wd-button
          v-if="isMock"
          block
          type="primary"
          size="large"
          custom-class="btn"
          @click="mockPay"
        >
          模拟支付成功
        </wd-button>
        <template v-else>
          <wd-button block type="primary" size="large" custom-class="btn" @click="doPay">立即支付</wd-button>
          <wd-button block plain type="primary" custom-class="btn ghost" @click="start">支付遇到问题，重新发起</wd-button>
        </template>
        <!-- 换渠道重付：仅订单模式（时序图 4，3.5）；结账单走同渠道 -->
        <wd-button
          v-if="mode === 'order'"
          block
          plain
          type="primary"
          custom-class="btn ghost"
          @click="switchChannel"
        >
          切换为{{ otherChannel }}支付
        </wd-button>
      </view>
      <view class="channel" v-if="mode === 'order'">当前渠道：{{ payChannel === 'wechat' ? '微信支付' : '支付宝支付' }}</view>
    </view>

    <!-- 轮询中 -->
    <view v-else-if="state === 'polling'" class="body">
      <view class="spin"></view>
      <text class="tip">{{ mode === 'bill' ? '正在确认结账结果…' : '正在确认支付结果…' }}</text>
      <text class="refresh" @click="queryNow">手动刷新</text>
    </view>

    <!-- 支付成功 -->
    <view v-else-if="state === 'done'" class="body">
      <view class="ok">✓</view>
      <view class="amt ok-amt"><text class="cur">¥</text>{{ fen2yuan(amount) }}</view>
      <text class="label">{{ mode === 'bill' ? '结账成功，欢迎再次光临' : '支付成功，后厨已开始制作' }}</text>
      <wd-button block type="primary" size="large" custom-class="btn" @click="backMenu">
        {{ mode === 'bill' ? '返回菜单' : '返回菜单继续点餐' }}
      </wd-button>
    </view>

    <!-- 失败 -->
    <view v-else class="body">
      <view class="bad">!</view>
      <text class="label">{{ mode === 'bill' ? '结账未完成或已失败' : '支付未完成或已失败' }}</text>
      <wd-button block type="primary" size="large" custom-class="btn" @click="start">重新支付</wd-button>
      <wd-button block plain type="primary" custom-class="btn ghost" @click="backMenu">返回菜单</wd-button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { request } from '@/utils/request'
import { requestPayment, type PaymentParams } from '@/utils/pay'
import { pollOrderStatus, pollSessionSettled, type Poller } from '@/utils/poller'
import { fen2yuan } from '@/utils/money'
import { useSessionStore } from '@/stores/session'
import { useAuthStore } from '@/stores/auth'
import type { PrepayResp, OrderStatusResp, SwitchResp, SessionOrdersResp } from '@/types/api'

const session = useSessionStore()
const auth = useAuthStore()

const mode = ref<'order' | 'bill'>('order')
const orderId = ref(0)
const billId = ref(0)
const state = ref<'loading' | 'ready' | 'polling' | 'done' | 'fail'>('loading')
const amount = ref(0)
const outTradeNo = ref('')
const params = ref<Record<string, unknown>>({})
const isMock = ref(false)
const payChannel = ref(auth.channel || 'wechat')
let poller: Poller | null = null

const otherChannel = computed(() => (payChannel.value === 'wechat' ? '支付宝' : '微信'))

function queryNow() {
  // 手动查单（弱网补单，9.8）：订单/会话均以服务端终态为准
  const q = mode.value === 'order' ? queryOrderNow() : querySessionNow()
  q.catch(() => {})
}
function queryOrderNow() {
  return request<OrderStatusResp>({ url: `/orders/${orderId.value}/status` }).then((s) => {
    if (s.status === 'paid' || s.status === 'done') {
      amount.value = s.total_amount
      state.value = 'done'
    } else if (s.status === 'refunded') {
      state.value = 'fail'
    }
  })
}
function querySessionNow() {
  return request<SessionOrdersResp>({ url: `/sessions/${session.sid}/orders` }).then((s) => {
    if (s.status === 'settled') state.value = 'done'
  })
}

function startPoll() {
  stopPoll()
  if (mode.value === 'order') {
    poller = pollOrderStatus(orderId.value, (s) => {
      if (s.status === 'paid' || s.status === 'done') {
        amount.value = s.total_amount
        state.value = 'done'
      } else if (s.status === 'refunded') {
        state.value = 'fail'
      }
    })
  } else {
    poller = pollSessionSettled(session.sid || 0, (status) => {
      if (status === 'settled') state.value = 'done'
    })
  }
  poller.start()
}
function stopPoll() {
  if (poller) {
    poller.stop()
    poller = null
  }
}

// 模拟支付成功（MOCK 网关）：触发与真实渠道完全相同的内部回调 → 轮询定终态
async function mockPay() {
  state.value = 'polling'
  try {
    await request({
      url: '/pay/notify/mock',
      method: 'POST',
      data: {
        channel: payChannel.value,
        out_trade_no: outTradeNo.value,
        amount: amount.value,
        channel_trade_no: 'MOCK' + outTradeNo.value,
      },
    })
    startPoll()
  } catch {
    state.value = 'fail'
  }
}

// 真实渠道：调起支付，成功后轮询
async function doPay() {
  try {
    await requestPayment(params.value as unknown as PaymentParams)
  } catch {
    uni.showToast({ title: '支付未完成', icon: 'none' })
    return
  }
  startPoll()
}

// 换渠道重付（时序图 4）：生成新 out_trade_no + 新渠道预下单；旧单号迟到回调天然不双入账（3.1 MVP）
async function switchChannel() {
  const to = payChannel.value === 'wechat' ? 'alipay' : 'wechat'
  state.value = 'loading'
  try {
    const resp = await request<SwitchResp>({
      url: '/pay/switch',
      method: 'POST',
      data: { order_id: orderId.value, to },
    })
    outTradeNo.value = resp.out_trade_no
    amount.value = resp.amount
    params.value = resp.params || {}
    payChannel.value = to
    isMock.value = !!(resp.params && resp.params.mock === true)
    state.value = 'ready'
    uni.showToast({ title: `已切换为${to === 'wechat' ? '微信' : '支付宝'}支付`, icon: 'none' })
  } catch {
    state.value = 'fail'
  }
}

async function start() {
  if (mode.value === 'order' && !orderId.value) return
  if (mode.value === 'bill' && !billId.value) return
  stopPoll()
  state.value = 'loading'
  if (mode.value === 'order') {
    // 先查单：已支付/终态直接落终态（弱网补单语义，9.8）
    try {
      const st = await request<OrderStatusResp>({ url: `/orders/${orderId.value}/status` })
      if (st.status === 'paid' || st.status === 'done') {
        amount.value = st.total_amount
        state.value = 'done'
        return
      }
      if (st.status === 'refunded') {
        state.value = 'fail'
        return
      }
    } catch {
      // 查单失败不阻断，继续 prepay
    }
  }
  try {
    const resp = await request<PrepayResp>({
      url: '/pay/prepay',
      method: 'POST',
      data: mode.value === 'order'
        ? { order_id: orderId.value, channel: payChannel.value }
        : { bill_id: billId.value, channel: payChannel.value },
    })
    outTradeNo.value = resp.out_trade_no
    amount.value = resp.amount
    params.value = resp.params || {}
    isMock.value = !!(resp.params && resp.params.mock === true)
    state.value = 'ready'
  } catch {
    state.value = 'fail'
  }
}

function backMenu() {
  uni.redirectTo({ url: '/pages/menu/index' })
}

onLoad((options) => {
  const oid = Number(options?.order_id || 0)
  const bid = Number(options?.bill_id || 0)
  if (oid) {
    mode.value = 'order'
    orderId.value = oid
  } else if (bid) {
    mode.value = 'bill'
    billId.value = bid
    if (!session.sid) {
      uni.showToast({ title: '参数错误', icon: 'none' })
      setTimeout(() => uni.reLaunch({ url: '/pages/menu/index' }), 600)
      return
    }
  } else {
    uni.showToast({ title: '参数错误', icon: 'none' })
    setTimeout(() => uni.reLaunch({ url: '/pages/menu/index' }), 600)
    return
  }
  start()
})

onUnload(() => {
  stopPoll()
})
</script>

<style lang="scss" scoped>
.pay {
  min-height: 100vh;
  background: $bg;
  display: flex;
  align-items: center;
  justify-content: center;
}
.body {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0 48rpx;
  width: 100%;
  box-sizing: border-box;
}
.spin {
  width: 64rpx;
  height: 64rpx;
  border: 6rpx solid #f0e6de;
  border-top-color: $brand;
  border-radius: 50%;
  animation: rot 0.9s linear infinite;
}
@keyframes rot {
  to {
    transform: rotate(360deg);
  }
}
.tip {
  margin-top: 32rpx;
  color: $sub;
  font-size: 28rpx;
}
.refresh {
  margin-top: 24rpx;
  color: $brand;
  font-size: 26rpx;
  padding: 8rpx 24rpx;
}
.ok,
.bad {
  width: 112rpx;
  height: 112rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 56rpx;
  font-weight: 700;
  color: #fff;
  margin-bottom: 32rpx;
}
.ok {
  background: #22c55e;
}
.bad {
  background: #ef4444;
}
.amt {
  font-size: 72rpx;
  font-weight: 700;
  color: $ink;
}
.amt.ok-amt {
  color: #16a34a;
  margin-bottom: 8rpx;
}
.cur {
  font-size: 40rpx;
  margin-right: 4rpx;
}
.label {
  margin-top: 12rpx;
  color: $sub;
  font-size: 28rpx;
}
.mock-note {
  margin-top: 32rpx;
  font-size: 24rpx;
  color: #9ca3af;
  text-align: center;
  line-height: 1.6;
}
.btns {
  width: 100%;
  margin-top: 48rpx;
}
.btn {
  border-radius: 999rpx;
}
.btn.ghost {
  margin-top: 20rpx;
}
.channel {
  margin-top: 24rpx;
  font-size: 24rpx;
  color: #9ca3af;
}
</style>
