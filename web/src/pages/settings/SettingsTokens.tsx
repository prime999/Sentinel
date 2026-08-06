import { FormEvent, useEffect, useState } from 'react'
import { api, APIToken, APITokenCreated } from '../../api'
import { colors } from '../../theme'

export default function SettingsTokens() {
  const [tokens, setTokens] = useState<APIToken[]>([])
  const [name, setName] = useState('')
  const [created, setCreated] = useState<APITokenCreated | null>(null)
  const [error, setError] = useState('')

  async function load() {
    setTokens(await api.listTokens())
  }

  useEffect(() => { load().catch(() => {}) }, [])

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    setError('')
    setCreated(null)
    try {
      const t = await api.createToken(name)
      setCreated(t)
      setName('')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Revoke this API token?')) return
    await api.deleteToken(id)
    await load()
  }

  return (
    <>
      {error && <div style={styles.error}>{error}</div>}
      {created && (
        <div style={styles.tokenBox}>
          <strong>Token created — copy now, it won&apos;t be shown again:</strong>
          <code style={styles.token}>{created.token}</code>
        </div>
      )}
      <form onSubmit={handleCreate} style={styles.card}>
        <h3 style={styles.title}>API Tokens</h3>
        <p style={styles.desc}>Use Bearer tokens for programmatic API access.</p>
        <div style={{ display: 'flex', gap: 12 }}>
          <input required className="input" placeholder="Token name" value={name} onChange={e => setName(e.target.value)} style={{ flex: 1 }} />
          <button type="submit" className="btn btn-primary">Create token</button>
        </div>
      </form>
      <div style={{ ...styles.card, marginTop: 24 }}>
        {tokens.length === 0 ? (
          <p style={styles.desc}>No API tokens yet.</p>
        ) : (
          <table style={styles.table}>
            <thead><tr><th>Name</th><th>Prefix</th><th>Created</th><th>Last used</th><th></th></tr></thead>
            <tbody>
              {tokens.map(t => (
                <tr key={t.id}>
                  <td>{t.name}</td>
                  <td><code>{t.prefix}…</code></td>
                  <td>{new Date(t.created_at).toLocaleString()}</td>
                  <td>{t.last_used_at ? new Date(t.last_used_at).toLocaleString() : '—'}</td>
                  <td><button type="button" className="btn" onClick={() => handleDelete(t.id)}>Revoke</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  )
}

const styles: Record<string, React.CSSProperties> = {
  card: { background: colors.card, border: `1px solid ${colors.border}`, borderRadius: 12, padding: 28 },
  title: { margin: '0 0 8px' },
  desc: { color: colors.textMuted, fontSize: 14, margin: '0 0 20px' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 14 },
  tokenBox: { background: colors.bgElevated, border: `1px solid ${colors.border}`, borderRadius: 8, padding: 16, marginBottom: 16 },
  token: { display: 'block', marginTop: 8, wordBreak: 'break-all', fontSize: 13 },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
}
