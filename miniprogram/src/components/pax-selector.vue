<template>
  <!-- 就餐人数选择（自写）：1–20 步进（服务端兜底默认 2；唯一入口 PATCH /sessions/:sid/pax，下单 pax 仅开台首设） -->
  <view class="pax">
    <view class="label">就餐人数</view>
    <view class="stepper">
      <view class="btn" :class="{ disabled: modelValue <= min || disabled }" @click="step(-1)">−</view>
      <view class="num"><text class="n">{{ modelValue }}</text> 人</view>
      <view class="btn" :class="{ disabled: modelValue >= max || disabled }" @click="step(1)">＋</view>
    </view>
  </view>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: number
  min?: number
  max?: number
  disabled?: boolean
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

const min = props.min ?? 1
const max = props.max ?? 20

function step(d: number) {
  if (props.disabled) return
  const next = props.modelValue + d
  if (next < min || next > max) return
  emit('update:modelValue', next)
}
</script>

<style lang="scss" scoped>
.pax {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.label {
  font-size: 30rpx;
  color: $ink;
  font-weight: 600;
}
.stepper {
  display: flex;
  align-items: center;
}
.btn {
  width: 64rpx;
  height: 64rpx;
  border-radius: 50%;
  border: 2rpx solid #e5e7eb;
  color: $sub;
  font-size: 32rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
}
.btn.disabled {
  opacity: 0.4;
}
.num {
  min-width: 120rpx;
  text-align: center;
  font-size: 28rpx;
  color: $ink;
}
.n {
  font-size: 40rpx;
  font-weight: 700;
  color: $brand;
}
</style>
