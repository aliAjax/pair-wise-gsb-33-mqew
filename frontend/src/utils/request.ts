import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/authStore'

declare module 'axios' {
  // silent: 本次请求失败时不弹出全局错误提示（用于自动重试等场景）
  export interface AxiosRequestConfig {
    silent?: boolean
  }
}

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

request.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

request.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && typeof body.code === 'number' && body.code !== 0) {
      if (!res.config.silent) {
        ElMessage.error(body.message || '请求失败')
      }
      return Promise.reject(new Error(body.message))
    }
    return body?.data
  },
  (err) => {
    const status = err.response?.status
    const msg = err.response?.data?.message || '网络异常'
    if (status === 401) {
      const auth = useAuthStore()
      auth.logout()
    }
    if (!err.config?.silent) {
      ElMessage.error(msg)
    }
    return Promise.reject(err)
  },
)

export default request
