import { Link } from 'react-router-dom'
import { Incident } from '../api'
import { colors } from '../theme'

function timeAgo(iso: string): string {
  const sec = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (sec < 60) return `${sec}s ago`
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`
  return `${Math.floor(sec / 86400)}d ago`
}

function incidentTitle(inc: Incident): string {
  const name = inc.monitor_name || 'Monitor'
  if (inc.type === 'down') return `${name} is down`
  if (inc.type === 'slow') return `High latency on ${name}`
  if (inc.type === 'ssl_expiry' || inc.type?.includes('ssl')) return `SSL issue on ${name}`
  if (inc.type === 'recovery') return `${name} recovered`
  return `${name}: ${inc.type}`
}

export default function DashboardRail({
  incidents,
  uptimePct,
}: {
  incidents: Incident[]
  uptimePct: number | null
}) {
  const byNewest = [...incidents].sort(
    (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime(),
  )
  const last = byNewest[0]
  const recent = byNewest.slice(0, 5)

  const dayBars = buildDayBars(incidents, 30)

  return (
    <aside style={styles.rail}>
      <section style={styles.card}>
        <div style={styles.cardLabel}>Last Incident</div>
        {last ? (
          <>
            <div style={styles.lastTime}>{timeAgo(last.started_at)}</div>
            <div style={styles.lastTitle}>{incidentTitle(last)}</div>
            {last.message && (
              <div style={styles.lastMsg}>{last.message}</div>
            )}
            <div style={styles.lastMeta}>
              {last.resolved_at ? 'Resolved' : 'Open'}
            </div>
            <Link to="/incidents" style={styles.link}>View incidents →</Link>
          </>
        ) : (
          <div style={styles.empty}>No incidents yet</div>
        )}
      </section>

      <section style={styles.card}>
        <div style={styles.cardHeader}>
          <span style={styles.cardLabel}>Recent Incidents</span>
          <Link to="/incidents" style={styles.linkMuted}>View all</Link>
        </div>
        {recent.length === 0 ? (
          <div style={styles.empty}>All clear</div>
        ) : (
          <ul style={styles.list}>
            {recent.map(inc => (
              <li key={inc.id} style={styles.listItem}>
                <span style={{
                  ...styles.dot,
                  background: inc.resolved_at ? colors.green : (inc.type === 'slow' ? colors.yellow : colors.red),
                }} />
                <div style={{ minWidth: 0, flex: 1 }}>
                  <div style={styles.listTitle}>{incidentTitle(inc)}</div>
                  <div style={styles.listMeta}>
                    {timeAgo(inc.started_at)}
                    {inc.resolved_at ? ' · resolved' : ''}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section style={styles.card}>
        <div style={styles.cardLabel}>Uptime (30 days)</div>
        <div style={styles.uptimeValue}>
          {uptimePct == null ? '—' : `${uptimePct.toFixed(2)}%`}
        </div>
        <div style={styles.bars}>
          {dayBars.map((ok, i) => (
            <span
              key={i}
              title={ok ? 'Healthy' : 'Incident'}
              style={{
                ...styles.bar,
                background: ok ? colors.green : colors.red,
                opacity: ok ? 0.85 : 1,
              }}
            />
          ))}
        </div>
      </section>
    </aside>
  )
}

function buildDayBars(incidents: Incident[], days: number): boolean[] {
  const now = new Date()
  const bars: boolean[] = []
  for (let i = days - 1; i >= 0; i--) {
    const dayStart = new Date(now)
    dayStart.setHours(0, 0, 0, 0)
    dayStart.setDate(dayStart.getDate() - i)
    const dayEnd = new Date(dayStart)
    dayEnd.setDate(dayEnd.getDate() + 1)
    const hit = incidents.some(inc => {
      if (inc.type === 'recovery') return false
      const start = new Date(inc.started_at).getTime()
      const end = inc.resolved_at ? new Date(inc.resolved_at).getTime() : Date.now()
      return start < dayEnd.getTime() && end >= dayStart.getTime()
    })
    bars.push(!hit)
  }
  return bars
}

const styles: Record<string, React.CSSProperties> = {
  rail: {
    display: 'flex',
    flexDirection: 'column',
    gap: 16,
    width: 300,
    flex: '0 1 300px',
  },
  card: {
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: colors.radius,
    padding: 24,
  },
  cardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 4,
  },
  cardLabel: {
    fontSize: 12,
    fontWeight: 600,
    color: colors.textMuted,
    textTransform: 'uppercase',
    letterSpacing: '0.04em',
    marginBottom: 12,
  },
  lastTime: {
    fontSize: 28,
    fontWeight: 700,
    letterSpacing: '-0.02em',
    marginBottom: 8,
  },
  lastTitle: {
    fontSize: 14,
    fontWeight: 600,
    marginBottom: 6,
  },
  lastMsg: {
    fontSize: 12,
    color: colors.textMuted,
    marginBottom: 8,
    lineHeight: 1.4,
  },
  lastMeta: {
    fontSize: 12,
    fontWeight: 600,
    color: colors.textMuted,
    marginBottom: 12,
  },
  link: {
    fontSize: 13,
    fontWeight: 600,
    color: colors.brand,
  },
  linkMuted: {
    fontSize: 12,
    fontWeight: 500,
    color: colors.textMuted,
    marginBottom: 12,
  },
  empty: {
    fontSize: 13,
    color: colors.textMuted,
  },
  list: {
    listStyle: 'none',
    margin: 0,
    padding: 0,
    display: 'flex',
    flexDirection: 'column',
    gap: 14,
  },
  listItem: {
    display: 'flex',
    gap: 10,
    alignItems: 'flex-start',
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: '50%',
    marginTop: 5,
    flexShrink: 0,
  },
  listTitle: {
    fontSize: 13,
    fontWeight: 600,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  listMeta: {
    fontSize: 11,
    color: colors.textMuted,
    marginTop: 2,
  },
  uptimeValue: {
    fontSize: 28,
    fontWeight: 700,
    letterSpacing: '-0.02em',
    marginBottom: 16,
  },
  bars: {
    display: 'flex',
    gap: 3,
    height: 28,
    alignItems: 'flex-end',
  },
  bar: {
    flex: 1,
    height: '100%',
    borderRadius: 3,
    minWidth: 0,
  },
}
