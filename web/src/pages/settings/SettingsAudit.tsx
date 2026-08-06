import { useEffect, useState } from 'react'
import { api, AuditEntry } from '../../api'
import { colors } from '../../theme'

export default function SettingsAudit() {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    api.listAudit().then(setEntries).catch(err => setError(err instanceof Error ? err.message : 'Failed to load'))
  }, [])

  return (
    <div style={styles.card}>
      <h3 style={styles.title}>Audit Log</h3>
      <p style={styles.desc}>Recent administrative actions.</p>
      {error && <div style={styles.error}>{error}</div>}
      {entries.length === 0 ? (
        <p style={styles.desc}>No audit entries yet.</p>
      ) : (
        <table style={styles.table}>
          <thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Resource</th><th>Detail</th></tr></thead>
          <tbody>
            {entries.map(e => (
              <tr key={e.id}>
                <td>{new Date(e.created_at).toLocaleString()}</td>
                <td>{e.actor}</td>
                <td>{e.action}</td>
                <td>{e.resource}</td>
                <td style={{ color: colors.textMuted }}>{e.detail || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12, padding: 28 },
  title: { margin: '0 0 8px' },
  desc: { color: colors.textMuted, fontSize: 14, margin: '0 0 20px' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 14 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
