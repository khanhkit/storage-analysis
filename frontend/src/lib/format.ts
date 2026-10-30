import type { SortState, TreeNode } from './types'

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes)) {
    return '--'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = Math.max(bytes, 0)
  let unitIndex = 0

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }

  return `${value.toFixed(value < 10 && unitIndex > 0 ? 1 : 0)} ${units[unitIndex]}`
}

export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) {
    return '--'
  }
  return `${(value * 100).toFixed(1)}%`
}

export function sortNodes(nodes: TreeNode[], sort: SortState): TreeNode[] {
  const sorted = [...nodes]

  sorted.sort((a, b) => {
    if (a.type !== b.type) {
      return a.type === 'dir' ? -1 : 1
    }

    let comparison = 0
    if (sort.by === 'name') {
      comparison = a.name.localeCompare(b.name)
    } else if (sort.by === 'items') {
      comparison = a.childCount - b.childCount
    } else {
      comparison = a.sizeBytes - b.sizeBytes
    }

    return sort.direction === 'asc' ? comparison : -comparison
  })

  return sorted
}

export function percentOf(value: number, total: number): number {
  if (!Number.isFinite(value) || total <= 0) {
    return 0
  }
  return Math.min(Math.max(value / total, 0), 1)
}

export function barColor(percent: number): string {
  const clamped = Math.min(Math.max(percent, 0), 1)
  const hue = 160 - clamped * 140
  return `hsl(${hue} 70% 55%)`
}
