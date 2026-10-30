export type NodeType = 'file' | 'dir'

export interface TreeNode {
  path: string
  name: string
  type: NodeType
  sizeBytes: number
  modTime: string
  childCount: number
  children?: TreeNode[]
}

export interface ScanPayload {
  node: TreeNode
  fromCache: boolean
  modTime: string
  scannedAt: string
  warnings?: string[]
}

export interface DriveData {
  drives: string[]
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
}

export type SortBy = 'name' | 'size' | 'items'
export type SortDirection = 'asc' | 'desc'

export interface SortState {
  by: SortBy
  direction: SortDirection
}

export interface OperationData {
  mode: string
  source: string
  destination?: string
  exitCode: number
  durationMs: number
  output: string
}

export interface StatsPayload {
  cpuPercent: number
  processCpuPercent: number
  cores: number
  threads: number
  goroutines: number
  filesScanned: number
  dirsScanned: number
  scanning: boolean
  startedAt: string
  elapsedMs: number
  duplicateFilesScanned: number
  duplicateDirsScanned: number
  duplicateHashed: number
  duplicateScanning: boolean
  duplicateStartedAt: string
  duplicateElapsedMs: number
}

export interface DuplicateGroup {
  hash: string
  sizeBytes: number
  paths: string[]
}

export interface DuplicatePayload {
  groups: DuplicateGroup[]
  fileCount: number
  duplicateFiles: number
  scannedAt: string
  warnings?: string[]
}
