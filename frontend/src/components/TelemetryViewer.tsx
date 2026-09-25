"use client"

import React, { useState, useEffect, useMemo } from "react"
import { fetchJediCanLogs, fetchDasErrorsLogs } from "@/lib/api"
import { JediCanLogsResponse, CANFrame } from "@/types"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Radio,
  Terminal,
  RefreshCw,
  Cpu,
  Layers,
  Heart,
  Table,
  Code2,
  Activity,
  ArrowUpRight,
  ArrowDownLeft,
  ChevronUp,
  ChevronDown,
  ArrowUpDown,
} from "lucide-react"

type CANSortField = "timestamp_ms" | "direction" | "can_id" | "dlc" | "payload" | "is_heartbeat" | "subsystem_id"
type SortOrder = "asc" | "desc" | null

interface TelemetryViewerProps {
  serverId?: number
}

export function TelemetryViewer({ serverId }: TelemetryViewerProps) {
  const [activeTab, setActiveTab] = useState<"jedi-can" | "das-errors">("jedi-can")
  const [canData, setCanData] = useState<JediCanLogsResponse | null>(null)
  const [content, setContent] = useState<string>("")
  const [viewMode, setViewMode] = useState<"table" | "raw">("table")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lines, setLines] = useState(100)

  // Ordenamiento interactivo de columnas CAN
  const [sortField, setSortField] = useState<CANSortField | null>(null)
  const [sortOrder, setSortOrder] = useState<SortOrder>(null)

  const loadContent = async () => {
    try {
      setLoading(true)
      setError(null)
      if (activeTab === "jedi-can") {
        const res = await fetchJediCanLogs(serverId, lines)
        setCanData(res)
        setContent(res.content || "")
      } else {
        const res = await fetchDasErrorsLogs(serverId, lines)
        setCanData(null)
        setContent(res.content || "El archivo está vacío o no contiene registros recientes.")
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error cargando archivo de telemetría"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadContent()
  }, [activeTab, serverId, lines])

  const handleSort = (field: CANSortField) => {
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

  const sortedFrames = useMemo(() => {
    if (!canData || !canData.frames) return []
    if (!sortField || !sortOrder) return canData.frames

    return [...canData.frames].sort((a, b) => {
      switch (sortField) {
        case "timestamp_ms":
          return sortOrder === "asc" ? a.timestamp_ms - b.timestamp_ms : b.timestamp_ms - a.timestamp_ms
        case "direction":
          return sortOrder === "asc" ? a.direction.localeCompare(b.direction) : b.direction.localeCompare(a.direction)
        case "can_id":
          return sortOrder === "asc" ? a.can_id.localeCompare(b.can_id) : b.can_id.localeCompare(a.can_id)
        case "dlc":
          return sortOrder === "asc" ? a.dlc - b.dlc : b.dlc - a.dlc
        case "payload": {
          const pA = a.payload.join(" ")
          const pB = b.payload.join(" ")
          return sortOrder === "asc" ? pA.localeCompare(pB) : pB.localeCompare(pA)
        }
        case "is_heartbeat": {
          const vA = a.is_heartbeat ? 1 : 0
          const vB = b.is_heartbeat ? 1 : 0
          return sortOrder === "asc" ? vA - vB : vB - vA
        }
        case "subsystem_id":
          return sortOrder === "asc" ? a.subsystem_id - b.subsystem_id : b.subsystem_id - a.subsystem_id
        default:
          return 0
      }
    })
  }, [canData, sortField, sortOrder])

  const renderSortIndicator = (field: CANSortField) => {
    if (sortField !== field || !sortOrder) {
      return <ArrowUpDown className="w-3.5 h-3.5 text-slate-500 opacity-60 group-hover:opacity-100 transition-opacity" />
    }
    if (sortOrder === "asc") {
      return <ChevronUp className="w-3.5 h-3.5 text-blue-400 font-bold" />
    }
    return <ChevronDown className="w-3.5 h-3.5 text-blue-400 font-bold" />
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900/80 p-5 rounded-xl border border-slate-800">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
            <Radio className="h-5 w-5 text-emerald-400" />
            Telemetría de Buses Industriales y Detectores
          </h2>
          <p className="text-sm text-slate-400 mt-1">
            Lectura en vivo de logs especializados de bajo nivel: Bus CAN (JEDI CAN) y Flujo de Adquisición DAS
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={loadContent} disabled={loading} className="gap-1.5">
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Recargar
        </Button>
      </div>

      {/* Tabs Selector */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex gap-2">
          <Button
            variant={activeTab === "jedi-can" ? "default" : "outline"}
            size="sm"
            onClick={() => setActiveTab("jedi-can")}
            className="gap-2"
          >
            <Cpu className="w-4 h-4" /> Bus CAN: jedi_can_at_error.log
          </Button>
          <Button
            variant={activeTab === "das-errors" ? "default" : "outline"}
            size="sm"
            onClick={() => setActiveTab("das-errors")}
            className="gap-2"
          >
            <Layers className="w-4 h-4" /> DAS: dataacq.stderr.log
          </Button>
        </div>

        <div className="flex items-center gap-3">
          {activeTab === "jedi-can" && (
            <div className="flex items-center gap-1 bg-slate-900 border border-slate-800 p-1 rounded-lg">
              <button
                onClick={() => setViewMode("table")}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded text-xs transition-colors ${
                  viewMode === "table"
                    ? "bg-blue-600 text-white font-medium shadow-sm"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                <Table className="w-3.5 h-3.5" /> Tramas Parseadas
              </button>
              <button
                onClick={() => setViewMode("raw")}
                className={`flex items-center gap-1.5 px-2.5 py-1 rounded text-xs transition-colors ${
                  viewMode === "raw"
                    ? "bg-blue-600 text-white font-medium shadow-sm"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                <Code2 className="w-3.5 h-3.5" /> Texto Raw
              </button>
            </div>
          )}

          <div className="flex items-center gap-2 text-xs text-slate-400">
            <span>Últimas líneas:</span>
            {[50, 100, 200].map((l) => (
              <button
                key={l}
                onClick={() => setLines(l)}
                className={`px-2 py-1 rounded font-mono ${
                  lines === l
                    ? "bg-slate-700 text-white font-bold"
                    : "bg-slate-800 text-slate-400 hover:bg-slate-700"
                }`}
              >
                {l}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Tarjeta de Resumen CAN */}
      {activeTab === "jedi-can" && canData && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div className="bg-slate-900/60 border border-slate-800 p-3 rounded-lg flex items-center justify-between">
            <span className="text-xs text-slate-400">Subsistema Asignado:</span>
            <Badge variant="default" className="text-xs bg-indigo-600/40 text-indigo-300 border-indigo-500/40">
              #1 Generador y Tubo RX
            </Badge>
          </div>
          <div className="bg-slate-900/60 border border-slate-800 p-3 rounded-lg flex items-center justify-between">
            <span className="text-xs text-slate-400">Tramas Parseadas:</span>
            <span className="text-xs font-mono font-bold text-emerald-400">
              {canData.total_frames} tramas
            </span>
          </div>
          <div className="bg-slate-900/60 border border-slate-800 p-3 rounded-lg flex items-center justify-between">
            <span className="text-xs text-slate-400">Tramas Heartbeat (478h/479h):</span>
            <span className="text-xs font-mono font-bold text-amber-400 flex items-center gap-1">
              <Heart className="w-3 h-3 text-amber-400 animate-pulse" />
              {canData.frames.filter((f) => f.is_heartbeat).length}
            </span>
          </div>
        </div>
      )}

      {/* Visor de Consola / Terminal / Tabla */}
      <Card className="border-slate-800 bg-slate-950 shadow-2xl overflow-hidden">
        <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-800 text-xs text-slate-400">
          <span className="flex items-center gap-2 font-mono">
            <Terminal className="w-3.5 h-3.5 text-blue-400" />
            {activeTab === "jedi-can" ? "jedi_can_at_error.log" : "dataacq.stderr.log"}
          </span>
          <div className="flex items-center gap-3">
            {activeTab === "jedi-can" && sortField && sortOrder && (
              <span className="text-blue-400 font-mono text-[11px]">
                Ordenado por {sortField} ({sortOrder === "asc" ? "▲ Asc" : "▼ Desc"})
              </span>
            )}
            <span>{loading ? "Leyendo stream remoto..." : "Conexión SSH Establecida"}</span>
          </div>
        </div>
        <CardContent className="p-0">
          {loading ? (
            <div className="flex items-center justify-center p-12 text-slate-500 font-mono text-sm">
              <RefreshCw className="h-5 w-5 animate-spin mr-2 text-blue-400" />
              <span>Extrayendo registros remotos...</span>
            </div>
          ) : error ? (
            <div className="p-6 text-rose-400 text-sm font-mono">{error}</div>
          ) : activeTab === "jedi-can" && viewMode === "table" && canData ? (
            <div className="overflow-x-auto max-h-[550px]">
              <table className="w-full text-left text-xs font-mono text-slate-300">
                <thead className="bg-slate-900/90 text-slate-400 uppercase tracking-wider sticky top-0 border-b border-slate-800 select-none">
                  <tr>
                    {/* Tiempo Relativo */}
                    <th
                      onClick={() => handleSort("timestamp_ms")}
                      className="px-4 py-2.5 cursor-pointer hover:bg-slate-800/80 hover:text-slate-200 transition-colors group"
                    >
                      <div className="flex items-center gap-1.5">
                        <span>Tiempo Relativo (ms)</span>
                        {renderSortIndicator("timestamp_ms")}
                      </div>
                    </th>

                    {/* Dir */}
                    <th
                      onClick={() => handleSort("direction")}
                      className="px-4 py-2.5 cursor-pointer hover:bg-slate-800/80 hover:text-slate-200 transition-colors group"
                    >
                      <div className="flex items-center gap-1.5">
                        <span>Dir</span>
                        {renderSortIndicator("direction")}
                      </div>
                    </th>

                    {/* CAN ID */}
                    <th
                      onClick={() => handleSort("can_id")}
                      className="px-4 py-2.5 cursor-pointer hover:bg-slate-800/80 hover:text-slate-200 transition-colors group"
                    >
                      <div className="flex items-center gap-1.5">
                        <span>CAN ID</span>
                        {renderSortIndicator("can_id")}
                      </div>
                    </th>

                    {/* DLC */}
                    <th
                      onClick={() => handleSort("dlc")}
                      className="px-4 py-2.5 cursor-pointer hover:bg-slate-800/80 hover:text-slate-200 transition-colors group"
                    >
                      <div className="flex items-center gap-1.5">
                        <span>DLC</span>
                        {renderSortIndicator("dlc")}
                      </div>
                    </th>

                    {/* Payload */}
                    <th
                      onClick={() => handleSort("payload")}
                      className="px-4 py-2.5 cursor-pointer hover:bg-slate-800/80 hover:text-slate-200 transition-colors group"
                    >
                      <div className="flex items-center gap-1.5">
                        <span>Payload (Bytes)</span>
                        {renderSortIndicator("payload")}
                      </div>
                    </th>

                    {/* Tipo / Función */}
                    <th
                      onClick={() => handleSort("is_heartbeat")}
                      className="px-4 py-2.5 cursor-pointer hover:bg-slate-800/80 hover:text-slate-200 transition-colors group"
                    >
                      <div className="flex items-center gap-1.5">
                        <span>Tipo / Función</span>
                        {renderSortIndicator("is_heartbeat")}
                      </div>
                    </th>

                    {/* TSM */}
                    <th
                      onClick={() => handleSort("subsystem_id")}
                      className="px-4 py-2.5 cursor-pointer hover:bg-slate-800/80 hover:text-slate-200 transition-colors group"
                    >
                      <div className="flex items-center gap-1.5">
                        <span>TSM</span>
                        {renderSortIndicator("subsystem_id")}
                      </div>
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {sortedFrames.map((frame: CANFrame, idx: number) => (
                    <tr key={idx} className="hover:bg-slate-900/50 transition-colors">
                      <td className="px-4 py-2 text-slate-400">{frame.timestamp_ms} ms</td>
                      <td className="px-4 py-2">
                        {frame.direction === "T" ? (
                          <span className="inline-flex items-center gap-0.5 text-blue-400 font-bold">
                            <ArrowUpRight className="w-3 h-3" /> TX
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-0.5 text-emerald-400 font-bold">
                            <ArrowDownLeft className="w-3 h-3" /> RX
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-2 font-bold text-slate-100">{frame.can_id}</td>
                      <td className="px-4 py-2 text-slate-400">{frame.dlc}</td>
                      <td className="px-4 py-2">
                        {frame.payload.length > 0 ? (
                          <div className="flex gap-1">
                            {frame.payload.map((byte, bIdx) => (
                              <span
                                key={bIdx}
                                className="bg-slate-900 border border-slate-800 px-1.5 py-0.5 rounded text-amber-300 font-bold text-[11px]"
                              >
                                {byte}
                              </span>
                            ))}
                          </div>
                        ) : (
                          <span className="text-slate-600">-</span>
                        )}
                      </td>
                      <td className="px-4 py-2">
                        {frame.is_heartbeat ? (
                          <span className="inline-flex items-center gap-1 text-[11px] text-amber-400 bg-amber-950/40 px-2 py-0.5 rounded border border-amber-900/50">
                            <Heart className="w-2.5 h-2.5" /> Heartbeat
                          </span>
                        ) : (
                          <span className="text-slate-400 text-[11px]">Control / Datos</span>
                        )}
                      </td>
                      <td className="px-4 py-2 text-indigo-300 text-[11px]">
                        #1 Tubo & Generador
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <pre className="p-4 text-xs font-mono text-slate-300 max-h-[500px] overflow-auto whitespace-pre-wrap leading-relaxed">
              {content}
            </pre>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
