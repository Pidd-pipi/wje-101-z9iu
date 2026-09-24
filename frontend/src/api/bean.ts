import request from '@/utils/request'
import type { BeanListItem, CoffeeBean, UserBeanGroups } from '@/constants/bean'
import type { PageData } from '@/types/api'

export function listBeans(params: { page?: number; page_size?: number; origin?: string; process?: string; keyword?: string }) {
  return request.get<never, PageData<BeanListItem>>('/beans', { params })
}
export function createBean(payload: Partial<CoffeeBean>) { return request.post<never, CoffeeBean>('/beans', payload) }
export function updateBean(id: number, payload: Partial<CoffeeBean>) { return request.put<never, CoffeeBean>(`/beans/${id}`, payload) }
export function deleteBean(id: number) { return request.delete<never, { deleted: boolean }>(`/beans/${id}`) }

// Put a bean into the current user's want-to-drink list (one entry per bean).
export function addBeanToList(id: number) { return request.post<never, { in_list: boolean }>(`/beans/${id}/list`) }
// Move a want-to-drink bean back out of the list.
export function removeBeanFromList(id: number) { return request.delete<never, { in_list: boolean }>(`/beans/${id}/list`) }

// Fetch a user's want-to-drink / already-drunk bean groups for the profile.
export function getUserBeans(userId: number | string) { return request.get<never, UserBeanGroups>(`/users/${userId}/beans`) }
