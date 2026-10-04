import { forwardRef, type ButtonHTMLAttributes } from 'react'
import { cn } from '../../lib/utils'

type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'danger'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: 'sm' | 'md' | 'lg' | 'icon'
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button(
    {
      className,
      variant = 'default',
      size = 'md',
      type = 'button',
      ...props
    },
    ref,
  ) {
    return (
      <button
        ref={ref}
        className={cn(
          'button',
          `button-${variant}`,
          `button-${size}`,
          className,
        )}
        type={type}
        {...props}
      />
    )
  },
)
