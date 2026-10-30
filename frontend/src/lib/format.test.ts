import { describe, expect, it } from 'vitest'
import { formatBytes, percentOf, sortNodes } from './format'
import type { SortState, TreeNode } from './types'

describe('formatBytes', () => {
  it('formats bytes into human readable units', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1024 * 1024)).toBe('1.0 MB')
  })
})

describe('percentOf', () => {
  it('returns zero when total is invalid', () => {
    expect(percentOf(10, 0)).toBe(0)
  })
})

describe('sortNodes', () => {
  it('sorts by size descending by default', () => {
    const nodes: TreeNode[] = [
      { path: 'a', name: 'a', type: 'file', sizeBytes: 5, modTime: '', childCount: 0 },
      { path: 'b', name: 'b', type: 'file', sizeBytes: 10, modTime: '', childCount: 0 }
    ]

    const sort: SortState = { by: 'size', direction: 'desc' }
    const sorted = sortNodes(nodes, sort)
    expect(sorted[0].path).toBe('b')
  })
})
