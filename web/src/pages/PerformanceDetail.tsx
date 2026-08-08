import { useCallback, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Area, AreaChart, CartesianGrid, Legend, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import { api, PerformanceResult, PerformanceStats, PerformanceTarget } from '../api'
import { useAuth } from '../context/AuthContext'
import MetricCard from '../components/MetricCard'
import { colors } from '../theme'
import { useAdaptivePoll } from '../utils/poll'

export default function PerformanceDetail() {
  const { isAdmin } = useAuth()
  const { id } = useParams<{ id: string }>()
  const [target, setTarget] = useState<PerformanceTarget | null>(null)
  const [stats, setStats] = useState<PerformanceStats | null>(null)
  const [results, setResults] = useState<PerformanceResult[]>([])
  const [period, setPeriod] = useState('24h')
  const periodRef = useRef(period)
  periodRef.current = period

  const load = useCallback(async () => {
    if (!id) return null
    const [t, s, r] = await Promise.all([
      api.getPerformanceTarget(id),
      api.performanceStats(id, periodRef.current),
      api.performanceResults(id),
    ])
    setTarget(t)
    setStats(s)
    setResults(r)
    return t
  }, [id])

  useAdaptivePoll(id, load, [period])

  if (!target) return <div style={{ color: colors.textMuted }}>Loading…</div>

  const perf = stats?.performance
  const chartData = (stats?.points || []).map(p => ({
    time: new Date(p.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    ms: p.response_time_ms,
    dns: p.dns_ms ?? 0,
    tcp: p.tcp_ms ?? 0,
    tls: p.tls_ms ?? 0,
    ttfb: p.ttfb_ms ?? 0,
    download: Math.max(0, p.response_time_ms - (p.ttfb_ms ?? p.response_time_ms)),
  }))

  return (
    <div className="page">
      <div style={styles.header}>
        <div>
          <Link to="/performance" style={styles.back}>← Performance</Link>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, margin: '8px 0 4px' }}>
            <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>{target.name}</h1>
            {target.last_status !== 'up' && target.last_status !== 'unknown' && (
              <span style={{
                color: colors.yellow, background: colors.yellowDim,
                padding: '4px 10px', borderRadius: 999, fontSize: 11, fontWeight: 600,
              }}>
                Slow
              </span>
            )}
          </div>
          <div style={{ color: colors.textMuted, fontSize: 14 }}>{target.url}</div>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <select value={period} onChange={e => setPeriod(e.target.value)} className="input" style={{ width: 'auto', padding: '8px 14px' }}>
            <option value="24h">Last 24 hours</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
          </select>
          {isAdmin && <Link to={`/performance/targets/${id}/edit`} className="btn">Edit</Link>}
        </div>
      </div>

      {perf && perf.p50_ms > 0 && (
        <div className="grid-4" style={{ marginBottom: 24 }}>
          <MetricCard label="Avg" value={`${perf.avg_ms} ms`} accent="blue" />
          <MetricCard label="P50" value={`${perf.p50_ms} ms`} />
          <MetricCard label="P95" value={`${perf.p95_ms} ms`} accent={perf.p95_ms > target.slow_threshold_ms ? 'yellow' : 'default'} />
          <MetricCard label="P99" value={`${perf.p99_ms} ms`} />
        </div>
      )}

      <div style={styles.chartCard}>
        <h3 style={{ margin: '0 0 16px', fontSize: 16, fontWeight: 600 }}>Response Time</h3>
        <div style={{ height: 300 }}>
          {chartData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="svcGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={colors.brand} stopOpacity={0.4} />
                    <stop offset="100%" stopColor={colors.brand} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid stroke={colors.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="time" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                <YAxis unit="ms" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                <Tooltip contentStyle={{ background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 8, color: colors.text }} />
                <ReferenceLine y={target.slow_threshold_ms} stroke={colors.yellow} strokeDasharray="4 4" label={{ value: 'SLA', fill: colors.yellow, fontSize: 11 }} />
                <Area type="monotone" dataKey="ms" stroke={colors.brand} fill="url(#svcGrad)" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div style={styles.empty}>Collecting latency data…</div>
          )}
        </div>
      </div>

      {chartData.some(p => p.dns > 0 || p.ttfb > 0) && (
        <div style={styles.chartCard}>
          <h3 style={{ margin: '0 0 16px', fontSize: 16, fontWeight: 600 }}>Timing Breakdown</h3>
          <div style={{ height: 280 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <CartesianGrid stroke={colors.border} strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="time" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                <YAxis unit="ms" tick={{ fill: colors.textMuted, fontSize: 11 }} axisLine={false} tickLine={false} />
                <Tooltip contentStyle={{ background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 8, color: colors.text }} />
                <Legend wrapperStyle={{ fontSize: 12, color: colors.textMuted }} />
                <Area type="monotone" dataKey="dns" stackId="1" stroke="#58a6ff" fill="#58a6ff" fillOpacity={0.6} name="DNS" />
                <Area type="monotone" dataKey="tcp" stackId="1" stroke="#bc8cff" fill="#bc8cff" fillOpacity={0.6} name="TCP" />
                <Area type="monotone" dataKey="tls" stackId="1" stroke="#f778ba" fill="#f778ba" fillOpacity={0.6} name="TLS" />
                <Area type="monotone" dataKey="ttfb" stackId="1" stroke={colors.brand} fill={colors.brand} fillOpacity={0.6} name="TTFB" />
                <Area type="monotone" dataKey="download" stackId="1" stroke={colors.yellow} fill={colors.yellow} fillOpacity={0.5} name="Download" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      <div style={styles.chartCard}>
        <h3 style={{ margin: '0 0 16px', fontSize: 16, fontWeight: 600 }}>Latency Log</h3>
        {results.length === 0 ? (
          <div style={styles.emptyLog}>Waiting for first probe…</div>
        ) : (
          <table style={styles.table}>
            <thead>
              <tr>
                <th style={styles.th}>Time</th>
                <th style={styles.th}>Status</th>
                <th style={styles.th}>Total</th>
                <th style={styles.th}>TTFB</th>
                <th style={styles.th}>DNS</th>
              </tr>
            </thead>
            <tbody>
              {results.map(r => (
                <tr key={r.id}>
                  <td style={styles.td}>{new Date(r.checked_at).toLocaleString()}</td>
                  <td style={{
                    ...styles.td,
                    color: (r.status === 'degraded' || r.status === 'down') ? colors.yellow : colors.green,
                    fontWeight: 600,
                  }}>
                    {r.status === 'degraded' || r.status === 'down' ? 'Slow' : 'OK'}
                  </td>
                  <td style={{ ...styles.td, color: (r.status === 'degraded' || r.status === 'down') ? colors.yellow : colors.text }}>{r.response_time_ms} ms</td>
                  <td style={styles.td}>{r.ttfb_ms != null ? `${r.ttfb_ms} ms` : '—'}</td>
                  <td style={styles.td}>{r.dns_ms != null ? `${r.dns_ms} ms` : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  header: {
    display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
    marginBottom: 24, paddingBottom: 20, borderBottom: `1px solid ${colors.border}`,
  },
  back: { color: colors.textMuted, fontSize: 13, textDecoration: 'none' },
  chartCard: {
    background: colors.card, border: `1px solid ${colors.border}`,
    borderRadius: 12, padding: '20px 24px', marginBottom: 20,
  },
  empty: { color: colors.textMuted, textAlign: 'center', paddingTop: 120 },
  emptyLog: { color: colors.textMuted, textAlign: 'center', padding: '24px 0', fontSize: 14 },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 13 },
  th: {
    textAlign: 'left', padding: '10px 12px', borderBottom: `1px solid ${colors.border}`,
    color: colors.textMuted, fontWeight: 600, fontSize: 12, textTransform: 'uppercase',
  },
  td: { padding: '12px', borderBottom: `1px solid ${colors.border}` },
}
