<template>
  <!-- 规格弹窗（自写，9.2 点餐缺失组件）：每组单选，默认第一项；确定后返回 {组名: 选项名} -->
  <wd-popup v-model="show" position="bottom" custom-class="spec-popup">
    <view class="spec" v-if="groups && groups.length">
      <view class="title">{{ dish?.name }}</view>
      <view v-for="g in groups" :key="g.name" class="group">
        <view class="gname">{{ g.name }}</view>
        <view class="opts">
          <view
            v-for="o in g.options"
            :key="o.name"
            class="opt"
            :class="{ active: selection[g.name] === o.name }"
            @click="select(g.name, o.name)"
          >
            {{ o.name }}
          </view>
        </view>
      </view>
      <wd-button block type="primary" custom-class="confirm" @click="confirm">确定</wd-button>
    </view>
  </wd-popup>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { DishDTO } from '@/types/api'
import { parseSpecGroups, defaultSelection, type SpecGroup } from '@/utils/specs'

const props = defineProps<{
  modelValue: boolean
  dish: DishDTO | null
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'confirm', selection: Record<string, string>, label: string): void
}>()

const show = ref(props.modelValue)
const groups = ref<SpecGroup[] | null>(null)
const selection = ref<Record<string, string>>({})

watch(
  () => props.modelValue,
  (v) => {
    show.value = v
    if (v && props.dish) {
      groups.value = parseSpecGroups(props.dish.specs)
      selection.value = groups.value ? defaultSelection(groups.value) : {}
    }
  },
)

function select(group: string, opt: string) {
  selection.value = { ...selection.value, [group]: opt }
}
function confirm() {
  if (!groups.value) return
  const label = Object.keys(selection.value)
    .map((k) => selection.value[k])
    .join(' · ')
  emit('confirm', selection.value, label)
  show.value = false
}
</script>

<style lang="scss" scoped>
.spec {
  padding: 40rpx 32rpx calc(40rpx + env(safe-area-inset-bottom));
  background: #fff;
  border-radius: 24rpx 24rpx 0 0;
}
.title {
  font-size: 34rpx;
  font-weight: 600;
  color: $ink;
  margin-bottom: 8rpx;
}
.group {
  margin-top: 28rpx;
}
.gname {
  font-size: 26rpx;
  color: $sub;
  margin-bottom: 16rpx;
}
.opts {
  display: flex;
  flex-wrap: wrap;
}
.opt {
  padding: 12rpx 32rpx;
  border-radius: 999rpx;
  background: $bg;
  border: 2rpx solid transparent;
  color: $ink;
  font-size: 28rpx;
  margin: 0 16rpx 16rpx 0;
}
.opt.active {
  background: $brand-light;
  color: $brand;
  border-color: $brand;
}
.confirm {
  margin-top: 24rpx;
}
</style>
