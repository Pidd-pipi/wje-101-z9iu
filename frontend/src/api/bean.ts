import request from '@/utils/request'
import type { CoffeeBean } from '@/constants/bean'
import type { PageData } from '@/types/api'

export function listBeans(params: { page?: number; page_size?: number; origin?: string; process?: string; keyword?: string }) {
  return request.get<never, PageData<CoffeeBean>>('/beans', { params })
}
export function createBean(payload: Partial<CoffeeBean>) { return request.post<never, CoffeeBean>('/beans', payload) }
export function updateBean(id: number, payload: Partial<CoffeeBean>) { return request.put<never, CoffeeBean>(`/beans/${id}`, payload) }
export function deleteBean(id: number) { return request.delete<never, { deleted: boolean }>(`/beans/${id}`) }

// 待喝名单：登录用户把豆种加入名单（每豆一条）。
export function addBeanToWant(id: number) {
  return request.post<never, { id: number; bean_id: number; track_status: 'want' }>(`/beans/${id}/want`)
}
// 待喝名单：把仍处于待喝（笔记数为 0）的豆移出名单。
export function removeBeanFromWant(id: number) {
  return request.delete<never, { removed: boolean }>(`/beans/${id}/want`)
}

