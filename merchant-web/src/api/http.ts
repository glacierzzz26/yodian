import axios from 'axios'
import type { ApiResp } from '@/types'

/**
 * 统一响应包络（设计方案 16.1）：{code, msg, data}
 * 后端就绪后各 module 内部改用 http 请求，契约不变。
 */
export const http = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('yodian.token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (resp) => {
    const body = resp.data as ApiResp<unknown>
    if (body.code !== 0) return Promise.reject(new Error(body.msg))
    return body.data as never
  },
  (err) => Promise.reject(err),
)
