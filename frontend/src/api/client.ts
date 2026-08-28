import axios from 'axios'
import type { AxiosError } from 'axios'

/** 全局 axios 实例：baseURL /api，统一错误消息。 */
export const http = axios.create({ baseURL: '/api', timeout: 30000 })

export interface ApiError {
  error?: string
}

/** 从 axios 错误中提取可读消息。 */
export function errorMessage(err: unknown): string {
  if (axios.isAxiosError(err)) {
    const data = (err as AxiosError<ApiError>).response?.data
    if (data?.error) return data.error
    if (err.code === 'ECONNABORTED') return '请求超时'
    if (!err.response) return '无法连接服务器，请确认后端已启动'
    return `请求失败（${err.response.status}）`
  }
  return err instanceof Error ? err.message : '发生未知错误'
}
