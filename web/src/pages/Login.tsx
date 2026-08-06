import { FormEvent, useState } from 'react'
import AppLogo from '../components/AppLogo'
import { api } from '../api'
import { colors } from '../theme'

type Mode = 'login' | 'forgot'

export default function Login({ onLogin }: { onLogin: () => void }) {
  const [mode, setMode] = useState<Mode>('login')
  const [username, setUsername] = useState('admin')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [sent, setSent] = useState(false)

  async function handleLogin(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')
    try {
      await api.login(username, password)
      onLogin()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    }
  }

  async function handleForgot(e: FormEvent) {
    e.preventDefault()
    if (sent) return
    setError('')
    setSent(true)
    try {
      await api.forgotPassword(email.trim())
      setMessage('Password reset link sent to your email.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed')
      setSent(false)
    }
  }

  return (
    <div style={styles.wrap}>
      <div style={styles.left}>
        <AppLogo size={48} />
        <p style={styles.intro}>INTRODUCING</p>
        <h1 style={styles.hero}>Sentinel</h1>
        <p style={styles.tagline}>Self-hosted infrastructure monitoring for websites, ports, SSL, and DNS.</p>
      </div>
      <form onSubmit={mode === 'login' ? handleLogin : handleForgot} style={styles.card}>
        <h2 style={{ margin: '0 0 4px', fontSize: 20 }}>
          {mode === 'login' ? 'Sign in' : 'Reset password'}
        </h2>
        <p style={{ color: colors.textMuted, margin: '0 0 24px', fontSize: 14 }}>
          {mode === 'login'
            ? 'Access your monitoring dashboard'
            : 'Enter your email address and we will send you a reset link.'}
        </p>
        {error && <div style={styles.error}>{error}</div>}
        {mode === 'login' ? (
          <>
            <label className="field">
              <span className="field-label">Username</span>
              <input className="input" value={username} onChange={e => setUsername(e.target.value)} required />
            </label>
            <label className="field" style={{ marginTop: 16 }}>
              <span className="field-label">Password</span>
              <input className="input" type="password" value={password} onChange={e => setPassword(e.target.value)} required />
            </label>
            <button type="submit" className="btn btn-primary" style={{ marginTop: 24, width: '100%', justifyContent: 'center', padding: 12 }}>
              Sign in
            </button>
          </>
        ) : message ? (
          <div style={styles.ok}>{message}</div>
        ) : (
          <>
            <label className="field">
              <span className="field-label">Email</span>
              <input className="input" type="email" value={email} onChange={e => setEmail(e.target.value)} required autoFocus />
            </label>
            <button type="submit" className="btn btn-primary" disabled={sent} style={{ marginTop: 24, width: '100%', justifyContent: 'center', padding: 12 }}>
              Send reset link
            </button>
          </>
        )}
        <button
          type="button"
          className="btn"
          style={{ marginTop: 12, width: '100%', justifyContent: 'center', fontSize: 13 }}
          onClick={() => {
            setMode(mode === 'login' ? 'forgot' : 'login')
            setError('')
            setMessage('')
            setSent(false)
          }}
        >
          {mode === 'login' ? 'Forgot password?' : 'Back to sign in'}
        </button>
      </form>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  wrap: {
    minHeight: '100vh', display: 'grid', gridTemplateColumns: '1fr 400px',
    background: colors.bg,
  },
  left: {
    display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: 8,
    padding: '48px 64px', background: `linear-gradient(135deg, ${colors.bg} 0%, #0a2e2c 100%)`,
    borderRight: `1px solid ${colors.border}`,
  },
  intro: { margin: '16px 0 0', fontSize: 12, fontWeight: 700, letterSpacing: '0.2em', color: colors.brand },
  hero: { margin: '8px 0', fontSize: 48, fontWeight: 800, color: colors.text },
  tagline: { color: colors.textMuted, fontSize: 16, lineHeight: 1.6, maxWidth: 400 },
  card: {
    display: 'flex', flexDirection: 'column', justifyContent: 'center',
    padding: '48px 40px', background: colors.card,
  },
  ok: {
    background: colors.greenDim, color: colors.green, padding: 10,
    borderRadius: 8, marginBottom: 16, fontSize: 14,
    border: `1px solid rgba(63,185,80,0.3)`,
  },
  error: {
    background: colors.redDim, color: colors.red, padding: 10,
    borderRadius: 8, marginBottom: 16, fontSize: 14,
    border: `1px solid rgba(248,81,73,0.3)`,
  },
}
