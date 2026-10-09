import type { ReactNode } from 'react'
import type { MonitorType } from '../api'

export type MonitorKind = 'site' | MonitorType | 'standalone'

const sizeDefault = 18

function SvgWrap({ children, size }: { children: ReactNode; size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden>
      {children}
    </svg>
  )
}

export default function MonitorKindIcon({ kind, size = sizeDefault }: { kind: MonitorKind; size?: number }) {
  switch (kind) {
    case 'site':
      return (
        <SvgWrap size={size}>
          <path
            d="M12 3.5 18.5 7v10L12 20.5 5.5 17V7L12 3.5Z"
            stroke="currentColor"
            strokeWidth="1.6"
            strokeLinejoin="round"
          />
          <path
            d="M9.5 10.5h5v5h-5v-5Z"
            stroke="currentColor"
            strokeWidth="1.4"
            strokeLinejoin="round"
          />
          <path d="M12 10.5V8M12 15.5V18" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
        </SvgWrap>
      )
    case 'http':
      return (
        <SvgWrap size={size}>
          <rect x="3.5" y="5" width="17" height="12" rx="1.5" stroke="currentColor" strokeWidth="1.6" />
          <path d="M3.5 9h17" stroke="currentColor" strokeWidth="1.6" />
          <path d="M8 17.5h8" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
        </SvgWrap>
      )
    case 'ssl':
      return (
        <SvgWrap size={size}>
          <path
            d="M12 3.5 16 5.5v4.2c0 3.1-1.7 5.9-4 7.3-2.3-1.4-4-4.2-4-7.3V5.5l4-2Z"
            stroke="currentColor"
            strokeWidth="1.6"
            strokeLinejoin="round"
          />
          <path d="M9.5 12.5 11 14l3.5-3.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </SvgWrap>
      )
    case 'port':
      return (
        <SvgWrap size={size}>
          <path d="M6 8h12M6 12h12M6 16h12" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
          <path d="M4 8v8M20 8v8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
        </SvgWrap>
      )
    case 'dns':
      return (
        <SvgWrap size={size}>
          <circle cx="12" cy="12" r="7.5" stroke="currentColor" strokeWidth="1.6" />
          <path d="M4.5 12h15M12 4.5c2.5 2.8 2.5 12.2 0 15M12 4.5c-2.5 2.8-2.5 12.2 0 15" stroke="currentColor" strokeWidth="1.4" />
        </SvgWrap>
      )
    case 'heartbeat':
    case 'standalone':
      return (
        <SvgWrap size={size}>
          <path
            d="M4 12h2.5l2-4 3 8 2.5-5H20"
            stroke="currentColor"
            strokeWidth="1.6"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </SvgWrap>
      )
    default:
      return (
        <SvgWrap size={size}>
          <circle cx="12" cy="12" r="7" stroke="currentColor" strokeWidth="1.6" />
        </SvgWrap>
      )
  }
}

export function monitorKindFor(type?: string, standalone?: boolean): MonitorKind {
  if (standalone) return 'standalone'
  if (!type || type === 'http') return 'http'
  return type as MonitorKind
}
