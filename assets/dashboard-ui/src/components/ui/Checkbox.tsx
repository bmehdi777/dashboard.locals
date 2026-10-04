import { forwardRef, type InputHTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

export const Checkbox = forwardRef<
  HTMLInputElement,
  InputHTMLAttributes<HTMLInputElement>
>(function Checkbox({ className, type = 'checkbox', ...props }, ref) {
  return (
    <input
      ref={ref}
      type={type}
      className={cn('checkbox', className)}
      {...props}
    />
  )
})
