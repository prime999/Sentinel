import { useCallback, useEffect, useState } from 'react'

const MIN_COL = 64
const MAX_FIT = 720

function storageKey(id: string) {
  return `sentinel.colw.${id}`
}

export function useColumnResize(tableId: string, columnCount: number) {
  const [widths, setWidths] = useState<(number | null)[]>(() => {
    try {
      const raw = localStorage.getItem(storageKey(tableId))
      if (!raw) return Array(columnCount).fill(null)
      const parsed = JSON.parse(raw) as unknown
      if (!Array.isArray(parsed) || parsed.length !== columnCount) return Array(columnCount).fill(null)
      return parsed.map(v => (typeof v === 'number' && v > 0 ? v : null))
    } catch {
      return Array(columnCount).fill(null)
    }
  })

  useEffect(() => {
    try {
      localStorage.setItem(storageKey(tableId), JSON.stringify(widths))
    } catch {
      /* ignore quota */
    }
  }, [tableId, widths])

  const startResize = useCallback((index: number, e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    const th = (e.currentTarget as HTMLElement).closest('th')
    const startW = th?.getBoundingClientRect().width ?? 120
    const startX = e.clientX

    const onMove = (ev: MouseEvent) => {
      const next = Math.max(MIN_COL, Math.round(startW + (ev.clientX - startX)))
      setWidths(prev => {
        const copy = [...prev]
        copy[index] = next
        return copy
      })
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }, [])

  const autoFit = useCallback((index: number, table: HTMLTableElement | null) => {
    if (!table) return
    let max = MIN_COL
    const cells = table.querySelectorAll(`tr > *:nth-child(${index + 1})`)
    cells.forEach(node => {
      const el = node as HTMLElement
      max = Math.max(max, Math.ceil(el.scrollWidth) + 20)
    })
    setWidths(prev => {
      const copy = [...prev]
      copy[index] = Math.min(MAX_FIT, max)
      return copy
    })
  }, [])

  return { widths, startResize, autoFit }
}

export function ResizeHandle({
  onDrag,
  onFit,
}: {
  onDrag: (e: React.MouseEvent) => void
  onFit?: () => void
}) {
  return (
    <span
      className="col-resize-handle"
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize column"
      title="Drag to resize column · Double-click to fit content"
      onMouseDown={onDrag}
      onDoubleClick={e => {
        e.preventDefault()
        e.stopPropagation()
        onFit?.()
      }}
    />
  )
}

export function ColGroup({ widths }: { widths: (number | null)[] }) {
  return (
    <colgroup>
      {widths.map((w, i) => (
        <col key={i} style={w ? { width: w } : undefined} />
      ))}
    </colgroup>
  )
}

export function ResizableTh({
  index,
  children,
  style,
  startResize,
  autoFit,
  tableRef,
  className,
}: {
  index: number
  children?: React.ReactNode
  style?: React.CSSProperties
  startResize: (index: number, e: React.MouseEvent) => void
  autoFit: (index: number, table: HTMLTableElement | null) => void
  tableRef: React.RefObject<HTMLTableElement | null>
  className?: string
}) {
  return (
    <th className={className} style={style}>
      {children}
      <ResizeHandle onDrag={e => startResize(index, e)} onFit={() => autoFit(index, tableRef.current)} />
    </th>
  )
}

