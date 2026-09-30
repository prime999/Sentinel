import { ChangeEvent, CSSProperties, useEffect, useRef, useState } from 'react'
import {
  api,
  BackupSectionPreview,
  PlatformBackupPreview,
} from '../../api'
import ModalCloseButton from '../../components/ModalCloseButton'
import { useSaveToast } from '../../components/useSaveToast'
import { useAuth } from '../../context/AuthContext'
import { colors } from '../../theme'

type ImportMode = 'create_only' | 'overwrite'

function hasBackupData(parsed: Record<string, unknown>): boolean {
  const arrays = ['monitors', 'customers', 'users', 'performance_targets', 'hosts', 'maintenance_windows']
  return arrays.some(k => Array.isArray(parsed[k]) && (parsed[k] as unknown[]).length > 0)
}

function sectionSummary(label: string, sec?: BackupSectionPreview): string {
  if (!sec || sec.total === 0) return ''
  const conflicts = sec.conflicts?.length ?? 0
  return `${label}: ${sec.to_create} new, ${conflicts} conflicts`
}

function allItemsAreSkipsInCreateOnly(preview: PlatformBackupPreview): boolean {
  const sections = [
    preview.customers,
    preview.users,
    preview.monitors,
    preview.performance_targets,
    preview.hosts,
    preview.maintenance_windows,
  ]
  const hasItems = sections.some(s => (s?.total ?? 0) > 0)
  const hasCreates = sections.some(s => (s?.to_create ?? 0) > 0)
  return hasItems && !hasCreates
}

function previewHasItems(preview: PlatformBackupPreview): boolean {
  const sections = [
    preview.customers,
    preview.users,
    preview.monitors,
    preview.performance_targets,
    preview.hosts,
    preview.maintenance_windows,
  ]
  return sections.some(s => (s?.total ?? 0) > 0) || preview.settings_included
}

export default function SettingsBackup() {
  const { isPlatformAdmin } = useAuth()
  const [customers, setCustomers] = useState<{ id: string; name: string }[]>([])
  const [exportCustomer, setExportCustomer] = useState('')
  const [exportBusy, setExportBusy] = useState(false)
  const [importPayload, setImportPayload] = useState<Record<string, unknown> | null>(null)
  const [preview, setPreview] = useState<PlatformBackupPreview | null>(null)
  const [importModalOpen, setImportModalOpen] = useState(false)
  const [importMode, setImportMode] = useState<ImportMode>('create_only')
  const [importBusy, setImportBusy] = useState(false)
  const { toast, showSaved } = useSaveToast()
  const [error, setError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!isPlatformAdmin) return
    api.listCustomers()
      .then(c => setCustomers(c.map(x => ({ id: x.id, name: x.name }))))
      .catch(() => {})
  }, [isPlatformAdmin])

  useEffect(() => {
    if (!importModalOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !importBusy) closeImportModal()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [importModalOpen, importBusy])

  function closeImportModal() {
    if (importBusy) return
    setImportModalOpen(false)
    setImportPayload(null)
    setPreview(null)
    setImportMode('create_only')
  }

  async function handleExport() {
    setExportBusy(true)
    setError('')
    try {
      await api.exportMonitorBackup(exportCustomer || undefined)
      showSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed')
    } finally {
      setExportBusy(false)
    }
  }

  async function loadPreview(payload: Record<string, unknown>) {
    setImportBusy(true)
    setError('')
    try {
      const p = await api.previewMonitorBackup(payload)
      setPreview(p)
      setImportModalOpen(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Preview failed')
      setPreview(null)
      setImportPayload(null)
    } finally {
      setImportBusy(false)
    }
  }

  async function onFileChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setError('')
    try {
      const parsed = JSON.parse(await file.text()) as Record<string, unknown>
      if (!hasBackupData(parsed)) {
        setError('No backup data found in this file')
        return
      }
      setImportPayload(parsed)
      await loadPreview(parsed)
    } catch {
      setError('Could not read backup file (invalid JSON)')
    }
  }

  async function handleImport() {
    if (!importPayload || !preview) return
    setImportBusy(true)
    setError('')
    try {
      await api.importMonitorBackup({ ...importPayload, mode: importMode })
      showSaved()
      setImportModalOpen(false)
      setImportPayload(null)
      setPreview(null)
      setImportMode('create_only')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed')
    } finally {
      setImportBusy(false)
    }
  }

  const sectionLines = preview
    ? [
      sectionSummary('Customers', preview.customers),
      sectionSummary('Users', preview.users),
      sectionSummary('Monitors', preview.monitors),
      sectionSummary('Performance', preview.performance_targets),
      sectionSummary('Hosts', preview.hosts),
      sectionSummary('Maintenance', preview.maintenance_windows),
    ].filter(Boolean)
    : []

  const monitorConflicts = preview?.monitors?.conflicts ?? []
  const missingDeps = preview?.missing_dependencies ?? []
  const canRestore = preview != null && previewHasItems(preview)

  return (
    <div style={styles.card}>
      <div style={styles.header}>
        <div>
          <h3 style={styles.title}>Platform backup</h3>
          <p style={styles.desc}>
            Export customers, users, monitors, performance targets, hosts, maintenance windows, and settings
            for disaster recovery. Check history, incidents, email/audit logs, and API tokens are not included.
          </p>
        </div>
      </div>

      {error && <div style={styles.error}>{error}</div>}
      {toast}

      <section style={styles.section}>
        <h4 style={styles.sectionTitle}>Export</h4>
        <p style={styles.sectionDesc}>
          Downloads a JSON snapshot including secrets (passwords, SMTP, webhooks, Slack, HTTP auth, tokens).
        </p>
        <div style={styles.row}>
          {isPlatformAdmin && (
            <label style={styles.field}>
              <span style={styles.label}>Customer</span>
              <select
                className="input"
                style={styles.select}
                value={exportCustomer}
                onChange={e => setExportCustomer(e.target.value)}
              >
                <option value="">Full platform</option>
                {customers.map(c => (
                  <option key={c.id} value={c.id}>{c.name} (tenant pack)</option>
                ))}
              </select>
            </label>
          )}
          <button type="button" className="btn btn-primary" disabled={exportBusy} onClick={handleExport}>
            {exportBusy ? 'Exporting…' : 'Download backup'}
          </button>
        </div>
      </section>

      <section style={styles.section}>
        <h4 style={styles.sectionTitle}>Import</h4>
        <p style={styles.sectionDesc}>
          Choose a backup file to review and restore in a dialog. Store backup files securely and recreate API tokens after restore.
        </p>
        <div style={styles.row}>
          <input ref={fileRef} type="file" accept=".json,application/json" style={{ display: 'none' }} onChange={onFileChange} />
          <button type="button" className="btn btn-primary" disabled={importBusy} onClick={() => fileRef.current?.click()}>
            {importBusy ? 'Loading…' : 'Import from backup…'}
          </button>
        </div>
      </section>

      {importModalOpen && preview && (
        <div
          style={styles.backdrop}
          role="presentation"
          onMouseDown={e => {
            if (e.target === e.currentTarget && !importBusy) closeImportModal()
          }}
        >
          <div style={styles.dialog} role="dialog" aria-modal="true" aria-labelledby="import-backup-title">
            <div style={styles.dialogHead}>
              <h3 id="import-backup-title" style={styles.dialogTitle}>Restore from backup</h3>
              <ModalCloseButton onClick={closeImportModal} disabled={importBusy} />
            </div>
            <p style={styles.dialogDesc}>
              Review what will happen, then restore. With &quot;Add missing only&quot;, existing ids are left unchanged (skipped).
            </p>

            {missingDeps.length > 0 && (
              <div style={styles.warn}>
                Missing customers in target system:
                <ul style={styles.invalidList}>
                  {missingDeps.map((d, i) => (
                    <li key={i}>{d}</li>
                  ))}
                </ul>
              </div>
            )}

            <ul style={styles.sectionList}>
              {sectionLines.map(line => (
                <li key={line}>{line}</li>
              ))}
            </ul>
            {preview.settings_included && (
              <p style={styles.previewSummary}>Settings bundle included in file.</p>
            )}
            {(preview.monitors?.quota_blocked ?? 0) > 0 && (
              <p style={styles.previewSummary}>
                {preview.monitors?.quota_blocked} monitor(s) may exceed customer quota.
              </p>
            )}

            {monitorConflicts.length > 0 && (
              <>
                <p style={styles.label}>Monitor conflicts (sample)</p>
                <div style={styles.tableWrap}>
                  <table style={styles.table}>
                    <thead>
                      <tr>
                        <th style={styles.th}>Backup name</th>
                        <th style={styles.th}>Existing name</th>
                      </tr>
                    </thead>
                    <tbody>
                      {monitorConflicts.slice(0, 8).map(c => (
                        <tr key={c.id}>
                          <td style={styles.td}>{c.import_name}</td>
                          <td style={styles.td}>{c.existing_name}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}

            <fieldset style={styles.modeFieldset}>
              <legend style={styles.label}>When an id already exists</legend>
              <label style={styles.radio}>
                <input
                  type="radio"
                  name="import-mode"
                  checked={importMode === 'create_only'}
                  onChange={() => setImportMode('create_only')}
                />
                Add missing only (skip conflicts)
              </label>
              <label style={styles.radio}>
                <input
                  type="radio"
                  name="import-mode"
                  checked={importMode === 'overwrite'}
                  onChange={() => setImportMode('overwrite')}
                />
                Overwrite conflicts with backup
              </label>
            </fieldset>

            {importMode === 'create_only' && allItemsAreSkipsInCreateOnly(preview) && (
              <p style={styles.hint}>
                Everything in this file already exists. Restore will skip conflicts unless you choose overwrite.
              </p>
            )}

            <div style={styles.dialogActions}>
              <button type="button" className="btn" disabled={importBusy} onClick={closeImportModal}>
                Cancel
              </button>
              <button
                type="button"
                className="btn btn-primary"
                disabled={importBusy || !canRestore}
                onClick={handleImport}
              >
                {importBusy ? 'Restoring…' : 'Restore from backup'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

const styles: Record<string, CSSProperties> = {
  card: {
    background: colors.bgElevated,
    border: `1px solid ${colors.border}`,
    borderRadius: 12,
    padding: '24px 28px',
    maxWidth: 900,
  },
  header: { marginBottom: 20 },
  title: { margin: 0, fontSize: 20, fontWeight: 600, color: colors.text },
  desc: { margin: '8px 0 0', fontSize: 15, color: colors.textMuted, lineHeight: 1.5 },
  section: {
    marginTop: 28,
    paddingTop: 24,
    borderTop: `1px solid ${colors.border}`,
  },
  sectionTitle: { margin: '0 0 8px', fontSize: 17, fontWeight: 600 },
  sectionDesc: { margin: '0 0 16px', fontSize: 15, color: colors.textMuted, lineHeight: 1.5 },
  row: { display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 12 },
  field: { display: 'flex', alignItems: 'center', gap: 8, fontSize: 15, color: colors.textMuted },
  label: { fontWeight: 500, flexShrink: 0 },
  select: { width: 'auto', minWidth: 180, padding: '0 12px' },
  error: { background: colors.redDim, color: colors.red, padding: 12, borderRadius: 8, marginBottom: 16 },
  warn: { background: colors.yellowDim, color: colors.text, padding: 12, borderRadius: 8, marginBottom: 12, fontSize: 14 },
  backdrop: {
    position: 'fixed',
    inset: 0,
    zIndex: 1000,
    background: 'var(--overlay)',
    display: 'grid',
    placeItems: 'center',
    padding: 24,
  },
  dialog: {
    width: '100%',
    maxWidth: 560,
    maxHeight: 'min(90vh, 720px)',
    overflow: 'auto',
    background: colors.card,
    border: `1px solid ${colors.border}`,
    borderRadius: 10,
    padding: '24px 28px',
    boxShadow: 'var(--shadow)',
  },
  dialogHead: {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: 12,
    marginBottom: 8,
  },
  dialogTitle: { margin: 0, fontSize: 19, fontWeight: 700 },
  dialogDesc: { margin: '0 0 16px', fontSize: 15, color: colors.textMuted, lineHeight: 1.5 },
  dialogActions: {
    display: 'flex',
    justifyContent: 'flex-end',
    flexWrap: 'wrap',
    gap: 10,
    marginTop: 20,
    paddingTop: 16,
    borderTop: `1px solid ${colors.border}`,
  },
  previewSummary: { margin: '0 0 12px', fontSize: 15, color: colors.text },
  hint: { margin: '0 0 12px', fontSize: 14, color: colors.textMuted, lineHeight: 1.45 },
  sectionList: { margin: '0 0 16px', paddingLeft: 20, fontSize: 15, color: colors.text },
  modeFieldset: { border: 'none', margin: '16px 0', padding: 0 },
  radio: { display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, fontSize: 15, cursor: 'pointer' },
  tableWrap: { overflowX: 'auto', marginBottom: 12 },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: 14 },
  th: {
    textAlign: 'left',
    padding: '8px 10px',
    borderBottom: `1px solid ${colors.border}`,
    color: colors.textMuted,
    fontWeight: 600,
    fontSize: 13,
  },
  td: { padding: '8px 10px', borderBottom: `1px solid ${colors.border}` },
  invalidList: { margin: '8px 0 0', paddingLeft: 20, fontSize: 14, color: colors.textMuted },
}
