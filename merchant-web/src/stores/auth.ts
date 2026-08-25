import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { http } from '@/api/http'
import type { Operator, Role } from '@/types'

const TOKEN_KEY = 'yodian.token'
const USER_KEY = 'yodian.user'

/**
 * 阶段 4.1 联调：真实鉴权 POST /auth/staff/login {employee_no, password}
 * → {token, id, name, employee_no, role}。角色以服务端返回为准，越权由后端 40003 拦截。
 */
export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) || '')
  const operator = ref<Operator | null>(
    JSON.parse(localStorage.getItem(USER_KEY) || 'null'),
  )

  const isLoggedIn = computed(() => !!token.value && !!operator.value)
  const role = computed<Role | ''>(() => operator.value?.role || '')

  async function realLogin(employeeNo: string, password: string): Promise<Operator> {
    const data = (await http.post('/auth/staff/login', {
      employee_no: employeeNo,
      password,
    })) as { token: string; id: number; name: string; employee_no: string; role: Role }
    const user: Operator = {
      id: data.id,
      name: data.name,
      role: data.role,
      employeeNo: data.employee_no,
    }
    token.value = data.token
    operator.value = user
    localStorage.setItem(TOKEN_KEY, token.value)
    localStorage.setItem(USER_KEY, JSON.stringify(user))
    return user
  }

  function logout() {
    token.value = ''
    operator.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  return { token, operator, isLoggedIn, role, realLogin, logout }
})
