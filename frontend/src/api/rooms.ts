import { http } from './client'
import type { Room, RoomInput } from '@/types'

export const roomApi = {
  list: () => http.get<Room[]>('/rooms').then((r) => r.data),
  create: (input: RoomInput) => http.post<Room>('/rooms', input).then((r) => r.data),
  update: (id: number, input: RoomInput) => http.put<Room>(`/rooms/${id}`, input).then((r) => r.data),
  remove: (id: number) => http.delete(`/rooms/${id}`).then((r) => r.data),
}
