import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold transition-all duration-300 backdrop-blur-sm",
  {
    variants: {
      variant: {
        default:
          "bg-gradient-to-r from-purple-500/20 to-purple-600/20 text-purple-700 border border-purple-300/50",
        secondary:
          "bg-gradient-to-r from-pink-500/20 to-red-500/20 text-pink-700 border border-pink-300/50",
        destructive:
          "bg-gradient-to-r from-red-500/20 to-red-600/20 text-red-700 border border-red-300/50",
        outline: "glass-tertiary text-gray-700 border border-white/50",
        success:
          "bg-gradient-to-r from-green-500/20 to-blue-500/20 text-green-700 border border-green-300/50",
        warning:
          "bg-gradient-to-r from-yellow-500/20 to-orange-500/20 text-yellow-700 border border-yellow-300/50",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  )
}

export { Badge, badgeVariants }