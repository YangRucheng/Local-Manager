import { http } from './client'
import type { Annex } from '@/types'

export const annexApi = {
  upload: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return http
      .post<Annex>('/annex/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      .then((r) => r.data)
  },
  get: (id: number) => http.get<Annex>(`/annex/${id}`).then((r) => r.data),
  recompute: () => http.post('/annex/recompute').then((r) => r.data),
  cleanup: () => http.post<{ count: number; cleaned: { id: number; original_name: string }[] }>('/annex/cleanup').then((r) => r.data),
}
