import { useCallback, useEffect, useMemo, useState } from 'react'
import { copyItem, deleteItem, fetchChildren, fetchDrives, fetchDuplicates, fetchScan, fetchStats, moveItem } from './lib/api'
import type { DuplicatePayload, ScanPayload, SortState, StatsPayload, TreeNode } from './lib/types'
import TreeView from './components/TreeView'
import { formatBytes } from './lib/format'

const DEFAULT_PATH = 'C:\\'

function App() {
  const [rootPath, setRootPath] = useState(
    () => localStorage.getItem('storage-root') ?? DEFAULT_PATH
  )
  const [drives, setDrives] = useState<string[]>([])
  const [scan, setScan] = useState<ScanPayload | null>(null)
  const [tree, setTree] = useState<TreeNode | null>(null)
  const [loading, setLoading] = useState(false)
  const [loadingPaths, setLoadingPaths] = useState<Set<string>>(new Set())
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [warnings, setWarnings] = useState<string[]>([])
  const [status, setStatus] = useState<string>('')
  const [destination, setDestination] = useState('')
  const [useCache, setUseCache] = useState(true)
  const [sort, setSort] = useState<SortState>({ by: 'size', direction: 'desc' })
  const [stats, setStats] = useState<StatsPayload | null>(null)
  const [activeTab, setActiveTab] = useState<'analysis' | 'duplicates'>('analysis')
  const [duplicateResults, setDuplicateResults] = useState<DuplicatePayload | null>(null)
  const [duplicateLoading, setDuplicateLoading] = useState(false)
  const [duplicateStatus, setDuplicateStatus] = useState('')
  const [duplicateMinSizeMb, setDuplicateMinSizeMb] = useState(() => {
    const stored = localStorage.getItem('storage-duplicate-min-mb')
    const parsed = stored ? Number.parseFloat(stored) : 10
    if (!Number.isFinite(parsed) || parsed < 0) {
      return 10
    }
    return parsed
  })

  useEffect(() => {
    let active = true

    fetchDrives()
      .then((list) => {
        if (active) {
          setDrives(list)
        }
      })
      .catch(() => {
        if (active) {
          setDrives([])
        }
      })

    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    let active = true
    let timer: number | undefined

    const tick = async () => {
      try {
        const data = await fetchStats()
        if (active) {
          setStats(data)
        }
      } catch {
        if (active) {
          setStats((prev) => prev)
        }
      }
    }

    tick()
    const interval = loading ? 1000 : 3000
    timer = window.setInterval(tick, interval)

    return () => {
      active = false
      if (timer) {
        window.clearInterval(timer)
      }
    }
  }, [loading])

  useEffect(() => {
    localStorage.setItem('storage-duplicate-min-mb', duplicateMinSizeMb.toString())
  }, [duplicateMinSizeMb])

  const rememberRoot = (path: string) => {
    localStorage.setItem('storage-root', path)
  }

  const updateTreeChildren = useCallback(
    (node: TreeNode, targetPath: string, children: TreeNode[]): TreeNode => {
      if (node.path === targetPath) {
        return { ...node, children }
      }

      if (!node.children || node.children.length === 0) {
        return node
      }

      return {
        ...node,
        children: node.children.map((child) =>
          updateTreeChildren(child, targetPath, children)
        )
      }
    },
    []
  )

  const loadRoot = useCallback(async () => {
    setLoading(true)
    setStatus('Scanning…')
    try {
      const payload = await fetchScan(rootPath, -1, useCache)
      setScan(payload)
      setTree(payload.node)
      setWarnings(payload.warnings ?? [])
      setExpanded(new Set([payload.node.path]))
      setSelected(new Set())
      setStatus(payload.fromCache ? 'Loaded from cache.' : 'Fresh scan complete.')
      rememberRoot(rootPath)
    } catch (error) {
      setStatus(error instanceof Error ? error.message : 'Scan failed')
    } finally {
      setLoading(false)
    }
  }, [rootPath, useCache])

  const loadChildren = useCallback(
    async (node: TreeNode) => {
      if (node.type !== 'dir') {
        return
      }

      setLoadingPaths((prev) => new Set(prev).add(node.path))
      try {
        const payload = await fetchChildren(node.path, 1, useCache)
        setTree((prev) => (prev ? updateTreeChildren(prev, node.path, payload.node.children ?? []) : prev))
      } catch (error) {
        setStatus(error instanceof Error ? error.message : 'Failed to load children')
      } finally {
        setLoadingPaths((prev) => {
          const next = new Set(prev)
          next.delete(node.path)
          return next
        })
      }
    },
    [updateTreeChildren, useCache]
  )

  const toggleExpanded = useCallback(
    async (node: TreeNode) => {
      if (node.type !== 'dir') {
        return
      }

      setExpanded((prev) => {
        const next = new Set(prev)
        if (next.has(node.path)) {
          next.delete(node.path)
        } else {
          next.add(node.path)
        }
        return next
      })

      if (node.childCount > 0 && (!node.children || node.children.length === 0)) {
        await loadChildren(node)
      }
    },
    [loadChildren]
  )

  const toggleSelected = useCallback((node: TreeNode) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(node.path)) {
        next.delete(node.path)
      } else {
        next.add(node.path)
      }
      return next
    })
  }, [])

  const clearSelection = () => setSelected(new Set())

  const runBulkOperation = useCallback(
    async (mode: 'copy' | 'move' | 'delete') => {
      if (selected.size === 0) {
        setStatus('Select at least one item.')
        return
      }

      if (mode !== 'delete' && destination.trim() === '') {
        setStatus('Destination folder is required.')
        return
      }

      if (mode === 'delete') {
        const confirmed = window.confirm('Delete selected items? This cannot be undone.')
        if (!confirmed) {
          return
        }
      }

      setLoading(true)
      setStatus(`${mode.toUpperCase()} in progress…`)
      try {
        for (const path of selected) {
          if (mode === 'copy') {
            await copyItem(path, destination)
          } else if (mode === 'move') {
            await moveItem(path, destination)
          } else {
            await deleteItem(path)
          }
        }
        clearSelection()
        await loadRoot()
        setStatus(`${mode.toUpperCase()} completed.`)
      } catch (error) {
        setStatus(error instanceof Error ? error.message : 'Operation failed')
      } finally {
        setLoading(false)
      }
    },
    [selected, destination, loadRoot]
  )

  const runDuplicateScan = useCallback(async () => {
    setDuplicateLoading(true)
    setDuplicateStatus('Scanning duplicates…')
    setDuplicateResults(null)
    try {
      const minBytes = Math.max(0, Math.round(duplicateMinSizeMb * 1024 * 1024))
      const payload = await fetchDuplicates(rootPath, minBytes)
      setDuplicateResults(payload)
      setDuplicateStatus(`Found ${payload.groups.length} groups.`)
    } catch (error) {
      setDuplicateStatus(error instanceof Error ? error.message : 'Duplicate scan failed')
    } finally {
      setDuplicateLoading(false)
    }
  }, [duplicateMinSizeMb, rootPath])

  const totalSize = tree?.sizeBytes ?? 0
  const totalItems = tree?.childCount ?? 0
  const analysisScanning = loading || stats?.scanning
  const duplicateScanning = duplicateLoading || stats?.duplicateScanning
  const countFormatter = useMemo(() => new Intl.NumberFormat('en-US'), [])
  const formatCount = (value: number | null | undefined) => countFormatter.format(value ?? 0)
  const cpuPercent = stats ? Math.round(stats.cpuPercent) : 0
  const processCpuPercent = stats ? Math.round(stats.processCpuPercent) : 0
  const elapsedSeconds = stats ? (stats.elapsedMs / 1000).toFixed(1) : '0.0'
  const duplicateElapsedSeconds = stats ? (stats.duplicateElapsedMs / 1000).toFixed(1) : '0.0'

  const statusMeta = useMemo(() => {
    if (!scan) {
      return 'No scan yet.'
    }
    const scannedAt = new Date(scan.scannedAt).toLocaleString()
    return `${scan.fromCache ? 'Cache' : 'Fresh'} · ${scannedAt}`
  }, [scan])

  return (
    <div className="app">
      <header className="app-header">
        <div>
          <p className="eyebrow">Storage Intelligence</p>
          <h1>Storage Analysis Dashboard</h1>
          <p className="subtitle">Tốc độ cao, cache mạnh, quản lý file trực quan.</p>
        </div>
        <div className="summary">
          <div>
            <span>Tổng dung lượng</span>
            <strong>{formatBytes(totalSize)}</strong>
          </div>
          <div>
            <span>Mục con trực tiếp</span>
            <strong>{totalItems}</strong>
          </div>
        </div>
      </header>

      <div className="app-shell">
        <aside className="sidebar">
          <section className="panel">
            <h2>Phân tích nhanh</h2>
            <label className="field">
              <span>Root path</span>
              <input
                value={rootPath}
                onChange={(event) => setRootPath(event.target.value)}
                placeholder="C:\\"
              />
            </label>
            {drives.length > 0 && (
              <div className="drive-list">
                {drives.map((drive) => (
                  <button
                    key={drive}
                    className={`chip ${rootPath === drive ? 'active' : ''}`}
                    onClick={() => setRootPath(drive)}
                  >
                    {drive}
                  </button>
                ))}
              </div>
            )}
            <label className="field toggle">
              <input
                type="checkbox"
                checked={useCache}
                onChange={(event) => setUseCache(event.target.checked)}
              />
              <span>Ưu tiên cache + reload nhanh</span>
            </label>
            {activeTab === 'analysis' && (
              <button className="button primary" onClick={loadRoot} disabled={loading}>
                {loading ? 'Đang quét…' : 'Analyze'}
              </button>
            )}
            {activeTab === 'duplicates' && (
              <div className="duplicate-controls">
                <label className="field">
                  <span>Min size (MB)</span>
                  <input
                    type="number"
                    min={0}
                    step={1}
                    value={duplicateMinSizeMb}
                    onChange={(event) => {
                      const nextValue = Number.parseFloat(event.target.value)
                      if (!Number.isFinite(nextValue) || nextValue < 0) {
                        setDuplicateMinSizeMb(0)
                      } else {
                        setDuplicateMinSizeMb(nextValue)
                      }
                    }}
                  />
                </label>
                <button className="button primary" onClick={runDuplicateScan} disabled={duplicateLoading}>
                  {duplicateLoading ? 'Đang quét…' : 'Scan duplicates'}
                </button>
                {duplicateStatus && <p className="status highlight">{duplicateStatus}</p>}
              </div>
            )}
            <p className="status">{statusMeta}</p>
            {status && <p className="status highlight">{status}</p>}
            {activeTab === 'analysis' && analysisScanning && (
              <div className="progress">
                <div className="progress-bar">
                  <span className="progress-fill" />
                </div>
                <p className="status">Đang quét: {formatCount(stats?.filesScanned)} files · {formatCount(stats?.dirsScanned)} folders · {elapsedSeconds}s</p>
              </div>
            )}
            {activeTab === 'duplicates' && duplicateScanning && (
              <div className="progress">
                <div className="progress-bar">
                  <span className="progress-fill" />
                </div>
                <p className="status">Đang quét: {formatCount(stats?.duplicateFilesScanned)} files · {formatCount(stats?.duplicateDirsScanned)} folders · {formatCount(stats?.duplicateHashed)} hashed · {duplicateElapsedSeconds}s</p>
              </div>
            )}
            <div className="stats-grid">
              <div className="stat">
                <span>CPU load</span>
                <strong>{cpuPercent}%</strong>
              </div>
              <div className="stat">
                <span>Process CPU</span>
                <strong>{processCpuPercent}%</strong>
              </div>
              <div className="stat">
                <span>Cores</span>
                <strong>{stats?.cores ?? 0}</strong>
              </div>
              <div className="stat">
                <span>Threads</span>
                <strong>{stats?.threads ?? 0}</strong>
              </div>
              <div className="stat">
                <span>Goroutines</span>
                <strong>{stats?.goroutines ?? 0}</strong>
              </div>
            </div>
            {warnings.length > 0 && (
              <div className="warnings">
                <p>Warnings</p>
                <ul>
                  {warnings.slice(0, 3).map((warning) => (
                    <li key={warning}>{warning}</li>
                  ))}
                </ul>
              </div>
            )}
          </section>

          <section className="panel">
            <h2>Sort & hành động</h2>
            <div className="field-group">
              <label className="field">
                <span>Sort by</span>
                <select
                  value={sort.by}
                  onChange={(event) => setSort((prev) => ({ ...prev, by: event.target.value as SortState['by'] }))}
                >
                  <option value="size">Size</option>
                  <option value="name">Name</option>
                  <option value="items">Items</option>
                </select>
              </label>
              <label className="field">
                <span>Order</span>
                <select
                  value={sort.direction}
                  onChange={(event) =>
                    setSort((prev) => ({
                      ...prev,
                      direction: event.target.value as SortState['direction']
                    }))
                  }
                >
                  <option value="desc">Desc</option>
                  <option value="asc">Asc</option>
                </select>
              </label>
            </div>

            <label className="field">
              <span>Destination (copy/move)</span>
              <input
                value={destination}
                onChange={(event) => setDestination(event.target.value)}
                placeholder="D:\\Backup"
              />
            </label>

            <div className="button-row">
              <button className="button" onClick={() => runBulkOperation('copy')} disabled={loading}>
                Copy
              </button>
              <button className="button" onClick={() => runBulkOperation('move')} disabled={loading}>
                Move
              </button>
              <button className="button danger" onClick={() => runBulkOperation('delete')} disabled={loading}>
                Delete
              </button>
            </div>
            <p className="helper">Đã chọn: {selected.size} item</p>
          </section>
        </aside>

        <main className="main">
          <div className="tabs">
            <button
              className={`tab ${activeTab === 'analysis' ? 'active' : ''}`}
              onClick={() => setActiveTab('analysis')}
            >
              Analysis
            </button>
            <button
              className={`tab ${activeTab === 'duplicates' ? 'active' : ''}`}
              onClick={() => setActiveTab('duplicates')}
            >
              Duplicates
            </button>
          </div>
          {activeTab === 'analysis' && (
            <section className="panel table-panel">
              <div className="table-header">
                <h2>Storage Tree</h2>
                <span className="muted">Click folder để mở nhanh.</span>
              </div>
              {tree ? (
                <TreeView
                  root={tree}
                  expanded={expanded}
                  selected={selected}
                  loading={loadingPaths}
                  sort={sort}
                  onToggle={toggleExpanded}
                  onSelect={toggleSelected}
                />
              ) : (
                <div className="empty-state">
                  <p>Chưa có dữ liệu. Nhấn Analyze để bắt đầu.</p>
                </div>
              )}
            </section>
          )}
          {activeTab === 'duplicates' && (
            <section className="panel table-panel">
              <div className="table-header">
                <h2>Duplicate Files</h2>
                <span className="muted">Files lớn hơn ngưỡng sẽ được so hash.</span>
              </div>
              {duplicateResults ? (
                <div className="duplicates">
                  <p className="helper">Tổng file kiểm tra: {duplicateResults.fileCount} · File trùng: {duplicateResults.duplicateFiles}</p>
                  {duplicateResults.warnings && duplicateResults.warnings.length > 0 && (
                    <div className="warnings">
                      <p>Warnings</p>
                      <ul>
                        {duplicateResults.warnings.slice(0, 3).map((warning) => (
                          <li key={warning}>{warning}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                  {duplicateResults.groups.length === 0 && (
                    <div className="empty-state">
                      <p>Không tìm thấy file trùng.</p>
                    </div>
                  )}
                  {duplicateResults.groups.map((group) => (
                    <div key={group.hash} className="duplicate-group">
                      <div className="duplicate-header">
                        <strong>{formatBytes(group.sizeBytes)}</strong>
                        <span className="badge">{group.paths.length} files</span>
                      </div>
                      <ul>
                        {group.paths.map((path) => (
                          <li key={path}>{path}</li>
                        ))}
                      </ul>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="empty-state">
                  <p>Nhấn "Scan duplicates" để bắt đầu.</p>
                </div>
              )}
            </section>
          )}
        </main>
      </div>
    </div>
  )
}

export default App
