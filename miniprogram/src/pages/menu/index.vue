<template>
  <view class="menu">
    <!-- 桌台胶囊（自写）：桌号 + 结账模式，吸顶；右侧订单/呼叫（需先下单取得会话） -->
    <view class="hd">
      <view class="pill">
        <text class="tno">{{ tableNo }}</text>
        <text class="tsep">·</text>
        <text class="tmode">{{ payModeLabel }}</text>
      </view>
      <view class="acts">
        <view class="act" @click="goOrders">
          <text class="act-ico">📋</text><text class="act-txt">订单</text>
        </view>
        <view class="act" @click="callWaiter">
          <text class="act-ico">📞</text><text class="act-txt">呼叫</text>
        </view>
        <view class="act" @click="goProfile">
          <text class="act-ico">👤</text><text class="act-txt">我的</text>
        </view>
      </view>
    </view>

    <!-- 主区：分类侧栏 + 菜品列表 -->
    <view class="bd">
      <scroll-view class="cats" scroll-y>
        <view
          v-for="c in cats"
          :key="c.id"
          class="cat"
          :class="{ active: activeCat === c.id }"
          @click="activeCat = c.id"
        >
          {{ c.name }}
        </view>
      </scroll-view>

      <scroll-view class="dishes" scroll-y>
        <template v-if="dishesOfActive.length">
          <view
            v-for="d in dishesOfActive"
            :key="d.id"
            class="dish"
            :class="{ soldout: d.is_sold_out }"
          >
            <view class="img">{{ d.image_url ? '' : '🍲' }}</view>
            <view class="info">
              <view class="dname">{{ d.name }}</view>
              <view v-if="specsOf(d)" class="dspec">可选规格</view>
              <view class="dprice">
                <text class="cur">¥</text>{{ priceOf(d) }}
              </view>
            </view>
            <view class="op">
              <view v-if="d.is_sold_out" class="soldout-tag">已沽清</view>
              <Stepper
                v-else
                :model-value="qtyOf(d.id)"
                :min="0"
                :max="99"
                @update:model-value="onQty(d, $event)"
              />
            </view>
          </view>
        </template>
        <view v-else class="empty">本类暂无菜品</view>
      </scroll-view>
    </view>

    <!-- 底部购物车条 -->
    <view class="cartbar" :class="{ empty: cart.totalQty === 0 }">
      <view class="sum">
        <view class="qty">已选 {{ cart.totalQty }} 件</view>
        <view class="total" v-if="cart.totalQty > 0">
          <text class="cur">¥</text>{{ fen2yuan(cart.totalPrice) }}
        </view>
        <view class="hint" v-else>选好菜品后下单</view>
      </view>
      <view class="go" :class="{ disabled: cart.totalQty === 0 }" @click="goOrder">去结算</view>
    </view>

    <!-- 规格弹窗 -->
    <SpecDialog v-model="showSpec" :dish="specDish" @confirm="onSpecConfirm" />
  </view>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onLoad, onPullDownRefresh } from '@dcloudio/uni-app'
import { useSessionStore } from '@/stores/session'
import { useCartStore } from '@/stores/cart'
import { request } from '@/utils/request'
import { parseSpecGroups } from '@/utils/specs'
import { fen2yuan } from '@/utils/money'
import type { DishDTO } from '@/types/api'
import Stepper from '@/components/stepper.vue'
import SpecDialog from '@/components/spec-dialog.vue'

const session = useSessionStore()
const cart = useCartStore()

const activeCat = ref(0)
const showSpec = ref(false)
const specDish = ref<DishDTO | null>(null)
const loading = ref(true)
const error = ref('')

const payModeLabel = computed(() => {
  const m = session.table?.pay_mode
  if (m === 'prepay') return '先付'
  if (m === 'postpay') return '后付'
  if (m === 'frontend') return '前台结'
  return '—'
})
const tableNo = computed(() => session.table?.table_no || '—')

// 分类侧栏：「全部」+ 后端分类
const cats = computed(() => {
  const menu = session.menu
  if (!menu) return []
  const all = [{ id: 0, name: '全部', dishes: menu.categories.flatMap((c) => c.dishes) }]
  return all.concat(menu.categories.map((c) => ({ id: c.id, name: c.name, dishes: c.dishes })))
})

const dishesOfActive = computed(() => {
  const cat = cats.value.find((c) => c.id === activeCat.value)
  return cat ? cat.dishes : []
})

function specsOf(d: DishDTO) {
  return parseSpecGroups(d.specs)
}
function priceOf(d: DishDTO) {
  return fen2yuan(d.price)
}
function qtyOf(dishId: number) {
  return cart.countByDish(dishId)
}

// 数量步进：v>旧值 = 加购（有规格先弹窗）；v<旧值 = 减购
function onQty(d: DishDTO, v: number) {
  const old = qtyOf(d.id)
  if (v > old) {
    if (d.is_sold_out) return
    const groups = parseSpecGroups(d.specs)
    if (groups) {
      specDish.value = d
      showSpec.value = true
      return
    }
    cart.addDish(d, 1)
  } else {
    cart.decDish(d.id)
  }
}
function onSpecConfirm(selection: Record<string, string>, label: string) {
  if (specDish.value) cart.addDish(specDish.value, 1, selection, label)
}

function goOrder() {
  if (cart.totalQty === 0) return
  uni.navigateTo({ url: '/pages/order/index' })
}

// 3.5：本桌会话订单（拼桌互见）；需先下单取得 sid
function goOrders() {
  if (!session.sid) {
    uni.showToast({ title: '下单后即可查看本桌订单', icon: 'none' })
    return
  }
  uni.navigateTo({ url: '/pages/orders/index' })
}

// 3.5：我的（手机号授权可选，9.4）
function goProfile() {
  uni.navigateTo({ url: '/pages/profile/index' })
}

// 3.5：呼叫服务员（会话内）
async function callWaiter() {
  if (!session.sid) {
    uni.showToast({ title: '下单后即可呼叫服务员', icon: 'none' })
    return
  }
  try {
    await request({ url: `/sessions/${session.sid}/call-waiter`, method: 'POST', data: {} })
    uni.showToast({ title: '已呼叫，服务员马上到', icon: 'success' })
  } catch {
    // 服务端已统一 toast
  }
}

onLoad(async () => {
  // 直达路径（非扫码）也能用：有 tid/sig 则补拉菜单；都没有 → 错误屏
  if (!session.menu && session.tid && session.sig) {
    try {
      await session.fetchMenu()
    } catch (e) {
      error.value = (e as { msg?: string })?.msg || '菜单加载失败'
    }
  }
  loading.value = false
  if (!session.tid) {
    error.value = '请从桌贴二维码进入'
  }
})

// 3.5 沽清同步：下拉重拉菜单（下架/沽清即时生效到两端，9.6 服务端仍为最终防线）
onPullDownRefresh(async () => {
  if (session.tid && session.sig) {
    try {
      await session.fetchMenu()
    } catch {
      // 保持旧菜单，仅提示
      uni.showToast({ title: '菜单刷新失败，请重试', icon: 'none' })
    }
  }
  uni.stopPullDownRefresh()
})
</script>

<style lang="scss" scoped>
.menu {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: $bg;
}
.hd {
  padding: 20rpx 24rpx 16rpx;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.acts {
  display: flex;
}
.act {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-left: 28rpx;
  padding: 6rpx 4rpx;
}
.act-ico {
  font-size: 34rpx;
  line-height: 1;
}
.act-txt {
  margin-top: 6rpx;
  font-size: 20rpx;
  color: $sub;
}
.pill {
  display: flex;
  align-items: center;
  padding: 10rpx 28rpx;
  border-radius: 999rpx;
  background: #fff;
  border: 2rpx solid #f0e6de;
  box-shadow: 0 4rpx 12rpx rgba(31, 35, 41, 0.04);
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
  padding: 4rpx 16rpx;
  border-radius: 999rpx;
}
.bd {
  flex: 1;
  display: flex;
  min-height: 0;
}
.cats {
  width: 176rpx;
  background: #fff;
  height: 100%;
}
.cat {
  padding: 28rpx 16rpx;
  font-size: 26rpx;
  color: $sub;
  text-align: center;
  border-left: 6rpx solid transparent;
}
.cat.active {
  color: $brand;
  font-weight: 600;
  background: $bg;
  border-left-color: $brand;
}
.dishes {
  flex: 1;
  height: 100%;
  padding: 0 24rpx;
  box-sizing: border-box;
}
.dish {
  display: flex;
  align-items: center;
  padding: 24rpx 0;
  border-bottom: 2rpx solid #f3ece6;
  opacity: 1;
}
.dish.soldout {
  opacity: 0.45;
}
.img {
  width: 128rpx;
  height: 128rpx;
  border-radius: 16rpx;
  background: #f3ece6;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48rpx;
  flex-shrink: 0;
}
.info {
  flex: 1;
  margin-left: 20rpx;
  min-width: 0;
}
.dname {
  font-size: 30rpx;
  color: $ink;
  font-weight: 500;
}
.dspec {
  margin-top: 6rpx;
  font-size: 22rpx;
  color: $sub;
  background: $bg;
  display: inline-flex;
  padding: 2rpx 14rpx;
  border-radius: 999rpx;
}
.dprice {
  margin-top: 10rpx;
  font-size: 32rpx;
  font-weight: 700;
  color: $brand;
}
.cur {
  font-size: 24rpx;
  margin-right: 2rpx;
}
.op {
  flex-shrink: 0;
  margin-left: 16rpx;
}
.soldout-tag {
  font-size: 22rpx;
  color: #9ca3af;
  padding: 8rpx 16rpx;
  border: 2rpx solid #e5e7eb;
  border-radius: 999rpx;
}
.empty {
  padding: 80rpx 0;
  text-align: center;
  color: #9ca3af;
  font-size: 26rpx;
}
.cartbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 16rpx 24rpx calc(16rpx + env(safe-area-inset-bottom));
  padding: 16rpx 20rpx 16rpx 28rpx;
  background: $ink;
  border-radius: 999rpx;
}
.cartbar.empty {
  background: #2b2f36;
}
.sum {
  display: flex;
  align-items: baseline;
}
.qty {
  color: #c8ccd4;
  font-size: 26rpx;
  margin-right: 20rpx;
}
.total {
  color: #fff;
  font-size: 36rpx;
  font-weight: 700;
}
.hint {
  color: #9aa0aa;
  font-size: 24rpx;
}
.go {
  background: $brand;
  color: #fff;
  font-size: 30rpx;
  font-weight: 600;
  padding: 16rpx 44rpx;
  border-radius: 999rpx;
}
.go.disabled {
  background: #5b6068;
  color: #cfd3d9;
}
</style>
