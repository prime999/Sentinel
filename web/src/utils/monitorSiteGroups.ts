import type { Monitor, Site } from '../api'
import { isPaused } from '../components/StatusBadge'

export type SiteGroup = {
  site: Site
  monitors: Monitor[]
}

const TYPE_ORDER: Record<string, number> = { http: 0, ssl: 1, dns: 2, port: 3, heartbeat: 4 }

export function sortSiteMonitors(monitors: Monitor[]): Monitor[] {
  return [...monitors].sort((a, b) => (TYPE_ORDER[a.type] ?? 9) - (TYPE_ORDER[b.type] ?? 9))
}

export function groupMonitorsBySite(monitors: Monitor[], sites: Site[]): {
  groups: SiteGroup[]
  standalone: Monitor[]
} {
  const siteMap = new Map(sites.map(s => [s.id, s]))
  const bySite = new Map<string, Monitor[]>()
  const standalone: Monitor[] = []

  for (const m of monitors) {
    const sid = m.site_id?.trim()
    if (sid && siteMap.has(sid)) {
      const list = bySite.get(sid) || []
      list.push(m)
      bySite.set(sid, list)
    } else {
      standalone.push(m)
    }
  }

  const groups: SiteGroup[] = []
  for (const [id, mons] of bySite) {
    const site = siteMap.get(id)
    if (!site) continue
    groups.push({ site, monitors: sortSiteMonitors(mons) })
  }
  groups.sort((a, b) => a.site.name.localeCompare(b.site.name, undefined, { sensitivity: 'base' }))
  return { groups, standalone }
}

export function aggregateSiteStatus(monitors: Monitor[]): 'up' | 'down' | 'degraded' | 'unknown' | 'paused' {
  if (monitors.length === 0) return 'unknown'
  if (monitors.every(m => isPaused(m))) return 'paused'
  const active = monitors.filter(m => !isPaused(m))
  if (active.length === 0) return 'paused'
  if (active.some(m => m.last_status === 'down')) return 'down'
  if (active.some(m => m.last_status === 'degraded')) return 'degraded'
  if (active.every(m => m.last_status === 'up')) return 'up'
  return 'unknown'
}

export function primarySiteMonitor(monitors: Monitor[]): Monitor | undefined {
  return monitors.find(m => m.type === 'http') || monitors[0]
}

export function checkBadgeLabel(type: string): string {
  switch (type) {
    case 'http':
      return 'HTTPS'
    case 'ssl':
      return 'SSL'
    case 'dns':
      return 'DNS'
    case 'port':
      return 'PORT'
    default:
      return type.toUpperCase()
  }
}

export function childCheckLabel(m: Monitor): string {
  switch (m.type) {
    case 'http':
      return 'HTTPS'
    case 'ssl':
      return 'SSL Certificate'
    case 'dns':
      return 'DNS'
    case 'port':
      return `Port ${m.port ?? ''}`.trim()
    case 'heartbeat':
      return 'Heartbeat'
    default:
      return m.type
  }
}

export function healthyCheckCounts(monitors: Monitor[]): { up: number; total: number } {
  const active = monitors.filter(m => !isPaused(m))
  const up = active.filter(m => m.last_status === 'up').length
  return { up, total: active.length }
}
