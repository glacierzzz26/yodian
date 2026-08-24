import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const isDark = ref(false)
  const siderCollapsed = ref(false)
  const degrade = ref(false) // 降级模式（设计方案 5.1），仅店长可开关

  function toggleTheme() {
    isDark.value = !isDark.value
    document.documentElement.dataset.theme = isDark.value ? 'dark' : 'light'
  }
  function toggleSider() {
    siderCollapsed.value = !siderCollapsed.value
  }

  return { isDark, siderCollapsed, degrade, toggleTheme, toggleSider }
})
