<template>
  <!-- 通用步进器：qty 为 min 时只显示「+」加号钮（点餐常见形态），否则 [- N +] -->
  <view v-if="modelValue <= min" class="stepper">
    <view
      class="btn add"
      :class="{ disabled }"
      @click.stop="step(1)"
    >
      <text class="plus">＋</text>
    </view>
  </view>
  <view v-else class="stepper">
    <view class="btn" :class="{ disabled }" @click.stop="step(-1)">−</view>
    <view class="num">{{ modelValue }}</view>
    <view class="btn" :class="{ disabled }" @click.stop="step(1)">＋</view>
  </view>
</template>

<script setup lang="ts">
// 自写步进器（点餐缺失组件之一；不引入 input-number，双端行为可控）
const props = defineProps<{
  modelValue: number
  min?: number
  max?: number
  disabled?: boolean
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

const min = props.min ?? 0
const max = props.max ?? 99

function step(d: number) {
  if (props.disabled) return
  const next = props.modelValue + d
  if (next < min || next > max) return
  emit('update:modelValue', next)
}
</script>

<style lang="scss" scoped>
.stepper {
  display: flex;
  align-items: center;
}
.btn {
  width: 44rpx;
  height: 44rpx;
  border-radius: 50%;
  border: 2rpx solid #e5e7eb;
  color: $sub;
  font-size: 28rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  box-sizing: border-box;
}
.btn.add {
  width: 56rpx;
  height: 56rpx;
  border: none;
  background: $brand;
  color: #fff;
  font-size: 32rpx;
}
.btn.disabled {
  opacity: 0.4;
}
.plus {
  font-size: 30rpx;
  line-height: 1;
}
.num {
  min-width: 48rpx;
  text-align: center;
  font-size: 28rpx;
  color: $ink;
  font-weight: 600;
}
</style>
