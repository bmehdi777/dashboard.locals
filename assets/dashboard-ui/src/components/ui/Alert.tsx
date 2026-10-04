import type { HTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

export function Alert({
  className,
  variant = 'info',
  ...props
}: HTMLAttributes<HTMLDivElement> & {
  variant?: 'info' | 'success' | 'warning' | 'danger'
}) {
  return (
    <div
      role={variant === 'danger' ? 'alert' : 'status'}
      className={cn('alert', `alert-${variant}`, className)}
      {...props}
    />
  )
}
