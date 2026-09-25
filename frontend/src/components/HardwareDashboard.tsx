"use client"

import React, { useState, useEffect } from "react"
import Link from "next/link"
import { HardwareSummaryData } from "@/types"
import { fetchHardwareSummary } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Activity,
  Flame,
  RotateCw,
  Cpu,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Zap,
} from "lucide-react"

export const TSM_DEFAULT_PROCESSES: Record<number, string[]> = {
  1: ["tubemgr", "scanRx", "hvg"],
  2: ["rotmgr", "gantry", "drive"],
  3: ["tablemgr", "tgp", "couch"],
  4: ["dasmgr", "das", "pdu"],
  5: ["gscb", "scanmgr", "estop"],
}

interface HardwareDashboardProps {
  serverId?: number
}

export function HardwareDashboard({ serverId }: HardwareDashboardProps) {
  const [data, setData] = useState<HardwareSummaryData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadData = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await fetchHardwareSummary(serverId)
      setData(res.data)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error cargando métricas de hardware"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [serverId])

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "NORMAL":
      case "OPERATIONAL":
      case "OK":
        return (
          <Badge variant="success" className="gap-1">
            <CheckCircle2 className="w-3 h-3" /> NORMAL
          </Badge>
        )
      case "WARNING":
        return (
          <Badge variant="warning" className="gap-1">
            <AlertTriangle className="w-3 h-3" /> ADVERTENCIA
          </Badge>
        )
      case "CRITICAL":
        return (
          <Badge variant="destructive" className="gap-1">
            <XCircle className="w-3 h-3" /> CRÍTICO
          </Badge>
        )
      default:
        return <Badge variant="secondary">{status}</Badge>
    }
  }

  if (loading && !data) {
    return (
      <div className="flex justify-center p-12 text-slate-400">
        <RefreshCw className="h-6 w-6 animate-spin mr-2 text-blue-500" />
        <span>Cargando métricas de hardware del tomógrafo...</span>
      </div>
    )
  }

  if (error && !data) {
    return (
      <Card className="border-rose-900 bg-rose-950/20 p-6 text-center text-rose-300">
        {error}
      </Card>
    )
  }

  if (!data) return null

  const tube = data.tube_rx
  const gantry = data.gantry_rotor
  const subsystems = data.subsystems || data.tsm_subsystems_health || []

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900/80 p-5 rounded-xl border border-slate-800">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
            <Activity className="h-5 w-5 text-blue-400" />
            Dashboard de Hardware y Telemetría Crítica
          </h2>
          <p className="text-sm text-slate-400 mt-1">
            Telemetría de Tubo de Rayos X, Gantry y los 5 Subsistemas Físicos TSM
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={loadData}
          disabled={loading}
          className="gap-1.5"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Recargar
        </Button>
      </div>

      {/* Tarjetas Principales: Tubo y Gantry */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Tubo RX */}
        <Card className="border-slate-800 bg-slate-900/80">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-base text-slate-100 flex items-center gap-2">
                <Zap className="h-4 w-4 text-amber-400" />
                Tubo de Rayos X (Performix HD)
              </CardTitle>
              {getStatusBadge(tube.status)}
            </div>
            <CardDescription className="text-xs text-slate-400">
              Desgaste por mAs acumulados y capacidad calorífica del ánodo
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Desgaste Acumulado (mAs)</span>
                <span className="font-mono text-slate-200">
                  {tube.accumulated_mas.toLocaleString()} / {tube.limit_nominal_mas.toLocaleString()} mAs ({tube.usage_percentage.toFixed(1)}%)
                </span>
              </div>
              <div className="h-2 w-full bg-slate-800 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all ${
                    tube.usage_percentage > 90
                      ? "bg-rose-500"
                      : tube.usage_percentage > 75
                      ? "bg-amber-500"
                      : "bg-blue-500"
                  }`}
                  style={{ width: `${Math.min(tube.usage_percentage, 100)}%` }}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 pt-2 border-t border-slate-800">
              <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800/80">
                <div className="flex items-center gap-1.5 text-xs text-slate-400 mb-1">
                  <Flame className="w-3.5 h-3.5 text-rose-400" />
                  Temperatura del Ánodo
                </div>
                <div className="font-mono text-base font-bold text-slate-100">
                  {tube.anode_temperature_celsius.toFixed(1)} °C
                </div>
              </div>

              <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800/80">
                <div className="text-xs text-slate-400 mb-1">Carga Calorífica</div>
                <div className="font-mono text-base font-bold text-slate-100">
                  {tube.anode_thermal_capacity_mhu.toFixed(2)} MHU
                </div>
              </div>
            </div>

            <div className="pt-2 border-t border-slate-800/80 flex items-center justify-between">
              <span className="text-xs text-slate-400">Ciclos de Calentamiento</span>
              <Link
                href="/warmup"
                className="inline-flex items-center gap-1 text-xs text-amber-400 hover:text-amber-300 font-medium transition-colors"
              >
                <Flame className="w-3.5 h-3.5 text-amber-500" /> Ver Rutinas Warm-up →
              </Link>
            </div>
          </CardContent>
        </Card>

        {/* Gantry & Rotor */}
        <Card className="border-slate-800 bg-slate-900/80">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-base text-slate-100 flex items-center gap-2">
                <RotateCw className="h-4 w-4 text-blue-400" />
                Rotor del Gantry & Mecánica
              </CardTitle>
              {getStatusBadge(gantry.status)}
            </div>
            <CardDescription className="text-xs text-slate-400">
              Revoluciones de rotación continua del estator y anillos deslizantes
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between text-xs">
                <span className="text-slate-400">Revoluciones Acumuladas</span>
                <span className="font-mono text-slate-200">
                  {gantry.accumulated_revolutions.toLocaleString()} / {gantry.limit_revolutions.toLocaleString()} revs ({gantry.usage_percentage.toFixed(1)}%)
                </span>
              </div>
              <div className="h-2 w-full bg-slate-800 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all ${
                    gantry.usage_percentage > 90
                      ? "bg-rose-500"
                      : gantry.usage_percentage > 75
                      ? "bg-amber-500"
                      : "bg-emerald-500"
                  }`}
                  style={{ width: `${Math.min(gantry.usage_percentage, 100)}%` }}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3 pt-2 border-t border-slate-800">
              <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800/80">
                <div className="text-xs text-slate-400 mb-1">Vida Mecánica</div>
                <div className="font-mono text-base font-bold text-slate-100">
                  {gantry.usage_percentage.toFixed(1)} %
                </div>
              </div>

              <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800/80">
                <div className="text-xs text-slate-400 mb-1">Límite Rodamientos</div>
                <div className="font-mono text-base font-bold text-slate-100">
                  {gantry.limit_revolutions.toLocaleString()}
                </div>
              </div>
            </div>

            <div className="pt-2 border-t border-slate-800/80 flex items-center justify-between">
              <span className="text-xs text-slate-400">Control Cinemático</span>
              <span className="text-xs font-mono text-emerald-400">1.0s / 360° Rot</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Subsistemas TSM */}
      <Card className="border-slate-800 bg-slate-900/80">
        <CardHeader className="pb-3">
          <CardTitle className="text-base text-slate-100 flex items-center gap-2">
            <Cpu className="h-4 w-4 text-purple-400" />
            Estado de Salud de los 5 Subsistemas Físicos TSM
          </CardTitle>
          <CardDescription className="text-xs text-slate-400">
            Monitoreo preventivo de procesos críticos y buses según la arquitectura GE Healthcare
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
            {subsystems.map((sub) => {
              const processes =
                sub.associated_processes || TSM_DEFAULT_PROCESSES[sub.id] || []

              return (
                <div
                  key={sub.id}
                  className="bg-slate-950/60 p-3 rounded-lg border border-slate-800 flex flex-col justify-between space-y-3"
                >
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs font-bold text-slate-300">
                        #{sub.id}
                      </span>
                      {getStatusBadge(sub.status)}
                    </div>
                    <div className="text-xs font-medium text-slate-100 line-clamp-1" title={sub.name}>
                      {sub.name}
                    </div>
                  </div>

                  <div className="space-y-1.5 pt-2 border-t border-slate-900 text-[11px]">
                    <div className="flex items-center justify-between text-slate-400">
                      <span>Alertas Activas:</span>
                      <span className={`font-mono font-bold ${
                        (sub.active_alerts_count || 0) > 0 ? "text-amber-400" : "text-emerald-400"
                      }`}>
                        {sub.active_alerts_count || 0}
                      </span>
                    </div>

                    <div className="text-[10px] text-slate-400 flex flex-wrap gap-1 mt-1">
                      {processes.slice(0, 3).map((p) => (
                        <span
                          key={p}
                          className="bg-slate-900 px-1.5 py-0.5 rounded text-slate-400 border border-slate-800"
                        >
                          {p}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
