<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import { ConfigProvider } from 'ant-design-vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { theme as antdTheme } from 'ant-design-vue'

const appStore = useAppStore()
const providerTheme = computed(() => ({
  algorithm: appStore.isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
  token: {
    colorPrimary: '#1677FF',
    borderRadius: 6,
  },
}))
</script>

<template>
  <ConfigProvider :locale="zhCN" :theme="providerTheme">
    <router-view />
  </ConfigProvider>
</template>
