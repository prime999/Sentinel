import CheckTypePill from './CheckTypePill'

/** Alias kept for existing call sites — same Lucide type pills as the monitors list. */
export default function TypeBadge({ type, url }: { type?: string; url?: string }) {
  return <CheckTypePill type={type} url={url} />
}
