<script setup lang="ts">
import { onLaunch, onShow, onHide } from '@dcloudio/uni-app'
onLaunch(() => {
  console.log('[yodian] app launch')
})
onShow(() => {})
onHide(() => {})
</script>

<style lang="scss">
/* Wot Design Uni 主题覆盖：主色改暖橙（9.2 餐饮基调，与商家端蓝区分） */
page {
  --wot-color-theme: #e34d2b;
  background-color: #faf6f2;
  color: #1f2329;
}
</style>
