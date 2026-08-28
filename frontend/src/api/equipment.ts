import { http } from './client'
import type { Equipment, EquipmentInput, EquipmentQuery, PageResult } from '@/types'

export const equipmentApi = {
  list: (query: EquipmentQuery) =>
    http
      .get<PageResult<Equipment>>('/equipment', {
        params: {
          room_id: query.room_id || undefined,
          cabinet_id: query.cabinet_id || undefined,
          keyword: query.keyword || undefined,
          page: query.page || 1,
          page_size: query.page_size || 20,
        },
      })
      .then((r) => r.data),
  get: (id: number) => http.get<Equipment>(`/equipment/${id}`).then((r) => r.data),
  create: (input: EquipmentInput) => http.post<Equipment>('/equipment', input).then((r) => r.data),
  update: (id: number, input: EquipmentInput) =>
    http.put<Equipment>(`/equipment/${id}`, input).then((r) => r.data),
  remove: (id: number) => http.delete(`/equipment/${id}`).then((r) => r.data),
}
