"use client"

import React, { useState, useEffect, useMemo } from "react"
import { TubeWarmupResponse, ATRECInitialData, WarmupRoutineCycle } from "@/types"
import { fetchTubeWarmup } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Flame,
  Clock,
  Thermometer,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Zap,
  ArrowUpRight,
  ShieldAlert,
  ClipboardPen,
  ChevronRight,
  ChevronUp,
  ChevronDown,
  ArrowUpDown,
  Activity,
  History,
  Timer,
} from "lucide-react"

type RoutineSortField =
  | "id"
  | "routine_type"
  | "status"
  | "date"
  | "initial_temp_celsius"
  | "final_temp_celsius"
  | "temp_rise_celsius"
  | "duration_seconds"
  | "ermes_code"

type SortOrder = "asc" | "desc" | null

interface TubeWarmupViewerProps {
  serverId?: number
  onAutoFillATREC?: (data: ATRECInitialData) => void
}

export function TubeWarmupViewer({ serverId, onAutoFillATREC }: TubeWarmupViewerProps) {
  const [data, setData] = useState<TubeWarmupResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [limit, setLimit] = useState<number>(50)
  const [activeTab, setActiveTab] = useState<"cycles" | "skipped">("cycles")

  // Ordenamiento interactivo de la tabla de ciclos
  const [sortField, setSortField] = useState<RoutineSortField | null>(null)
  const [sortOrder, setSortOrder] = useState<SortOrder>(null)

  const loadWarmupData = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await fetchTubeWarmup(serverId, limit)
      setData(res)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error cargando rutinas de calentamiento de tubo"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadWarmupData()
  }, [serverId, limit])

  const handleReportSkippedWarmup = (examId: string, ermesCode: string, msg: string) => {
    if (!onAutoFillATREC) return
    onAutoFillATREC({
      subsystem_id: 1, // Subsistema 1: Generador / Tubo RX
      severity: "CRITICAL",
      error_code: ermesCode || "200110044",
      description: `[Alerta de Calentamiento Omitido] Examen #${examId}: ${msg}. Reducción obligatoria de mA activo en generador para salvaguardar el ánodo de tungsteno.`,
      source_log_id: `EXAM-${examId}`,
    })
  }

  const handleSort = (field: RoutineSortField) => {
    if (sortField === field) {
      if (sortOrder === "asc") {
        setSortOrder("desc")
      } else if (sortOrder === "desc") {
        setSortField(null)
        setSortOrder(null)
      } else {
        setSortOrder("asc")
      }
    } else {
      setSortField(field)
      setSortOrder("asc")
    }
  }

  const sortedRoutines = useMemo(() => {
    if (!data || !data.routines) return []
    if (!sortField || !sortOrder) return data.routines

    return [...data.routines].sort((a: WarmupRoutineCycle, b: WarmupRoutineCycle) => {
      switch (sortField) {
        case "id":
          return sortOrder === "asc" ? a.id - b.id : b.id - a.id
        case "routine_type":
          return sortOrder === "asc"
            ? a.routine_type.localeCompare(b.routine_type)
            : b.routine_type.localeCompare(a.routine_type)
        case "status":
          return sortOrder === "asc"
            ? a.status.localeCompare(b.status)
            : b.status.localeCompare(a.status)
        case "date": {
          const tA = a.start_epoch ?? (a.start_time ? new Date(a.start_time).getTime() : 0)
          const tB = b.start_epoch ?? (b.start_time ? new Date(b.start_time).getTime() : 0)
          return sortOrder === "asc" ? tA - tB : tB - tA
        }
        case "initial_temp_celsius":
          return sortOrder === "asc"
            ? a.initial_temp_celsius - b.initial_temp_celsius
            : b.initial_temp_celsius - a.initial_temp_celsius
        case "final_temp_celsius":
          return sortOrder === "asc"
            ? a.final_temp_celsius - b.final_temp_celsius
            : b.final_temp_celsius - a.final_temp_celsius
        case "temp_rise_celsius":
          return sortOrder === "asc"
            ? a.temp_rise_celsius - b.temp_rise_celsius
            : b.temp_rise_celsius - a.temp_rise_celsius
        case "duration_seconds":
          return sortOrder === "asc"
            ? a.duration_seconds - b.duration_seconds
            : b.duration_seconds - a.duration_seconds
        case "ermes_code": {
          const eA = `${a.start_ermes_code} ${a.end_ermes_code}`
          const eB = `${b.start_ermes_code} ${b.end_ermes_code}`
          return sortOrder === "asc" ? eA.localeCompare(eB) : eB.localeCompare(eA)
        }
        default:
          return 0
      }
    })
  }, [data, sortField, sortOrder])

  const renderSortIndicator = (field: RoutineSortField) => {
    if (sortField !== field || !sortOrder) {
      return <ArrowUpDown className="w-3.5 h-3.5 text-slate-500 opacity-60 group-hover:opacity-100 transition-opacity" />
    }
    if (sortOrder === "asc") {
      return <ChevronUp className="w-3.5 h-3.5 text-amber-400 font-bold" />
    }
    return <ChevronDown className="w-3.5 h-3.5 text-amber-400 font-bold" />
  }

  return (
    <div className="space-y-6">
      {/* Encabezado */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900/80 p-5 rounded-xl border border-slate-800">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
            <Flame className="h-5 w-5 text-amber-500 animate-pulse" />
            Rutinas de Calentamiento de Tubo (Warm-up Routines)
          </h2>
          <p className="text-sm text-slate-400 mt-1">
            Supervisión térmica del ánodo de Rayos X, ciclos pre-escaneo diarios y auditoría de omisiones (gesys_ct99.log)
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 text-xs text-slate-400 bg-slate-950 px-2.5 py-1.5 rounded-lg border border-slate-800">
            <span>Límite:</span>
            <select
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
              className="bg-transparent text-slate-200 font-mono text-xs focus:outline-none cursor-pointer"
            >
              <option value={10} className="bg-slate-900 text-slate-200">10 ciclos</option>
              <option value={25} className="bg-slate-900 text-slate-200">25 ciclos</option>
              <option value={50} className="bg-slate-900 text-slate-200">50 ciclos</option>
              <option value={100} className="bg-slate-900 text-slate-200">100 ciclos</option>
            </select>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={loadWarmupData}
            disabled={loading}
            className="gap-1.5 text-xs"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Actualizar
          </Button>
        </div>
      </div>

      {loading && !data ? (
        <div className="flex justify-center p-12 text-slate-400">
          <RefreshCw className="h-6 w-6 animate-spin mr-2 text-amber-500" />
          <span>Analizando registros térmicos de calentamiento de tubo...</span>
        </div>
      ) : error && !data ? (
        <Card className="border-rose-900 bg-rose-950/20 p-6 text-center text-rose-300">
          {error}
        </Card>
      ) : !data ? null : (
        <>
          {/* Tarjetas KPI */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Card className="border-slate-800 bg-slate-900/60 shadow-lg">
              <CardContent className="p-4 flex items-center gap-3">
                <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded-xl text-amber-400">
                  <Flame className="w-5 h-5" />
                </div>
                <div>
                  <span className="text-xs text-slate-400">Ciclos Completados</span>
                  <div className="text-2xl font-bold font-mono text-slate-100">
                    {data.summary.total_completed_routines}
                  </div>
                  <span className="text-[11px] text-emerald-400 flex items-center gap-1">
                    <CheckCircle2 className="w-3 h-3" /> dailyPrepRx
                  </span>
                </div>
              </CardContent>
            </Card>

            <Card className="border-slate-800 bg-slate-900/60 shadow-lg">
              <CardContent className="p-4 flex items-center gap-3">
                <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl text-rose-400">
                  <ShieldAlert className="w-5 h-5" />
                </div>
                <div>
                  <span className="text-xs text-slate-400">Warm-ups Omitidos</span>
                  <div className="text-2xl font-bold font-mono text-rose-400">
                    {data.summary.total_skipped_events}
                  </div>
                  <span className="text-[11px] text-slate-400">scanRx (Ermes #200110044)</span>
                </div>
              </CardContent>
            </Card>

            <Card className="border-slate-800 bg-slate-900/60 shadow-lg">
              <CardContent className="p-4 flex items-center gap-3">
                <div className="p-3 bg-blue-500/10 border border-blue-500/20 rounded-xl text-blue-400">
                  <Thermometer className="w-5 h-5" />
                </div>
                <div>
                  <span className="text-xs text-slate-400">ΔT Promedio Ascenso</span>
                  <div className="text-2xl font-bold font-mono text-blue-400">
                    +{data.summary.average_temp_rise_celsius.toFixed(1)} °C
                  </div>
                  <span className="text-[11px] text-slate-400">
                    Último: +{data.summary.last_temp_rise_celsius.toFixed(1)} °C
                  </span>
                </div>
              </CardContent>
            </Card>

            <Card className="border-slate-800 bg-slate-900/60 shadow-lg">
              <CardContent className="p-4 flex items-center gap-3">
                <div className="p-3 bg-purple-500/10 border border-purple-500/20 rounded-xl text-purple-400">
                  <Timer className="w-5 h-5" />
                </div>
                <div>
                  <span className="text-xs text-slate-400">Duración Promedio</span>
                  <div className="text-2xl font-bold font-mono text-purple-400">
                    {data.summary.average_duration_seconds} seg
                  </div>
                  <span className="text-[11px] text-slate-400 font-mono">
                    Estado Ánodo: <strong className="text-emerald-400">{data.summary.tube_thermal_status}</strong>
                  </span>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Banner de advertencia si existen eventos de omisión de calentamiento */}
          {data.summary.total_skipped_events > 0 && (
            <div className="bg-amber-950/30 border border-amber-800/60 rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div className="flex items-start gap-3">
                <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
                <div>
                  <h4 className="text-sm font-semibold text-amber-200">
                    Auditoría de Omisión de Calentamiento Detectada (Restricción de mA Activa)
                  </h4>
                  <p className="text-xs text-slate-300 mt-0.5">
                    Se detectaron {data.summary.total_skipped_events} eventos donde se omitió el ciclo Cold Tube Warmup.
                    El generador redujo automáticamente la potencia a un límite seguro (440 mA @ 120kV).
                  </p>
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setActiveTab("skipped")}
                className="shrink-0 border-amber-600/40 text-amber-300 hover:bg-amber-950/50 text-xs"
              >
                Revisar Auditoría →
              </Button>
            </div>
          )}

          {/* Pestañas de Navegación */}
          <div className="flex items-center gap-2 border-b border-slate-800 pb-2">
            <button
              onClick={() => setActiveTab("cycles")}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                activeTab === "cycles"
                  ? "bg-amber-500/10 text-amber-400 border border-amber-500/30"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <History className="w-3.5 h-3.5" />
              Historial de Ciclos ({data.routines.length})
            </button>
            <button
              onClick={() => setActiveTab("skipped")}
              className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                activeTab === "skipped"
                  ? "bg-rose-500/10 text-rose-400 border border-rose-500/30"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              <ShieldAlert className="w-3.5 h-3.5" />
              Auditoría de Omisiones ({data.skipped_events.length})
            </button>
          </div>

          {/* Contenido Pestaña 1: Ciclos de Calentamiento */}
          {activeTab === "cycles" && (
            <Card className="border-slate-800 bg-slate-900/60 shadow-xl overflow-hidden">
              <CardHeader className="pb-3 border-b border-slate-800">
                <CardTitle className="text-base font-semibold text-slate-200 flex items-center justify-between">
                  <span>Ciclos Registrados en gesys_ct99.log (dailyPrepRx)</span>
                  <div className="flex items-center gap-3">
                    {sortField && sortOrder && (
                      <span className="text-amber-400 font-mono text-[11px] font-normal">
                        Ordenado por {sortField} ({sortOrder === "asc" ? "▲ Asc" : "▼ Desc"})
                      </span>
                    )}
                    <span className="text-xs font-mono font-normal text-slate-400">
                      Mostrando {sortedRoutines.length} eventos
                    </span>
                  </div>
                </CardTitle>
              </CardHeader>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-slate-300">
                  <thead className="bg-slate-950/80 text-xs uppercase text-slate-400 border-b border-slate-800 font-mono select-none">
                    <tr>
                      {/* # */}
                      <th
                        onClick={() => handleSort("id")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>#</span>
                          {renderSortIndicator("id")}
                        </div>
                      </th>

                      {/* Tipo de Rutina */}
                      <th
                        onClick={() => handleSort("routine_type")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>Tipo de Rutina</span>
                          {renderSortIndicator("routine_type")}
                        </div>
                      </th>

                      {/* Estado */}
                      <th
                        onClick={() => handleSort("status")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>Estado</span>
                          {renderSortIndicator("status")}
                        </div>
                      </th>

                      {/* Fecha Inicio / Fin */}
                      <th
                        onClick={() => handleSort("date")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>Fecha Inicio / Fin</span>
                          {renderSortIndicator("date")}
                        </div>
                      </th>

                      {/* T. Inicial */}
                      <th
                        onClick={() => handleSort("initial_temp_celsius")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>T. Inicial</span>
                          {renderSortIndicator("initial_temp_celsius")}
                        </div>
                      </th>

                      {/* T. Final */}
                      <th
                        onClick={() => handleSort("final_temp_celsius")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>T. Final</span>
                          {renderSortIndicator("final_temp_celsius")}
                        </div>
                      </th>

                      {/* Delta Térmico */}
                      <th
                        onClick={() => handleSort("temp_rise_celsius")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>Delta Térmico (ΔT)</span>
                          {renderSortIndicator("temp_rise_celsius")}
                        </div>
                      </th>

                      {/* Duración */}
                      <th
                        onClick={() => handleSort("duration_seconds")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>Duración</span>
                          {renderSortIndicator("duration_seconds")}
                        </div>
                      </th>

                      {/* Códigos Ermes */}
                      <th
                        onClick={() => handleSort("ermes_code")}
                        className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                      >
                        <div className="flex items-center gap-1.5">
                          <span>Códigos Ermes</span>
                          {renderSortIndicator("ermes_code")}
                        </div>
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60 font-mono text-xs">
                    {sortedRoutines.map((routine: WarmupRoutineCycle) => (
                      <tr key={routine.id} className="hover:bg-slate-800/40 transition-colors">
                        <td className="px-4 py-3 text-slate-400 font-bold">#{routine.id}</td>
                        <td className="px-4 py-3 font-semibold text-slate-200">
                          <span className="inline-flex items-center gap-1.5">
                            <Flame className="w-3.5 h-3.5 text-amber-500" />
                            {routine.routine_type.replace(/_/g, " ")}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant="success" className="gap-1 text-[10px]">
                            <CheckCircle2 className="w-2.5 h-2.5" /> {routine.status}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-slate-300">
                          <div>{routine.start_time}</div>
                          {routine.end_time && (
                            <div className="text-[11px] text-slate-400">
                              → {routine.end_time}
                            </div>
                          )}
                        </td>
                        <td className="px-4 py-3 text-blue-300 font-bold">
                          {routine.initial_temp_celsius.toFixed(1)} °C
                        </td>
                        <td className="px-4 py-3 text-amber-400 font-bold">
                          {routine.final_temp_celsius > 0 ? `${routine.final_temp_celsius.toFixed(1)} °C` : "-"}
                        </td>
                        <td className="px-4 py-3">
                          {routine.temp_rise_celsius > 0 ? (
                            <span className="text-emerald-400 font-bold bg-emerald-950/40 px-2 py-0.5 rounded border border-emerald-900/50">
                              +{routine.temp_rise_celsius.toFixed(1)} °C
                            </span>
                          ) : (
                            <span className="text-slate-500">-</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-slate-300">
                          {routine.duration_seconds > 0 ? `${routine.duration_seconds} seg` : "-"}
                        </td>
                        <td className="px-4 py-3 text-slate-400 text-[11px]">
                          {routine.start_ermes_code} → {routine.end_ermes_code}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          )}

          {/* Contenido Pestaña 2: Eventos Omitidos */}
          {activeTab === "skipped" && (
            <Card className="border-slate-800 bg-slate-900/60 shadow-xl overflow-hidden">
              <CardHeader className="pb-3 border-b border-slate-800">
                <CardTitle className="text-base font-semibold text-slate-200 flex items-center justify-between">
                  <span>Eventos de Omisión de Calentamiento (Ermes #200110044)</span>
                  <span className="text-xs font-mono font-normal text-rose-400">
                    Proceso: scanRx | ZeusrpGlobalActions.c:854
                  </span>
                </CardTitle>
                <CardDescription className="text-xs text-slate-400">
                  Estos eventos indican que se inició una adquisición de escáner sin la temperatura mínima requerida en el ánodo.
                </CardDescription>
              </CardHeader>
              <div className="divide-y divide-slate-800/80">
                {data.skipped_events.map((skip, idx) => (
                  <div key={idx} className="p-4 hover:bg-slate-800/30 transition-colors space-y-2">
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <Badge variant="destructive" className="gap-1 text-[11px]">
                          <AlertTriangle className="w-3 h-3" /> {skip.severity}
                        </Badge>
                        <span className="text-xs font-mono font-bold text-slate-200">
                          Examen #{skip.exam_id}
                        </span>
                        <span className="text-xs text-slate-400 font-mono">
                          • {skip.sr_id} • Ermes: {skip.ermes_code}
                        </span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-slate-400 font-mono">
                          {skip.date}
                        </span>
                        {onAutoFillATREC && (
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={() =>
                              handleReportSkippedWarmup(skip.exam_id, skip.ermes_code, skip.message)
                            }
                            className="h-7 text-xs gap-1.5 bg-amber-600/20 text-amber-300 hover:bg-amber-600/30 border border-amber-600/40"
                          >
                            <ClipboardPen className="w-3 h-3" />
                            Reportar en ATREC
                          </Button>
                        )}
                      </div>
                    </div>
                    <p className="text-xs font-mono text-rose-300 bg-slate-950 p-2.5 rounded border border-rose-950">
                      {skip.message}
                    </p>
                    <div className="flex items-center gap-4 text-[11px] text-slate-400 font-mono">
                      <span>Límite 120kV: <strong className="text-amber-300">{skip.max_ma_120kv} mA</strong></span>
                      <span>Límite 140kV: <strong className="text-amber-300">{skip.max_ma_140kv} mA</strong></span>
                      <span>Módulo: <strong className="text-slate-300">{skip.process}</strong></span>
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  )
}
