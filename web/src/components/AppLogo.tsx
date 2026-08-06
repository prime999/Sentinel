import defaultLogo from '../assets/logo.svg'

export { defaultLogo }

export default function AppLogo({
  src,
  size = 32,
  alt = 'Sentinel',
}: {
  src?: string | null
  size?: number
  alt?: string
}) {
  return (
    <img
      src={src || defaultLogo}
      alt={alt}
      width={size}
      height={size}
      style={{
        width: size,
        height: size,
        borderRadius: Math.max(4, Math.round(size * 0.2)),
        objectFit: 'contain',
        flexShrink: 0,
        display: 'block',
      }}
    />
  )
}
