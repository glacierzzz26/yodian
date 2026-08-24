import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Operator, Role } from '@/types'

const TOKEN_KEY = 'yodian.token'
const USER_KEY = 'yodian.user'

const ROLE_LABEL: Record<Role, string> = {
  owner: '店长',
  cashier: '收银员',
  kitchen: '后厨',
}

/**
 * 模拟登录：契约对齐 /auth/login（设计方案 16.1），后端就绪后把 mockLogin 换成 http。
 * 真实实现：POST /auth/login {employee_no, password} → {token, operator:{id,name,role,employee_no}}
 */
export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) || '')
  const operator = ref<Operator | null>(
    JSON.parse(localStorage.getItem(USER_KEY) || 'null'),
  )

  const isLoggedIn = computed(() => !!token.value && !!operator.value)
  const role = computed<Role | ''>(() => operator.value?.role || '')

  // 模拟登录（开发期任意工号 + 123456，角色由登录页选择决定可见菜单）
  function mockLogin(employeeNo: string, password: string, rolePick: Role): Promise<Operator> {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        if (password !== '123456') {
          reject(new Error('工号或密码错误'))
          return
        }
        const user: Operator = {
          id: Number(employeeNo) || 1001,
          name: `${ROLE_LABEL[rolePick]}${employeeNo}`,
          role: rolePick,
          employeeNo,
        }
        token.value = `mock-token-${employeeNo}`
        operator.value = user
        localStorage.setItem(TOKEN_KEY, token.value)
        localStorage.setItem(USER_KEY, JSON.stringify(user))
        resolve(user)
      }, 300)
    })
  }

  function logout() {
    token.value = ''
    operator.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  return { token, operator, isLoggedIn, role, mockLogin, logout }
})
