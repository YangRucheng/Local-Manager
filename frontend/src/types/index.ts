/** 配电室 */
export interface Room {
  id: number
  name: string
  remark: string
  cabinet_count: number
  image_ids: number[]
  created_at: string
  updated_at: string
}

/** 配电柜 */
export interface Cabinet {
  id: number
  room_id: number
  room_name: string
  name: string
  remark: string
  image_ids: number[]
  created_at: string
  updated_at: string
}

/** 台账记录（总表一行） */
export interface Equipment {
  id: number
  room_id: number
  room_name: string
  cabinet_id: number | null
  cabinet_name: string
  name: string
  model: string
  manufacturer: string
  quantity: number
  remark: string
  image_ids: number[]
  created_at: string
  updated_at: string
}

/** 附件 */
export interface Annex {
  id: number
  uuid: string
  original_name: string
  ext: string
  mime_type: string
  size: number
  ref_count: number
  created_at: string
  updated_at: string
  url?: string
}

/** 分页结果 */
export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

/** 台账列表查询参数 */
export interface EquipmentQuery {
  room_id?: number
  cabinet_id?: number
  keyword?: string
  page?: number
  page_size?: number
}

/** 配电室输入 */
export interface RoomInput {
  name: string
  remark?: string
  image_ids?: number[]
}

/** 配电柜输入 */
export interface CabinetInput {
  room_id: number
  name: string
  remark?: string
  image_ids?: number[]
}

/** 台账输入 */
export interface EquipmentInput {
  room_id: number
  cabinet_id?: number | null
  name: string
  model?: string
  manufacturer?: string
  quantity?: number
  remark?: string
  image_ids?: number[]
}

/** 附件在表单中的本地表示（上传中或已上传） */
export interface UploadedImage {
  annex_id: number
  status: 'uploading' | 'done'
  url?: string
}

/** 单张图片最多 10MB */
export const MAX_IMAGE_SIZE = 10 * 1024 * 1024
/** 每实体最多图片张数 */
export const MAX_IMAGES = 9

/** 附件图片访问 URL */
export function annexFileUrl(id: number): string {
  return `/api/annex/${id}/file`
}
