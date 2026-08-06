import { useEffect, useRef } from 'react'

export interface PollableTarget {
  enabled?: boolean
  last_checked_at?: string
  interval_seconds?: number
}

export function secondsUntilNextCheck(target: PollableTarget): number {
  const interval = target.interval_seconds ?? 60
  if (!target.last_checked_at) return interval
  const elapsed = Math.floor((Date.now() - new Date(target.last_checked_at).getTime()) / 1000)
  return Math.max(0, interval - elapsed)
}

export function pollDelayMs(target: PollableTarget | null): number {
  if (!target?.enabled) return 30000
  if (!target.last_checked_at) return 3000
  const remaining = secondsUntilNextCheck(target)
  if (remaining <= 0) return 2000
  return Math.min(60000, (remaining + 1) * 1000)
}

export function useAdaptivePoll<T extends PollableTarget>(
  id: string | undefined,
  load: () => Promise<T | null>,
  deps: unknown[] = [],
) {
  const refreshRef = useRef<(() => void) | null>(null)
  const loadRef = useRef(load)
  loadRef.current = load

  useEffect(() => {
    if (!id) return

    let cancelled = false
    let timeoutId: ReturnType<typeof setTimeout>

    const runLoad = async () => {
      try {
        return await loadRef.current()
      } catch {
        return null
      }
    }

    const schedule = (data: T | null) => {
      clearTimeout(timeoutId)
      if (cancelled) return
      timeoutId = setTimeout(async () => {
        const updated = await runLoad()
        if (!cancelled) schedule(updated)
      }, pollDelayMs(data))
    }

    const refresh = async () => {
      const updated = await runLoad()
      if (!cancelled) schedule(updated)
    }

    refreshRef.current = () => { void refresh() }
    void refresh()

    return () => {
      cancelled = true
      refreshRef.current = null
      clearTimeout(timeoutId)
    }
  }, [id, ...deps])

  return refreshRef
}
