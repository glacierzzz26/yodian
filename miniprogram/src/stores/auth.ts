// 登录态（Pinia）：token / customer_id / channel。token 持久化到本地（utils/token.ts）。
// 手机号授权独立于登录，拒绝不影响点餐（9.4）。
import { defineStore } from 'pinia'
import { silentLogin } from '@/utils/auth'
import { getToken, setToken, clearToken } from '@/utils/token'

interface AuthState {
  token: string
  customerId: number | null
  channel: string
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: getToken(),
    customerId: null,
    channel: '',
  }),
  actions: {
    async login() {
      const resp = await silentLogin()
      this.token = resp.token
      this.customerId = resp.customer_id
      this.channel = resp.channel
      setToken(resp.token)
    },
    logout() {
      this.token = ''
      this.customerId = null
      this.channel = ''
      clearToken()
    },
  },
})
