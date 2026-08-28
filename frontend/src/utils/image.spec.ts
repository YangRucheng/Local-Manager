import { describe, expect, it } from 'vitest'
import { mergeImageIds, validateImageSelection } from './image'

function file(name: string, size: number, type = 'image/png'): File {
  return new File([new Uint8Array(size)], name, { type })
}

describe('validateImageSelection', () => {
  it('空选择直接通过', () => {
    expect(validateImageSelection(0, [])).toBe('')
  })

  it('超过 9 张时报错', () => {
    const files = [file('a.png', 10), file('b.png', 10)]
    expect(validateImageSelection(8, files)).toContain('最多 9 张')
  })

  it('恰好在 9 张内通过', () => {
    const files = [file('a.png', 10), file('b.png', 10)]
    expect(validateImageSelection(7, files)).toBe('')
  })

  it('超过 10MB 时报错', () => {
    expect(validateImageSelection(0, [file('big.png', 10 * 1024 * 1024 + 1)])).toContain('10MB')
  })

  it('非图片文件报错', () => {
    expect(validateImageSelection(0, [file('x.txt', 10, 'text/plain')])).toContain('不是图片')
  })
})

describe('mergeImageIds', () => {
  it('合并并去重', () => {
    expect(mergeImageIds([1, 2, 3], [2, 4], [1])).toEqual([1, 2, 3, 4])
  })
})
