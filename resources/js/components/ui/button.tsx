import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-xl text-sm font-semibold transition-all duration-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "btn-primary text-white shadow-primary hover:transform hover:-translate-y-0.5 hover:shadow-lg",
        destructive:
          "bg-gradient-to-r from-red-500 to-pink-500 text-white shadow-md hover:shadow-lg hover:from-red-600 hover:to-pink-600",
        outline:
          "btn-outline backdrop-blur-sm",
        secondary:
          "btn-secondary text-white shadow-secondary hover:transform hover:-translate-y-0.5 hover:shadow-lg",
        ghost: "glass-tertiary hover:bg-white/30 text-gray-700",
        link: "text-blue-600 underline-offset-4 hover:underline hover:text-blue-700",
        glass: "glass shadow-glass hover:bg-white/40 text-gray-700",
      },
      size: {
        default: "h-10 px-6 py-2.5",
        sm: "h-8 rounded-lg px-4 text-xs",
        lg: "h-12 rounded-2xl px-8 text-base",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }