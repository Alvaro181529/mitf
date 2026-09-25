import * as React from "react"
import { cn } from "@/lib/utils"

interface ProgressProps extends React.HTMLAttributes<HTMLDivElement> {
  value: number
  max?: number
  indicatorColor?: string
}

export function Progress({ className, value, max = 100, indicatorColor, ...props }: ProgressProps) {
  const percentage = Math.min(Math.max((value / max) * 100, 0), 100)

  // Determinar color automático si no se pasa uno explícito
  let autoColor = "bg-blue-500"
  if (percentage >= 100) {
    autoColor = "bg-rose-500"
  } else if (percentage >= 80) {
    autoColor = "bg-amber-500"
  } else {
    autoColor = "bg-emerald-500"
  }

  return (
    <div
      className={cn("relative h-2.5 w-full overflow-hidden rounded-full bg-slate-800", className)}
      {...props}
    >
      <div
        className={cn("h-full transition-all duration-500 rounded-full", indicatorColor || autoColor)}
        style={{ width: `${percentage}%` }}
      />
    </div>
  )
}
