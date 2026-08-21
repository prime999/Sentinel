import { useEffect, useRef, useState } from 'react'
import { PollableTarget, secondsUntilNextCheck } from '../utils/poll'

function formatAgo(iso?: string): string | null {
  if (!iso) return null
  const sec = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000))
  if (sec < 60) return `${sec}s ago`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`
  return `${Math.floor(sec / 86400)}d ago`
}

export default function NextCheckCountdown({
  target,
  onDue,
  disabledLabel = 'Paused',
}: {
  target: PollableTarget
  onDue?: () => void
  disabledLabel?: string
}) {
  const targetRef = useRef(target)
  targetRef.current = target
  const onDueRef = useRef(onDue)
  onDueRef.current = onDue
  const [seconds, setSeconds] = useState(() => secondsUntilNextCheck(target))
  const [checkedAgo, setCheckedAgo] = useState(() => formatAgo(target.last_checked_at))

  useEffect(() => {
    const sync = () => {
      const next = secondsUntilNextCheck(targetRef.current)
      setSeconds(prev => {
        if (prev > 0 && next === 0) onDueRef.current?.()
        return next
      })
      setCheckedAgo(formatAgo(targetRef.current.last_checked_at))
    }
    sync()
    const tick = setInterval(sync, 1000)
    return () => clearInterval(tick)
  }, [target.last_checked_at, target.interval_seconds, target.enabled])

  const interval = Math.max(1, target.interval_seconds ?? 60)
  const checking = target.enabled && seconds === 0
  const elapsed = target.enabled ? Math.min(interval, interval - seconds) : 0
  const pct = target.enabled ? Math.round((elapsed / interval) * 100) : 0

  let status = disabledLabel
  let pulse: 'off' | 'live' | 'checking' = 'off'
  if (target.enabled) {
    if (checking) {
      status = 'Checking now'
      pulse = 'checking'
    } else if (!target.last_checked_at) {
      status = 'Waiting for first probe'
      pulse = 'live'
    } else {
      status = 'Monitoring Active'
      pulse = 'live'
    }
  }

  return (
    <div className="probe-status-card">
      <div className="probe-status-row">
        <span className={`probe-pulse-dot probe-pulse-dot--${pulse}`} />
        <span className="probe-status-title">{status}</span>
      </div>

      <div className="probe-next">
        {target.enabled
          ? (checking ? 'Probe in progress' : `Next probe in ${seconds}s`)
          : 'Probes paused'}
      </div>

      <div className="probe-bar" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
        <div className={`probe-bar-fill${checking ? ' probe-bar-fill--checking' : ''}`} style={{ width: `${pct}%` }} />
      </div>
      <div className="probe-bar-meta">{pct}%</div>

      <div className="probe-status-row probe-checked">
        <span className={`probe-pulse-dot probe-pulse-dot--${pulse === 'off' ? 'off' : 'live'}`} />
        <span>
          {checkedAgo ? `Checked ${checkedAgo}` : 'Not checked yet'}
        </span>
      </div>
    </div>
  )
}
