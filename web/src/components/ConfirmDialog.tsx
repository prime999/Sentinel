import { useEffect } from 'react'
import { colors } from '../theme'

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger = false,
  busy = false,
  onConfirm,
  onCancel,
}: {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  busy?: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !busy) onCancel()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, busy, onCancel])

  if (!open) return null

  return (
    <div
      style={styles.backdrop}
      role="presentation"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !busy) onCancel()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        aria-describedby="confirm-dialog-message"
        style={styles.dialog}
      >
        <h3 id="confirm-dialog-title" style={styles.title}>{title}</h3>
        <p id="confirm-dialog-message" style={styles.message}>{message}</p>
        <div style={styles.actions}>
          <button
            type="button"
            className="btn"
            disabled={busy}
            onClick={onCancel}
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            className={danger ? 'btn btn-danger' : 'btn btn-primary'}
            disabled={busy}
            onClick={onConfirm}
            autoFocus
          >
            {busy ? 'Please wait…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  backdrop: {
    position: 'fixed',
    inset: 0,
    zIndex: 1000,
    background: 'rgba(0, 0, 0, 0.55)',
    display: 'grid',
    placeItems: 'center',
    padding: 24,
  },
  dialog: {
    width: '100%',
    maxWidth: 420,
    background: colors.bgElevated,
    border: `1px solid ${colors.border}`,
    borderRadius: 14,
    padding: '24px 28px',
    boxShadow: '0 24px 48px rgba(0,0,0,0.45)',
  },
  title: {
    margin: '0 0 10px',
    fontSize: 18,
    fontWeight: 700,
    letterSpacing: '-0.01em',
  },
  message: {
    margin: '0 0 24px',
    fontSize: 14,
    lineHeight: 1.5,
    color: colors.textMuted,
  },
  actions: {
    display: 'flex',
    justifyContent: 'flex-end',
    gap: 10,
  },
}
