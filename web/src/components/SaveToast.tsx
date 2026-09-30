import { useEffect, useState } from 'react'
import ModalCloseButton from './ModalCloseButton'

/** Centered success dialog — two staggered pulses for 2s, then static check. */
export default function SaveToast({
  open,
  message = 'Changes saved',
  detail = 'Your changes have been saved successfully.',
  onClose,
}: {
  open: boolean
  message?: string
  detail?: string
  durationMs?: number
  onClose: () => void
}) {
  const [pulsing, setPulsing] = useState(true)

  useEffect(() => {
    if (!open) {
      setPulsing(true)
      return
    }
    setPulsing(true)
    const t = window.setTimeout(() => setPulsing(false), 2000)
    return () => window.clearTimeout(t)
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="save-toast"
      role="presentation"
      onMouseDown={e => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        className={`save-toast-card${pulsing ? ' is-pulsing' : ' is-settled'}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="save-toast-title"
        aria-describedby="save-toast-detail"
      >
        <div className="save-toast-head">
          <span className="save-toast-spacer" aria-hidden />
          <ModalCloseButton onClick={onClose} />
        </div>

        <div className="save-toast-icon" aria-hidden>
          <span className="save-toast-pulse save-toast-pulse--a" />
          <span className="save-toast-pulse save-toast-pulse--b" />
          <span className="save-toast-spark spark-1" />
          <span className="save-toast-spark spark-2" />
          <span className="save-toast-spark spark-3" />
          <span className="save-toast-spark spark-4" />
          <span className="save-toast-spark spark-5" />
          <span className="save-toast-badge">
            <svg viewBox="0 0 48 48" fill="none">
              <circle cx="24" cy="24" r="18" className="save-toast-ring" />
              <path className="save-toast-mark" d="M15 24.5l6 6 12-13" />
            </svg>
          </span>
        </div>

        <h3 id="save-toast-title" className="save-toast-title">{message}</h3>
        <p id="save-toast-detail" className="save-toast-detail">{detail}</p>

        <button type="button" className="btn btn-primary save-toast-ok" onClick={onClose} autoFocus>
          OK
        </button>
      </div>
    </div>
  )
}
