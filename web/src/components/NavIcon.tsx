import { CSSProperties } from 'react'
import { iconSizes } from '../icons'

/** Tinted icon from a public file path (SVG/PNG). Color follows `currentColor`. */
export default function NavIcon({
  src,
  size = iconSizes.nav,
  alt = '',
}: {
  src: string
  size?: number
  alt?: string
}) {
  const style: CSSProperties = {
    width: size,
    height: size,
    flexShrink: 0,
    display: 'inline-block',
    backgroundColor: 'currentColor',
    WebkitMaskImage: `url(${src})`,
    maskImage: `url(${src})`,
    WebkitMaskSize: 'contain',
    maskSize: 'contain',
    WebkitMaskRepeat: 'no-repeat',
    maskRepeat: 'no-repeat',
    WebkitMaskPosition: 'center',
    maskPosition: 'center',
  }
  return <span role="img" aria-label={alt || undefined} aria-hidden={!alt} style={style} />
}
