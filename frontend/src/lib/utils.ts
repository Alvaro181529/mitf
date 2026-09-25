import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(dateString: string | null | undefined): string {
  if (!dateString) return "N/A"
  try {
    const d = new Date(dateString)
    if (isNaN(d.getTime())) return dateString
    return d.toLocaleString("es-ES", {
      dateStyle: "medium",
      timeStyle: "medium",
    })
  } catch {
    return dateString
  }
}

export function formatNumber(n: number | string | undefined | null): string {
  if (n === undefined || n === null) return "0"
  const val = typeof n === "string" ? parseFloat(n) : n
  if (isNaN(val)) return String(n)
  return new Intl.NumberFormat("es-ES").format(val)
}
