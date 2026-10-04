import type { HTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

export function Badge({
  className,
  variant = 'neutral',
  ...props
}: HTMLAttributes<HTMLSpanElement> & {
  variant?: 'neutral' | 'success' | 'warning' | 'danger' | 'info'
}) {
  return <span className={cn('badge', `badge-${variant}`, className)} {...props} />
}
