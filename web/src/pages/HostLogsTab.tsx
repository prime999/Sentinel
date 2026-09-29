import { FormEvent, useCallback, useEffect, useState } from 'react'
import {
  api, Host, LogAlertRule, LogEvent, LogSource, LogSourceTemplate, LogVolumeBucket,
} from '../api'
import Panel from '../components/Panel'
import { colors, fonts } from '../theme'

type Props = { host: Host; isAdmin: boolean }

export default function HostLogsTab({ host, isAdmin }: Props) {
  const [sources, setSources] = useState<LogSource[]>([])
  const [rules, setRules] = useState<LogAlertRule[]>([])
  const [events, setEvents] = useState<LogEvent[]>([])
  const [volume, setVolume] = useState<LogVolumeBucket[]>([])
  const [templates, setTemplates] = useState<LogSourceTemplate[]>([])
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [level, setLevel] = useState('')
  const [sourceFilter, setSourceFilter] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const [showRule, setShowRule] = useState(false)
  const [tailLines, setTailLines] = useState<LogEvent[]>([])
  const [tailing, setTailing] = useState(false)

  const [form, setForm] = useState({
    name: '', path: '', type: 'file' as 'file' | 'journal',
    min_level: 'WARN', include_patterns: 'ERROR,CRITICAL,Exception,timeout',
  })
  const [ruleForm, setRuleForm] = useState({
    name: '', pattern: 'ERROR|CRITICAL|Exception', threshold: 5, window_seconds: 300,
    severity: 'warning', source_id: '', notify_email: true, notify_slack: true, notify_webhooks: false,
  })

  const refresh = useCallback(() => {
    api.listLogSources(host.id).then(setSources).catch(() => {})
    api.listLogAlertRules(host.id).then(setRules).catch(() => {})
    const from = new Date(Date.now() - 24 * 3600 * 1000).toISOString()
    api.queryLogVolume(host.id, { from, source_id: sourceFilter || undefined }).then(setVolume).catch(() => {})
    api.queryLogEvents(host.id, {
      q: search || undefined,
      level: level || undefined,
      source_id: sourceFilter || undefined,
      from,
      limit: 100,
    }).then(setEvents).catch(() => {})
  }, [host.id, search, level, sourceFilter])

  useEffect(() => { refresh() }, [refresh])
  useEffect(() => { api.logSourceTemplates().then(setTemplates).catch(() => {}) }, [])

  async function addSource(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const patterns = form.include_patterns.split(/[,|]/).map(s => s.trim()).filter(Boolean)
      await api.createLogSource(host.id, {
        name: form.name,
        path: form.path,
        type: form.type,
        min_level: form.min_level,
        include_patterns: patterns,
        enabled: true,
      })
      setShowAdd(false)
      setForm({ name: '', path: '', type: 'file', min_level: 'WARN', include_patterns: 'ERROR,CRITICAL,Exception,timeout' })
      refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add source')
    }
  }

  async function addRule(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.createLogAlertRule(host.id, {
        ...ruleForm,
        source_id: ruleForm.source_id || undefined,
        enabled: true,
      })
      setShowRule(false)
      refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add rule')
    }
  }

  function applyTemplate(t: LogSourceTemplate) {
    setForm({
      name: t.name,
      path: t.path,
      type: 'file',
      min_level: t.min_level || 'WARN',
      include_patterns: (t.include_patterns || []).join(','),
    })
    setShowAdd(true)
  }

  function startTail(sourceId: string) {
    setTailing(true)
    setTailLines([])
    const es = new EventSource(`/api/hosts/${host.id}/logs/sources/${sourceId}/tail`, { withCredentials: true })
    es.addEventListener('log', (ev) => {
      try {
        const e = JSON.parse((ev as MessageEvent).data) as LogEvent
        setTailLines(prev => [...prev.slice(-200), e])
      } catch { /* ignore */ }
    })
    es.onerror = () => {
      es.close()
      setTailing(false)
    }
    ;(window as unknown as { __logTail?: EventSource }).__logTail = es
  }

  function stopTail() {
    const es = (window as unknown as { __logTail?: EventSource }).__logTail
    es?.close()
    setTailing(false)
  }

  const volByMinute = summarizeVolume(volume)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {error && <div style={styles.error} role="alert">{error}</div>}

      <Panel>
        <div style={styles.rowBetween}>
          <h3 className="panel-title" style={{ margin: 0 }}>Log sources</h3>
          {isAdmin && (
            <button type="button" className="btn btn-primary" onClick={() => setShowAdd(v => !v)}>
              {showAdd ? 'Cancel' : 'Add source'}
            </button>
          )}
        </div>
        {isAdmin && templates.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 12 }}>
            {templates.map(t => (
              <button key={t.id} type="button" className="btn btn-ghost" onClick={() => applyTemplate(t)} style={{ fontSize: 13 }}>
                {t.name}
              </button>
            ))}
          </div>
        )}
        {showAdd && isAdmin && (
          <form onSubmit={addSource} style={{ display: 'grid', gap: 12, marginBottom: 16 }}>
            <div className="grid-2">
              <label className="field">
                <span className="field-label">Name</span>
                <input className="input" required value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              </label>
              <label className="field">
                <span className="field-label">Type</span>
                <select className="input" value={form.type} onChange={e => setForm(f => ({ ...f, type: e.target.value as 'file' | 'journal' }))}>
                  <option value="file">File</option>
                  <option value="journal">Journald unit</option>
                </select>
              </label>
            </div>
            <label className="field">
              <span className="field-label">{form.type === 'journal' ? 'Unit (e.g. nginx.service)' : 'Path (absolute, allowlisted)'}</span>
              <input className="input" required value={form.path} onChange={e => setForm(f => ({ ...f, path: e.target.value }))}
                placeholder={form.type === 'journal' ? 'nginx.service' : '/var/log/nginx/error.log'} />
            </label>
            <div className="grid-2">
              <label className="field">
                <span className="field-label">Minimum level</span>
                <select className="input" value={form.min_level} onChange={e => setForm(f => ({ ...f, min_level: e.target.value }))}>
                  {['DEBUG', 'INFO', 'WARN', 'ERROR', 'CRITICAL'].map(l => <option key={l} value={l}>{l}</option>)}
                </select>
              </label>
              <label className="field">
                <span className="field-label">Include patterns</span>
                <input className="input" value={form.include_patterns} onChange={e => setForm(f => ({ ...f, include_patterns: e.target.value }))} />
              </label>
            </div>
            <button type="submit" className="btn btn-primary">Save source</button>
          </form>
        )}
        {sources.length === 0 ? (
          <div style={styles.empty}>No log sources yet. Add a preset or custom path under /var/log or app storage/logs.</div>
        ) : (
          <table className="table" style={{ width: '100%' }}>
            <thead>
              <tr>
                <th>Name</th>
                <th>Path</th>
                <th>Health</th>
                <th>Min</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {sources.map(s => (
                <tr key={s.id}>
                  <td>{s.name}</td>
                  <td style={{ fontFamily: fonts.mono, fontSize: 13 }}>{s.path}</td>
                  <td>
                    <span style={{ color: healthColor(s.health) }}>{s.health}</span>
                    {s.health_detail ? <div style={styles.hint}>{s.health_detail}</div> : null}
                  </td>
                  <td>{s.min_level}</td>
                  <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                    <button type="button" className="btn btn-ghost" onClick={() => { setSourceFilter(s.id); startTail(s.id) }}>
                      Live tail
                    </button>
                    {isAdmin && (
                      <button type="button" className="btn btn-ghost" style={{ color: colors.red }}
                        onClick={() => api.deleteLogSource(host.id, s.id).then(refresh).catch(err => setError(String(err)))}>
                        Delete
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Panel>

      <Panel>
        <div style={styles.rowBetween}>
          <h3 className="panel-title" style={{ margin: 0 }}>Volume (24h)</h3>
        </div>
        {volByMinute.length === 0 ? (
          <div style={styles.empty}>No volume data yet — matched WARN+ lines appear here as 1-minute buckets.</div>
        ) : (
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: 2, height: 80, overflowX: 'auto' }}>
            {volByMinute.map(b => (
              <div key={b.t} title={`${b.t}: ${b.n}`}
                style={{ width: 6, height: Math.max(2, Math.min(80, b.n)), background: colors.blue || '#3b82f6', borderRadius: 1 }} />
            ))}
          </div>
        )}
      </Panel>

      <Panel>
        <div style={styles.rowBetween}>
          <h3 className="panel-title" style={{ margin: 0 }}>Events</h3>
          {tailing && <button type="button" className="btn btn-ghost" onClick={stopTail}>Stop live tail</button>}
        </div>
        <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
          <input className="input" placeholder="Search message" value={search} onChange={e => setSearch(e.target.value)} style={{ flex: 1, minWidth: 160 }} />
          <select className="input" value={level} onChange={e => setLevel(e.target.value)} style={{ width: 120 }}>
            <option value="">All levels</option>
            {['DEBUG', 'INFO', 'WARN', 'ERROR', 'CRITICAL'].map(l => <option key={l} value={l}>{l}</option>)}
          </select>
          <select className="input" value={sourceFilter} onChange={e => setSourceFilter(e.target.value)} style={{ minWidth: 160 }}>
            <option value="">All sources</option>
            {sources.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        </div>
        {(tailing ? tailLines : events).length === 0 ? (
          <div style={styles.empty}>No retained events match these filters.</div>
        ) : (
          <div style={{ maxHeight: 360, overflow: 'auto', fontFamily: fonts.mono, fontSize: 12, lineHeight: 1.45 }}>
            {(tailing ? tailLines : events).map(e => (
              <div key={e.id + e.timestamp} style={{ padding: '4px 0', borderBottom: `1px solid ${colors.border}` }}>
                <span style={{ color: colors.textMuted }}>{new Date(e.timestamp).toLocaleString()}</span>
                {' '}
                <span style={{ color: healthColor(e.level), fontWeight: 600 }}>{e.level}</span>
                {' '}
                <span>{e.message}</span>
              </div>
            ))}
          </div>
        )}
      </Panel>

      <Panel>
        <div style={styles.rowBetween}>
          <h3 className="panel-title" style={{ margin: 0 }}>Log alert rules</h3>
          {isAdmin && (
            <button type="button" className="btn btn-primary" onClick={() => setShowRule(v => !v)}>
              {showRule ? 'Cancel' : 'Add rule'}
            </button>
          )}
        </div>
        {showRule && isAdmin && (
          <form onSubmit={addRule} style={{ display: 'grid', gap: 12, marginBottom: 16 }}>
            <div className="grid-2">
              <label className="field">
                <span className="field-label">Name</span>
                <input className="input" required value={ruleForm.name} onChange={e => setRuleForm(f => ({ ...f, name: e.target.value }))} />
              </label>
              <label className="field">
                <span className="field-label">Source (optional)</span>
                <select className="input" value={ruleForm.source_id} onChange={e => setRuleForm(f => ({ ...f, source_id: e.target.value }))}>
                  <option value="">All sources</option>
                  {sources.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                </select>
              </label>
            </div>
            <label className="field">
              <span className="field-label">Pattern</span>
              <input className="input" required value={ruleForm.pattern} onChange={e => setRuleForm(f => ({ ...f, pattern: e.target.value }))} />
            </label>
            <div className="grid-2">
              <label className="field">
                <span className="field-label">Threshold</span>
                <input type="number" min={1} className="input" value={ruleForm.threshold}
                  onChange={e => setRuleForm(f => ({ ...f, threshold: +e.target.value || 1 }))} />
              </label>
              <label className="field">
                <span className="field-label">Window (seconds)</span>
                <input type="number" min={60} className="input" value={ruleForm.window_seconds}
                  onChange={e => setRuleForm(f => ({ ...f, window_seconds: +e.target.value || 300 }))} />
              </label>
            </div>
            <button type="submit" className="btn btn-primary">Save rule</button>
          </form>
        )}
        {rules.length === 0 ? (
          <div style={styles.empty}>No log alert rules. Pattern thresholds open host_log incidents when exceeded.</div>
        ) : (
          <table className="table" style={{ width: '100%' }}>
            <thead>
              <tr><th>Name</th><th>Pattern</th><th>Threshold</th><th>Window</th><th /></tr>
            </thead>
            <tbody>
              {rules.map(r => (
                <tr key={r.id}>
                  <td>{r.name}</td>
                  <td style={{ fontFamily: fonts.mono, fontSize: 13 }}>{r.pattern}</td>
                  <td>&gt; {r.threshold}</td>
                  <td>{Math.round(r.window_seconds / 60)}m</td>
                  <td style={{ textAlign: 'right' }}>
                    {isAdmin && (
                      <button type="button" className="btn btn-ghost" style={{ color: colors.red }}
                        onClick={() => api.deleteLogAlertRule(host.id, r.id).then(refresh).catch(err => setError(String(err)))}>
                        Delete
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Panel>
    </div>
  )
}

function summarizeVolume(buckets: LogVolumeBucket[]): { t: string; n: number }[] {
  const map = new Map<string, number>()
  for (const b of buckets) {
    const t = b.bucket_start
    map.set(t, (map.get(t) || 0) + b.count)
  }
  return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0])).map(([t, n]) => ({ t, n }))
}

function healthColor(h: string): string {
  const x = h.toLowerCase()
  if (x.includes('error') || x.includes('critical') || x.includes('denied') || x.includes('rejected') || x.includes('full')) return colors.red
  if (x.includes('warn') || x.includes('lag') || x.includes('rate') || x.includes('missing') || x.includes('rotat')) return colors.yellow || '#eab308'
  return colors.green
}

const styles: Record<string, React.CSSProperties> = {
  rowBetween: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12, gap: 12 },
  empty: { color: colors.textMuted, fontSize: 14, padding: '8px 0' },
  hint: { fontSize: 12, color: colors.textMuted },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8 },
}
