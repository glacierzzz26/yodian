<template>
  <view class="co">
    <!-- 加载中 -->
    <view v-if="loading" class="body"><view class="spin"></view><text class="tip">正在计算账单…</text></view>

    <!-- 无可结账 -->
    <view v-else-if="nothingToPay" class="body">
      <view class="ok">✓</view>
      <text class="label">{{ settled ? '本桌已结账' : '暂无待结账订单' }}</text>
      <wd-button block type="primary" size="large" custom-class="btn" @click="backMenu">返回菜单</wd-button>
    </view>

    <template v-else>
      <!-- 账单预览 -->
      <view class="card">
        <view class="c-hd">
          <text class="c-title">{{ tableNo }} 号桌 · {{ payModeLabel }}</text>
          <text class="c-pax">{{ preview.pax }} 人 · {{ preview.orders_count }} 笔</text>
        </view>
        <view class="row">
          <text>菜品合计</text><text>¥{{ fen2yuan(preview.dishes_amount) }}</text>
        </view>
        <view v-for="p in preview.per_head_items" :key="p.id" class="row sub">
          <text>{{ p.name }}（按 {{ p.qty }} 人）</text><text>¥{{ fen2yuan(p.price * p.qty) }}</text>
        </view>
        <view class="total">
          <text>应付合计</text><text class="t-num">¥{{ fen2yuan(preview.total) }}</text>
        </view>
        <view class="note">金额以服务器实时聚合为准；结账后锁定会话，完成后即可离开。</view>
      </view>

      <!-- 存在待支付结账单（会话被锁）：需先取消再重新发起（3.7/9.8） -->
      <view v-if="locked" class="card warn">
        <text class="w-title">存在待支付结账单</text>
        <text class="w-sub">上次结账未完成，会话已锁定。可取消后重新发起结账。</text>
        <wd-button block type="primary" custom-class="btn" :loading="busy" @click="recreate">取消结账并重新发起</wd-button>
      </view>
      <view v-else class="footer">
        <wd-button block type="primary" size="large" custom-class="btn" :loading="busy" @click="createBill">
          发起结账 ¥{{ fen2yuan(preview.total) }}
        </wd-button>
      </view>
    </template>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'
import { useSessionStore } from '@/stores/session'
import { request, type BizError } from '@/utils/request'
import { fen2yuan } from '@/utils/money'
import type { BillPreviewResp, CreateBillResp } from '@/types/api'

const session = useSessionStore()

const loading = ref(true)
const settled = ref(false)
const locked = ref(false)
const busy = ref(false)
const preview = ref<BillPreviewResp>({
  session_id: 0, pay_mode: '', pax: 0, orders_count: 0,
  dishes_amount: 0, per_head_items: [], per_head_amount: 0, total: 0,
})

const tableNo = ref('—')
const payModeLabel = ref('')

const nothingToPay = ref(false)

async function load() {
  if (!session.sid) return
  loading.value = true
  try {
    const p = await request<BillPreviewResp>({ url: `/sessions/${session.sid}/bill/preview` })
    preview.value = p
    tableNo.value = session.table?.table_no || '—'
    payModeLabel.value = p.pay_mode === 'postpay' ? '后付' : p.pay_mode === 'frontend' ? '前台结' : p.pay_mode
    locked.value = false
    if (p.total === 0 && p.orders_count === 0) {
      nothingToPay.value = true
      settled.value = true
    }
  } catch {
    nothingToPay.value = true
    settled.value = false
  } finally {
    loading.value = false
  }
}

async function createBill() {
  if (busy.value || !session.sid) return
  busy.value = true
  try {
    const resp = await request<CreateBillResp>({ url: `/sessions/${session.sid}/bill`, method: 'POST', data: {} })
    // 收银台按 bill_id 支付 → mock 回调 → 轮询会话 settled（3.5）
    uni.redirectTo({ url: `/pages/pay/index?bill_id=${resp.bill_id}` })
  } catch (e) {
    const biz = e as BizError
    if (biz && biz.__biz && (biz.code === 20004 || biz.code === 20010)) {
      locked.value = true
    }
    // 其他业务错：服务端已 toast
  } finally {
    busy.value = false
  }
}

// 取消待支付结账单 → 解锁 → 重新发起（3.7 超时未付/9.8 取消支付后重新结账）
async function recreate() {
  if (busy.value || !session.sid) return
  busy.value = true
  try {
    await request({ url: `/sessions/${session.sid}/bill/cancel`, method: 'POST', data: {} })
    uni.showToast({ title: '已取消，重新发起结账', icon: 'none' })
    locked.value = false
    await load()
  } catch {
    // 服务端已 toast
  } finally {
    busy.value = false
  }
}

function backMenu() {
  uni.redirectTo({ url: '/pages/menu/index' })
}

onLoad(() => {
  if (!session.sid) {
    uni.showToast({ title: '下单后即可结账', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 600)
    return
  }
  load()
})
onShow(() => {
  // 从收银台返回：重新拉预览（结账后 total=0 → 已结账态）
  if (session.sid) load()
})
</script>

<style lang="scss" scoped>
.co {
  min-height: 100vh;
  background: $bg;
  padding: 24rpx 24rpx calc(140rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
}
.body {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 200rpx;
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
.ok {
  width: 112rpx;
  height: 112rpx;
  border-radius: 50%;
  background: #22c55e;
  color: #fff;
  font-size: 56rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 32rpx;
}
.label {
  color: $sub;
  font-size: 28rpx;
  margin-bottom: 48rpx;
}
.card {
  background: #fff;
  border-radius: 20rpx;
  padding: 28rpx;
}
.c-hd {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20rpx;
}
.c-title {
  font-size: 32rpx;
  font-weight: 700;
  color: $ink;
}
.c-pax {
  font-size: 24rpx;
  color: #9ca3af;
}
.row {
  display: flex;
  justify-content: space-between;
  font-size: 28rpx;
  color: $ink;
  padding: 14rpx 0;
}
.row.sub {
  color: $sub;
  font-size: 26rpx;
}
.total {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 2rpx solid #f5efe9;
  margin-top: 12rpx;
  padding-top: 20rpx;
  font-size: 30rpx;
  color: $ink;
  font-weight: 600;
}
.t-num {
  font-size: 44rpx;
  font-weight: 700;
  color: $brand;
}
.note {
  margin-top: 16rpx;
  font-size: 22rpx;
  color: #9ca3af;
  line-height: 1.6;
}
.card.warn {
  margin-top: 20rpx;
  border: 2rpx solid #fde4d8;
}
.w-title {
  display: block;
  font-size: 30rpx;
  font-weight: 600;
  color: $brand;
}
.w-sub {
  display: block;
  margin: 8rpx 0 24rpx;
  font-size: 26rpx;
  color: $sub;
}
.footer {
  margin-top: 24rpx;
}
.btn {
  border-radius: 999rpx;
}
</style>
