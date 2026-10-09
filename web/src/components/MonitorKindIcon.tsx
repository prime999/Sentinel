import {
  Activity,
  Gauge,
  Globe,
  Layers3,
  Monitor,
  Network,
  Shield,
  type LucideIcon,
} from 'lucide-react'
import { colors } from '../theme'

export type MonitorKind = 'site' | 'http' | 'ssl' | 'dns' | 'port' | 'heartbeat' | 'standalone' | 'performance'

/** Per-type colors from the Lucide icon sheet. */
const kindMeta: Record<MonitorKind, { Icon: LucideIcon; color: string }> = {
  site: { Icon: Layers3, color: colors.brand },
  http: { Icon: Monitor, color: colors.blue },
  ssl: { Icon: Shield, color: '#60A5FA' },
  dns: { Icon: Globe, color: '#A78BFA' },
  port: { Icon: Network, color: '#FB923C' },
  heartbeat: { Icon: Activity, color: colors.green },
  standalone: { Icon: Activity, color: colors.green },
  performance: { Icon: Gauge, color: colors.red },
}

export function monitorKindFor(type?: string, standalone?: boolean): MonitorKind {
  if (standalone) return 'standalone'
  if (!type || type === 'http') return 'http'
  if (type === 'performance' || type === 'perf') return 'performance'
  if (type === 'ssl' || type === 'dns' || type === 'port' || type === 'heartbeat') return type
  return 'http'
}

export function kindColor(kind: MonitorKind): string {
  return kindMeta[kind].color
}

export default function MonitorKindIcon({
  kind,
  size = 18,
  color,
}: {
  kind: MonitorKind
  size?: number
  color?: string
}) {
  const meta = kindMeta[kind] || kindMeta.http
  const Icon = meta.Icon
  return <Icon size={size} color={color || meta.color} strokeWidth={1.75} aria-hidden />
}
