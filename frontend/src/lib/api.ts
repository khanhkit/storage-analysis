import type { ApiResponse, DriveData, DuplicatePayload, OperationData, ScanPayload, StatsPayload } from './types'

const API_BASE = import.meta.env.VITE_API_BASE ?? ''

// The credential stays in memory only: never store filesystem access tokens
// in localStorage, URLs, cookies, or persistent browser state.
let operationToken: string | null = null

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const send = () => fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(operationToken ? { Authorization: `Bearer ${operationToken}` } : {}),
      ...options?.headers
    }
  })

  let response = await send()
  if (response.status === 401) {
    operationToken = null
    const entered = window.prompt('Storage Analysis access token (not saved):')
    if (entered?.trim()) {
      operationToken = entered.trim()
      response = await send()
      if (response.status === 401) operationToken = null
    }
  }

  const payload = (await response.json()) as ApiResponse<T>;
  if (!response.ok || !payload.success) {
    throw new Error(payload.error ?? 'Request failed')
  }

  if (payload.data === undefined) {
    throw new Error('No data returned')
  }

  return payload.data
}

export async function fetchScan(path: string, depth: number, useCache: boolean): Promise<ScanPayload> {
  const query = new URLSearchParams({
    path,
    depth: depth.toString(),
    useCache: useCache ? '1' : '0'
  })
  return request<ScanPayload>(`/api/scan?${query.toString()}`)
}

export async function fetchChildren(path: string, depth: number, useCache: boolean): Promise<ScanPayload> {
  const query = new URLSearchParams({
    path,
    depth: depth.toString(),
    useCache: useCache ? '1' : '0'
  })
  return request<ScanPayload>(`/api/children?${query.toString()}`)
}

export async function fetchDrives(): Promise<string[]> {
  const data = await request<DriveData>('/api/drives')
  return data.drives
}

export async function fetchStats(): Promise<StatsPayload> {
  return request<StatsPayload>('/api/stats')
}

export async function fetchDuplicates(path: string, minSizeBytes: number): Promise<DuplicatePayload> {
  const query = new URLSearchParams({
    path,
    minSizeBytes: minSizeBytes.toString()
  })
  return request<DuplicatePayload>(`/api/duplicates?${query.toString()}`)
}

export async function copyItem(source: string, destination: string): Promise<OperationData> {
  return request<OperationData>('/api/ops/copy', {
    method: 'POST',
    body: JSON.stringify({ source, destination })
  })
}

export async function moveItem(source: string, destination: string): Promise<OperationData> {
  return request<OperationData>('/api/ops/move', {
    method: 'POST',
    body: JSON.stringify({ source, destination })
  })
}

export async function deleteItem(path: string): Promise<{ status: string }> {
  return request<{ status: string }>('/api/ops/delete', {
    method: 'POST',
    body: JSON.stringify({ path })
  })
}
