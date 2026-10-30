import type { TreeNode } from '../lib/types'
import { barColor, formatBytes, formatPercent, percentOf } from '../lib/format'

interface TreeRowProps {
  node: TreeNode
  depth: number
  parentSize: number
  expanded: boolean
  loading: boolean
  selected: boolean
  onToggle: (node: TreeNode) => void
  onSelect: (node: TreeNode) => void
}

function TreeRow({ node, depth, parentSize, expanded, loading, selected, onToggle, onSelect }: TreeRowProps) {
  const percent = percentOf(node.sizeBytes, parentSize)
  const barStyle = {
    width: `${percent * 100}%`,
    background: barColor(percent)
  }

  return (
    <div className={`tree-row ${selected ? 'selected' : ''}`}>
      <div className="tree-cell name" style={{ paddingLeft: `${depth * 1.1}rem` }}>
        {node.type === 'dir' && (
          <button className="toggle" onClick={() => onToggle(node)} aria-label="Toggle folder">
            {expanded ? '▾' : '▸'}
          </button>
        )}
        {node.type !== 'dir' && <span className="toggle spacer" />}
        <button className="select" onClick={() => onSelect(node)} aria-label="Select row">
          {selected ? '●' : '○'}
        </button>
        <span className={`icon ${node.type}`}>{node.type === 'dir' ? '📂' : '📄'}</span>
        <span className="label">{node.name || node.path}</span>
        {loading && <span className="badge">Loading</span>}
      </div>
      <div className="tree-cell size">
        <span>{formatBytes(node.sizeBytes)}</span>
      </div>
      <div className="tree-cell percent">
        <div className="bar">
          <span style={barStyle} />
        </div>
        <span>{formatPercent(percent)}</span>
      </div>
      <div className="tree-cell meta">
        <span>{node.type === 'dir' ? `${node.childCount} items` : 'file'}</span>
      </div>
    </div>
  )
}

export default TreeRow
