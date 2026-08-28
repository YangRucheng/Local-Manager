import { http } from './client'
import type { Cabinet, CabinetInput } from '@/types'

export const cabinetApi = {
  list: (roomId?: number) =>
    http
      .get<Cabinet[]>('/cabinets', { params: roomId ? { room_id: roomId } : undefined })
      .then((r) => r.data),
  create: (input: CabinetInput) => http.post<Cabinet>('/cabinets', input).then((r) => r.data),
  update: (id: number, input: CabinetInput) =>
    http.put<Cabinet>(`/cabinets/${id}`, input).then((r) => r.data),
  remove: (id: number) => http.delete(`/cabinets/${id}`).then((r) => r.data),
}
