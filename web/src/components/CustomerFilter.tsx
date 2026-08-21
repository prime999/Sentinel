import { useEffect, useMemo, useRef, useState } from 'react'
import { colors } from '../theme'

export const INTERNAL_CUSTOMER_ID = '__internal__'

export type CustomerOption = { id: string; name: string }

interface Props {
  customers: CustomerOption[]
  selectedIds: string[]
  onChange: (ids: string[]) => void
  includeInternal?: boolean
}

export default function CustomerFilter({
  customers,
  selectedIds,
  onChange,
  includeInternal = true,
}: Props) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const rootRef = useRef<HTMLDivElement>(null)

  const options = useMemo(() => {
    const list: CustomerOption[] = includeInternal
      ? [{ id: INTERNAL_CUSTOMER_ID, name: 'Internal (unassigned)' }, ...customers]
      : [...customers]
    return list
  }, [customers, includeInternal])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return options
    return options.filter(o => o.name.toLowerCase().includes(q))
  }, [options, query])

  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [])

  function toggle(id: string) {
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter(x => x !== id))
    } else {
      onChange([...selectedIds, id])
    }
  }

  function selectAll() {
    onChange(options.map(o => o.id))
  }

  function clearAll() {
    onChange([])
  }

  const label = (() => {
    if (selectedIds.length === 0) return 'All customers'
    if (selectedIds.length === 1) {
      return options.find(o => o.id === selectedIds[0])?.name || '1 customer'
    }
    return `${selectedIds.length} customers`
  })()

  return (
    <div ref={rootRef} style={styles.wrap}>
      <button
        type="button"
        className="btn"
        onClick={() => setOpen(v => !v)}
        style={{
          ...styles.trigger,
          ...(open || selectedIds.length > 0 ? styles.triggerActive : {}),
        }}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label="Filter by customer"
      >
        <span style={styles.triggerLabel}>{label}</span>
        <span style={styles.chevron}>{open ? '▴' : '▾'}</span>
      </button>

      {open && (
        <div style={styles.panel} role="listbox" aria-multiselectable="true">
          <input
            className="input"
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search customers…"
            aria-label="Search customers"
            style={styles.search}
            autoFocus
          />
          <div style={styles.actions}>
            <button type="button" className="btn" style={styles.actionBtn} onClick={selectAll}>Select all</button>
            <button type="button" className="btn" style={styles.actionBtn} onClick={clearAll}>Clear</button>
          </div>
          <div style={styles.list}>
            {filtered.length === 0 ? (
              <div style={styles.empty}>No matches</div>
            ) : (
              filtered.map(o => {
                const checked = selectedIds.includes(o.id)
                return (
                  <label key={o.id} style={{ ...styles.row, ...(checked ? styles.rowChecked : {}) }}>
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => toggle(o.id)}
                      style={styles.checkbox}
                    />
                    <span style={styles.name}>{o.name}</span>
                  </label>
                )
              })
            )}
          </div>
        </div>
      )}
    </div>
  )
}

/** Match a resource tenant_id against selected filter IDs. Empty selection = all. */
export function matchesCustomerFilter(tenantId: string | undefined, selectedIds: string[]): boolean {
  if (selectedIds.length === 0) return true
  const tid = tenantId || ''
  if (!tid) return selectedIds.includes(INTERNAL_CUSTOMER_ID)
  return selectedIds.includes(tid)
}

const styles: Record<string, React.CSSProperties> = {
  wrap: { position: 'relative', display: 'inline-block' },
  trigger: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 8,
    minWidth: 180,
    justifyContent: 'space-between',
    fontSize: 13,
    padding: '8px 12px',
  },
  triggerActive: {
    background: colors.bgElevated,
    borderColor: colors.borderLight,
  },
  triggerLabel: {
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    maxWidth: 220,
  },
  chevron: { color: colors.textMuted, fontSize: 11 },
  panel: {
    position: 'absolute',
    top: 'calc(100% + 6px)',
    left: 0,
    zIndex: 40,
    width: 300,
    maxWidth: 'min(300px, 90vw)',
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: 10,
    boxShadow: '0 12px 40px rgba(0,0,0,0.45)',
    padding: 10,
  },
  search: {
    width: '100%',
    marginBottom: 8,
    fontSize: 13,
    padding: '8px 10px',
  },
  actions: {
    display: 'flex',
    gap: 6,
    marginBottom: 8,
  },
  actionBtn: {
    fontSize: 12,
    padding: '4px 8px',
    color: colors.textMuted,
  },
  list: {
    maxHeight: 280,
    overflowY: 'auto',
    display: 'flex',
    flexDirection: 'column',
    gap: 2,
  },
  row: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    padding: '8px 10px',
    borderRadius: 8,
    cursor: 'pointer',
    fontSize: 13,
    color: colors.text,
  },
  rowChecked: {
    background: colors.bgElevated,
  },
  checkbox: {
    width: 15,
    height: 15,
    accentColor: colors.brand,
    flexShrink: 0,
  },
  name: {
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  empty: {
    padding: 12,
    color: colors.textMuted,
    fontSize: 13,
    textAlign: 'center',
  },
}
