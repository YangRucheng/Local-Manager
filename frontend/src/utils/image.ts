import { MAX_IMAGES, MAX_IMAGE_SIZE } from '@/types'

/**
 * 校验待上传的图片选择是否合法。
 * @param current 当前已选数量
 * @param selected 本次新增的文件
 * @returns 错误信息；为空表示通过
 */
export function validateImageSelection(current: number, selected: File[], max = MAX_IMAGES): string {
  if (!selected.length) return ''
  if (current + selected.length > max) {
    return `图片最多 ${max} 张`
  }
  for (const file of selected) {
    if (file.size > MAX_IMAGE_SIZE) {
      return `「${file.name}」超过 10MB，无法上传`
    }
    if (!file.type.startsWith('image/')) {
      return `「${file.name}」不是图片文件`
    }
  }
  return ''
}

/** 合并图片 id 列表并去重。 */
export function mergeImageIds(...lists: number[][]): number[] {
  const seen = new Set<number>()
  const out: number[] = []
  for (const id of lists.flat()) {
    if (seen.has(id)) continue
    seen.add(id)
    out.push(id)
  }
  return out
}
