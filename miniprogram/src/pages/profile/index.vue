<template>
  <view class="pf">
    <!-- 手机号授权（9.4 可选）：拒绝不影响点餐。绑定后用于订单联系/优惠，真实短信供应商阶段 4 -->
    <view class="card">
      <view class="hd">手机号绑定</view>
      <view v-if="boundPhone" class="bound">
        <view class="ok">✓</view>
        <view class="p">{{ maskPhone(boundPhone) }}</view>
        <text class="sub">已绑定，可联系您取餐/通知</text>
      </view>
      <template v-else>
        <view class="field">
          <input
            class="inp"
            type="number"
            :maxlength="11"
            v-model="phone"
            placeholder="请输入手机号"
            placeholder-class="ph"
          />
          <view class="send" :class="{ disabled: sending || phone.length !== 11 }" @click="sendCode">
            {{ sending ? '发送中…' : '发送验证码' }}
          </view>
        </view>
        <view class="field">
          <input
            class="inp"
            type="number"
            :maxlength="6"
            v-model="code"
            placeholder="请输入验证码"
            placeholder-class="ph"
          />
        </view>
        <view class="dev" v-if="devCode">开发环境验证码：{{ devCode }}</view>
        <wd-button block type="primary" custom-class="btn" :loading="binding" :disabled="code.length < 6" @click="bind">
          确认绑定
        </wd-button>
        <text class="hint">绑定手机号仅作通知联系用，与登录解耦，拒绝仍可正常点餐。</text>
      </template>
    </view>

    <view class="card about">
      <view class="row"><text>当前渠道</text><text>{{ auth.channel === 'wechat' ? '微信' : auth.channel === 'alipay' ? '支付宝' : auth.channel }}</text></view>
      <view class="row"><text>顾客编号</text><text>{{ auth.customerId ?? '—' }}</text></view>
    </view>
  </view>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { request } from '@/utils/request'
import { useAuthStore } from '@/stores/auth'
import type { PhoneCodeResp, PhoneBindResp } from '@/types/api'

const auth = useAuthStore()

const phone = ref('')
const code = ref('')
const sending = ref(false)
const binding = ref(false)
const devCode = ref('')
const boundPhone = ref('')

function maskPhone(p: string) {
  return p.replace(/^(\d{3})\d{4}(\d{4})$/, '$1****$2')
}

async function sendCode() {
  if (sending.value || !/^1[3-9]\d{9}$/.test(phone.value)) {
    if (!/^1[3-9]\d{9}$/.test(phone.value)) uni.showToast({ title: '手机号格式不正确', icon: 'none' })
    return
  }
  sending.value = true
  try {
    const resp = await request<PhoneCodeResp>({ url: '/auth/phone-code', method: 'POST', data: { phone: phone.value } })
    if (resp.dev_code) {
      devCode.value = resp.dev_code
      uni.showToast({ title: '验证码已发送（开发固定码）', icon: 'none' })
    }
  } catch {
    // 服务端已 toast
  } finally {
    sending.value = false
  }
}

async function bind() {
  if (binding.value || code.value.length < 6) return
  binding.value = true
  try {
    const resp = await request<PhoneBindResp>({ url: '/auth/phone-bind', method: 'POST', data: { phone: phone.value, code: code.value } })
    boundPhone.value = resp.phone
    uni.showToast({ title: '绑定成功', icon: 'success' })
  } catch {
    // 服务端已 toast
  } finally {
    binding.value = false
  }
}
</script>

<style lang="scss" scoped>
.pf {
  min-height: 100vh;
  background: $bg;
  padding: 24rpx;
  box-sizing: border-box;
}
.card {
  background: #fff;
  border-radius: 20rpx;
  padding: 28rpx;
  margin-bottom: 20rpx;
}
.hd {
  font-size: 32rpx;
  font-weight: 700;
  color: $ink;
  margin-bottom: 24rpx;
}
.field {
  display: flex;
  align-items: center;
  border: 2rpx solid #efe7e0;
  border-radius: 16rpx;
  padding: 8rpx 20rpx;
  margin-bottom: 20rpx;
}
.inp {
  flex: 1;
  height: 72rpx;
  font-size: 30rpx;
}
.ph {
  color: #c3c7cd;
}
.send {
  flex-shrink: 0;
  font-size: 26rpx;
  color: $brand;
  padding: 12rpx 8rpx 12rpx 20rpx;
  border-left: 2rpx solid #f0e6de;
}
.send.disabled {
  color: #c3c7cd;
}
.dev {
  font-size: 24rpx;
  color: $brand;
  background: $brand-light;
  padding: 12rpx 20rpx;
  border-radius: 12rpx;
  margin-bottom: 20rpx;
}
.btn {
  border-radius: 999rpx;
}
.hint {
  display: block;
  margin-top: 20rpx;
  font-size: 22rpx;
  color: #9ca3af;
  line-height: 1.6;
}
.bound {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24rpx 0;
}
.ok {
  width: 88rpx;
  height: 88rpx;
  border-radius: 50%;
  background: #22c55e;
  color: #fff;
  font-size: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16rpx;
}
.p {
  font-size: 34rpx;
  font-weight: 600;
  color: $ink;
}
.sub {
  margin-top: 8rpx;
  font-size: 24rpx;
  color: #9ca3af;
}
.row {
  display: flex;
  justify-content: space-between;
  font-size: 28rpx;
  color: $ink;
  padding: 12rpx 0;
}
</style>
