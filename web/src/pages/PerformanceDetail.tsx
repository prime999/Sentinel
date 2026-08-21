import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  Area, AreaChart, CartesianGrid, Legend, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'
import { api, PerformanceResult, PerformanceStats, PerformanceTarget } from '../api'
import { ColGroup, ResizableTh, useColumnResize } from '../components/ColumnResize'
import { useAuth } from '../context/AuthContext'
import DatePicker from '../components/DatePicker'
import MetricCard from '../components/MetricCard'
import NextCheckCountdown from '../components/NextCheckCountdown'
import { colors } from '../theme'
import { useAdaptivePoll } from '../utils/poll'

export default function PerformanceDetail() {
  const { isAdmin } = useAuth()
  const { id } = useParams<{ id: string }>()
  const [target, setTarget] = useState<PerformanceTarget | null>(null)
  const [stats, setStats] = useState<PerformanceStats | null>(null)
  const [period, setPeriod] = useState('24h')
  const periodRef = useRef(period)
  periodRef.current = period

  const load = useCallback(async () => {
    if (!id) return null
    const [t, s] = await Promise.all([
      api.getPerformanceTarget(id),
      api.performanceStats(id, periodRef.current),
    ])
    setTarget(t)
    setStats(s)
    return t
  }, [id])

  const refreshRef = useAdaptivePoll(id, load, [period])

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

      <NextCheckCountdown
        target={target}
        onDue={() => refreshRef.current?.()}
        disabledLabel="Paused"
      />

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

      {id && (
        <SLABreachLog targetId={id} slaMs={target.slow_threshold_ms} />
      )}
    </div>
  )
}

const PAGE_SIZE = 20

function SLABreachLog({ targetId, slaMs }: { targetId: string; slaMs: number }) {
  const [page, setPage] = useState(0)
  const [date, setDate] = useState('')
  const [items, setItems] = useState<PerformanceResult[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const tableRef = useRef<HTMLTableElement>(null)
  const { widths, startResize, autoFit } = useColumnResize('sla-log', 6)

  useEffect(() => {
    setPage(0)
  }, [targetId, date])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api.performanceResults(targetId, {
      limit: PAGE_SIZE,
      offset: page * PAGE_SIZE,
      date: date || undefined,
      breaches: true,
    })
      .then(res => {
        if (cancelled) return
        setItems(res.items)
        setTotal(res.total)
      })
      .catch(() => {
        if (!cancelled) {
          setItems([])
          setTotal(0)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [targetId, page, date])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const from = total === 0 ? 0 : page * PAGE_SIZE + 1
  const to = Math.min(total, (page + 1) * PAGE_SIZE)

  return (
    <div style={styles.chartCard}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16, gap: 12, flexWrap: 'wrap' }}>
        <div>
          <h3 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>SLA Breaches</h3>
          <div style={{ marginTop: 4, fontSize: 13, color: colors.textMuted }}>
            Probes slower than {slaMs > 0 ? `${slaMs} ms` : 'SLA'}
          </div>
        </div>
        <span style={{ fontSize: 13, color: colors.textMuted }}>
          {total === 0
            ? (date ? 'No breaches on this day' : 'No SLA breaches yet')
            : `Showing ${from}–${to} of ${total}`}
        </span>
      </div>

      <div style={styles.filterBar}>
        <DatePicker value={date} onChange={setDate} />
        {date && (
          <button type="button" className="btn" style={styles.resetBtn} onClick={() => setDate('')}>
            Clear date
          </button>
        )}
      </div>

      <div className="data-table-wrap">
        <table ref={tableRef} className="data-table" style={styles.table}>
          <ColGroup widths={widths} />
          <thead>
            <tr>
              <ResizableTh index={0} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Time</ResizableTh>
              <ResizableTh index={1} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Status</ResizableTh>
              <ResizableTh index={2} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Total</ResizableTh>
              <ResizableTh index={3} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>Over SLA</ResizableTh>
              <ResizableTh index={4} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>TTFB</ResizableTh>
              <ResizableTh index={5} style={styles.th} startResize={startResize} autoFit={autoFit} tableRef={tableRef}>DNS</ResizableTh>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} style={{ ...styles.td, color: colors.textMuted }}>Loading…</td>
              </tr>
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={6} style={{ ...styles.td, color: colors.textMuted }}>
                  {date ? 'No SLA breaches for this date.' : 'No probes have exceeded the SLA yet.'}
                </td>
              </tr>
            ) : (
              items.map(r => {
                const over = Math.max(0, r.response_time_ms - slaMs)
                return (
                  <tr key={r.id}>
                    <td style={styles.td}>{new Date(r.checked_at).toLocaleString()}</td>
                    <td style={{ ...styles.td, color: colors.yellow, fontWeight: 600 }}>Slow</td>
                    <td style={{ ...styles.td, color: colors.yellow }}>{r.response_time_ms} ms</td>
                    <td style={{ ...styles.td, color: colors.yellow }}>+{over} ms</td>
                    <td style={styles.td}>{r.ttfb_ms != null ? `${r.ttfb_ms} ms` : '—'}</td>
                    <td style={styles.td}>{r.dns_ms != null ? `${r.dns_ms} ms` : '—'}</td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>

      {total > PAGE_SIZE && (
        <div style={styles.pager}>
          <button
            type="button"
            className="btn"
            style={styles.pagerBtn}
            disabled={page <= 0 || loading}
            onClick={() => setPage(p => Math.max(0, p - 1))}
          >
            Previous
          </button>
          <span style={{ fontSize: 13, color: colors.textMuted }}>
            Page {page + 1} of {totalPages}
          </span>
          <button
            type="button"
            className="btn"
            style={styles.pagerBtn}
            disabled={page + 1 >= totalPages || loading}
            onClick={() => setPage(p => p + 1)}
          >
            Next
          </button>
        </div>
      )}
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
  filterBar: {
    display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap', marginBottom: 16,
  },
  resetBtn: { padding: '8px 12px', fontSize: 13 },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 13 },
  th: {
    textAlign: 'left', padding: '10px 12px', borderBottom: `1px solid ${colors.border}`,
    color: colors.textMuted, fontWeight: 600, fontSize: 12, textTransform: 'uppercase',
  },
  td: { padding: '12px', borderBottom: `1px solid ${colors.border}` },
  pager: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: 12,
    marginTop: 16,
    paddingTop: 12,
    borderTop: `1px solid ${colors.border}`,
  },
  pagerBtn: {
    minHeight: 36,
    padding: '0 14px',
    fontSize: 13,
  },
}
