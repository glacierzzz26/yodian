<template>
  <view class="order">
    <!-- 人数选择（首单开台定数；首单后锁定，改人数走 PATCH /sessions/:sid/pax） -->
    <view class="card">
      <PaxSelector :model-value="pax" :disabled="session.hasOrders" @update:model-value="setPax" />
      <view v-if="session.hasOrders" class="lock">首单人数已锁定，需调整请通过「按人数变更」或呼叫服务员</view>
      <view v-else-if="isPrepay && perHead.length" class="ph">
        按人计费：{{ perHeadNames }}（¥{{ fen2yuan(perHeadUnit) }}/人 × {{ pax }} 人），首单自动计入
      </view>
    </view>

    <!-- 已选菜品 -->
    <view class="card">
      <view class="l-hd">已选菜品</view>
      <view v-for="(it, idx) in cart.items" :key="idx" class="line">
        <view class="l-info">
          <view class="l-name">{{ it.name }}</view>
          <view v-if="it.specLabel" class="l-spec">{{ it.specLabel }}</view>
          <view class="l-price">¥{{ fen2yuan(it.price) }}/份</view>
        </view>
        <view class="l-op">
          <Stepper :model-value="it.qty" :min="1" :max="99" @update:model-value="setQty(idx, $event)" />
          <view class="del" @click="cart.removeAt(idx)">删除</view>
        </view>
      </view>
      <view v-if="cart.items.length === 0" class="l-empty">购物车为空</view>
    </view>

    <!-- 合计预览（实收以服务端订单总额为准） -->
    <view class="card">
      <view class="t-row"><text>合计（预览）</text><text class="t-num">¥{{ fen2yuan(cart.totalPrice) }}</text></view>
      <view class="t-note">实收金额以下单成功后的订单总额为准</view>
    </view>

    <view class="footer">
      <wd-button
        block
        type="primary"
        :loading="submitting"
        :disabled="cart.items.length === 0"
        custom-class="submit"
        @click="submit"
      >
        提交订单
      </wd-button>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { useSessionStore } from '@/stores/session'
import { useCartStore } from '@/stores/cart'
import { request } from '@/utils/request'
import { fen2yuan } from '@/utils/money'
import type { CreateOrderResp, DishDTO } from '@/types/api'
import PaxSelector from '@/components/pax-selector.vue'
import Stepper from '@/components/stepper.vue'

const session = useSessionStore()
const cart = useCartStore()

const pax = ref(2)
const submitting = ref(false)

const perHead = computed(() => session.menu?.per_head ?? [])
const isPrepay = computed(() => session.table?.pay_mode === 'prepay')
const perHeadUnit = computed(() => perHead.value.reduce((n, p) => n + p.price, 0))
const perHeadNames = computed(() => perHead.value.map((p) => p.name).join('、'))

function setPax(v: number) {
  pax.value = v
}
function setQty(idx: number, v: number) {
  cart.items[idx].qty = v
}

onLoad(() => {
  if (cart.items.length === 0) {
    uni.showToast({ title: '购物车为空', icon: 'none' })
    setTimeout(() => uni.navigateBack(), 600)
    return
  }
  pax.value = session.table?.seats || 2
  if (pax.value < 1) pax.value = 1
  if (pax.value > 20) pax.value = 20
})

async function submit() {
  if (submitting.value || cart.items.length === 0) return
  submitting.value = true
  try {
    const items = cart.items.map((i) => ({
      dish_id: i.dish_id,
      qty: i.qty,
      ...(i.specs ? { specs: i.specs } : {}),
    }))
    const resp = await request<CreateOrderResp>({
      url: '/orders',
      method: 'POST',
      data: { tid: session.tid, pax: pax.value, items },
    })
    cart.clear()
    session.markOrdered(resp.session_id)
    if (resp.pay_mode === 'prepay') {
      uni.redirectTo({ url: `/pages/pay/index?order_id=${resp.order_id}` })
    } else {
      uni.showToast({ title: '下单成功', icon: 'success' })
      setTimeout(() => uni.redirectTo({ url: '/pages/menu/index' }), 800)
    }
  } catch {
    // 服务端已统一 toast（信封拦截）；沽清/越权等业务错提示已出，停留本页允许调整
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="scss" scoped>
.order {
  min-height: 100vh;
  background: $bg;
  padding: 24rpx 24rpx calc(160rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
}
.card {
  background: #fff;
  border-radius: 20rpx;
  padding: 28rpx;
  margin-bottom: 24rpx;
}
.lock {
  margin-top: 16rpx;
  font-size: 24rpx;
  color: #9ca3af;
}
.ph {
  margin-top: 16rpx;
  font-size: 24rpx;
  color: $sub;
  background: $brand-light;
  padding: 14rpx 20rpx;
  border-radius: 12rpx;
  color: $brand;
}
.l-hd {
  font-size: 30rpx;
  font-weight: 600;
  color: $ink;
  margin-bottom: 8rpx;
}
.line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 0;
  border-bottom: 2rpx solid #f5efe9;
}
.l-info {
  min-width: 0;
}
.l-name {
  font-size: 30rpx;
  color: $ink;
}
.l-spec {
  font-size: 24rpx;
  color: $sub;
  margin-top: 4rpx;
}
.l-price {
  font-size: 26rpx;
  color: $sub;
  margin-top: 8rpx;
}
.l-op {
  display: flex;
  align-items: center;
}
.del {
  margin-left: 24rpx;
  font-size: 24rpx;
  color: #9ca3af;
  padding: 8rpx;
}
.l-empty {
  padding: 40rpx 0;
  text-align: center;
  color: #9ca3af;
  font-size: 26rpx;
}
.t-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 30rpx;
  color: $ink;
}
.t-num {
  font-size: 40rpx;
  font-weight: 700;
  color: $brand;
}
.t-note {
  margin-top: 12rpx;
  font-size: 22rpx;
  color: #9ca3af;
}
.footer {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 16rpx 24rpx calc(16rpx + env(safe-area-inset-bottom));
  background: rgba(250, 246, 242, 0.96);
}
.submit {
  border-radius: 999rpx;
}
</style>
