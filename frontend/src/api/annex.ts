import { http } from './client'
import type { Annex, AnnexQuery, PageResult } from '@/types'

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
  list: (query: AnnexQuery = {}) =>
    http
      .get<PageResult<Annex>>('/annex', {
        params: {
          keyword: query.keyword || undefined,
          page: query.page || 1,
          page_size: query.page_size || 20,
        },
      })
      .then((r) => r.data),
  recompute: () => http.post('/annex/recompute').then((r) => r.data),
}
