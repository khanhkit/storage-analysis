import TreeRow from './TreeRow'
import type { SortState, TreeNode } from '../lib/types'
import { sortNodes } from '../lib/format'

interface TreeViewProps {
  root: TreeNode
  expanded: Set<string>
  selected: Set<string>
  loading: Set<string>
  sort: SortState
  onToggle: (node: TreeNode) => void
  onSelect: (node: TreeNode) => void
}

function TreeView({ root, expanded, selected, loading, sort, onToggle, onSelect }: TreeViewProps) {
  const renderNode = (node: TreeNode, depth: number, parentSize: number) => {
    const isExpanded = expanded.has(node.path)
    const isLoading = loading.has(node.path)
    const children = node.children ? sortNodes(node.children, sort) : []

    return (
      <div key={node.path}>
        <TreeRow
          node={node}
          depth={depth}
          parentSize={parentSize}
          expanded={isExpanded}
          loading={isLoading}
          selected={selected.has(node.path)}
          onToggle={onToggle}
          onSelect={onSelect}
        />
        {isExpanded && node.type === 'dir' && children.map((child) => renderNode(child, depth + 1, node.sizeBytes))}
        {isExpanded && node.type === 'dir' && children.length === 0 && node.childCount > 0 && (
          <div className="tree-row placeholder" style={{ paddingLeft: `${depth + 1}rem` }}>
            <span>Đang tải...</span>
          </div>
        )}
      </div>
    )
  }

  return <div className="tree">{renderNode(root, 0, root.sizeBytes)}</div>
}

export default TreeView
