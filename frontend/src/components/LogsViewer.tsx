"use client"

import React, { useState, useEffect, useMemo } from "react"
import { LogEntry, LogPagination, ATRECInitialData } from "@/types"
import { fetchLogs } from "@/lib/api"
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input, Select } from "@/components/ui/input"
import { Dialog } from "@/components/ui/dialog"
import {
  FileText,
  Search,
  RefreshCw,
  AlertTriangle,
  ShieldAlert,
  Info,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  ChevronDown,
  ArrowUpDown,
  Eye,
  Copy,
  Check,
  Wrench,
  Terminal,
} from "lucide-react"

export const HOST_FRIENDLY_NAMES: Record<string, { label: string; subsystemId: number }> = {
  tubemgr: { label: "Tubo RX / Alta Tensión", subsystemId: 1 },
  scanRx: { label: "Recepción de Disparo RX", subsystemId: 1 },
  rotmgr: { label: "Rotor / Motor del Gantry", subsystemId: 2 },
  tgp: { label: "Mesa del Paciente / Couch", subsystemId: 3 },
  tablemgr: { label: "Mesa / Posicionamiento", subsystemId: 3 },
  dasmgr: { label: "Adquisición de Datos DAS", subsystemId: 4 },
  dms: { label: "Detectores y Offset", subsystemId: 4 },
  gscb: { label: "Consola / Botones de Paro", subsystemId: 5 },
  scanmgr: { label: "Gestión de Escaneo / E-Stop", subsystemId: 5 },
  visScript: { label: "Script de Visualización", subsystemId: 5 },
  imagecreate: { label: "Reconstrucción de Imagen", subsystemId: 4 },
  Svc_Notepad: { label: "Notas del Sistema / Svc", subsystemId: 5 },
  orp_subsys: { label: "Subsistema Óptico / ORP", subsystemId: 4 },
  fwmgr: { label: "Gestor de Firmware", subsystemId: 5 },
}

type SortField = "sr_id" | "severity" | "date" | "host" | "ermes_code" | "message"
type SortOrder = "asc" | "desc" | null

interface LogsViewerProps {
  serverId?: number
  onAutoFillATREC?: (data: ATRECInitialData) => void
}

export function LogsViewer({ serverId, onAutoFillATREC }: LogsViewerProps) {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [pagination, setPagination] = useState<LogPagination>({
    current_page: 1,
    total_pages: 1,
    total_records: 0,
    limit: 50,
    has_next: false,
    has_prev: false,
  })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Filtros
  const [severityFilter, setSeverityFilter] = useState("")
  const [hostFilter, setHostFilter] = useState("")
  const [searchQuery, setSearchQuery] = useState("")
  const [page, setPage] = useState(1)

  // Ordenamiento interactivo por cabeceras
  const [sortField, setSortField] = useState<SortField | null>(null)
  const [sortOrder, setSortOrder] = useState<SortOrder>(null)

  // Modal Detalle
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null)
  const [copied, setCopied] = useState(false)

  const loadLogs = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await fetchLogs({
        server_id: serverId,
        severity: severityFilter || undefined,
        host: hostFilter || undefined,
        search: searchQuery || undefined,
        page,
        limit: 50,
      })
      setLogs(res.data || [])
      setPagination(res.pagination)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error cargando logs"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadLogs()
  }, [serverId, severityFilter, hostFilter, page])

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setPage(1)
    loadLogs()
  }

  const handleSort = (field: SortField) => {
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

  // Lista ordenada en memoria
  const sortedLogs = useMemo(() => {
    if (!sortField || !sortOrder) return logs

    return [...logs].sort((a, b) => {
      let valA: string | number = ""
      let valB: string | number = ""

      switch (sortField) {
        case "sr_id": {
          const numA = parseInt((a.sr_id || "").replace(/\D/g, ""), 10) || 0
          const numB = parseInt((b.sr_id || "").replace(/\D/g, ""), 10) || 0
          return sortOrder === "asc" ? numA - numB : numB - numA
        }
        case "severity": {
          valA = a.severity_code ?? 0
          valB = b.severity_code ?? 0
          return sortOrder === "asc" ? (valA as number) - (valB as number) : (valB as number) - (valA as number)
        }
        case "date": {
          valA = a.timestamp_epoch ?? (a.timestamp_formatted || a.date ? new Date(a.timestamp_formatted || a.date).getTime() : 0)
          valB = b.timestamp_epoch ?? (b.timestamp_formatted || b.date ? new Date(b.timestamp_formatted || b.date).getTime() : 0)
          return sortOrder === "asc" ? (valA as number) - (valB as number) : (valB as number) - (valA as number)
        }
        case "host": {
          const nameA = HOST_FRIENDLY_NAMES[a.host]?.label || a.host || ""
          const nameB = HOST_FRIENDLY_NAMES[b.host]?.label || b.host || ""
          return sortOrder === "asc"
            ? nameA.localeCompare(nameB)
            : nameB.localeCompare(nameA)
        }
        case "ermes_code": {
          valA = a.ermes_code || ""
          valB = b.ermes_code || ""
          return sortOrder === "asc"
            ? String(valA).localeCompare(String(valB))
            : String(valB).localeCompare(String(valA))
        }
        case "message": {
          valA = a.message || ""
          valB = b.message || ""
          return sortOrder === "asc"
            ? String(valA).localeCompare(String(valB))
            : String(valB).localeCompare(String(valA))
        }
        default:
          return 0
      }
    })
  }, [logs, sortField, sortOrder])

  const renderSortIndicator = (field: SortField) => {
    if (sortField !== field || !sortOrder) {
      return <ArrowUpDown className="w-3.5 h-3.5 text-slate-500 opacity-60 group-hover:opacity-100 transition-opacity" />
    }
    if (sortOrder === "asc") {
      return <ChevronUp className="w-3.5 h-3.5 text-blue-400 font-bold" />
    }
    return <ChevronDown className="w-3.5 h-3.5 text-blue-400 font-bold" />
  }

  const getSubsystemIdFromHost = (host: string): number => {
    if (!host) return 1
    const match = HOST_FRIENDLY_NAMES[host]
    if (match) return match.subsystemId
    const lower = host.toLowerCase()
    if (lower.includes("tube") || lower.includes("hvg")) return 1
    if (lower.includes("rot") || lower.includes("gantry")) return 2
    if (lower.includes("table") || lower.includes("couch") || lower.includes("tgp")) return 3
    if (lower.includes("das") || lower.includes("det") || lower.includes("dms")) return 4
    if (lower.includes("gscb") || lower.includes("scanmgr") || lower.includes("stop")) return 5
    return 1
  }

  const getHostDisplayName = (host: string): string => {
    const match = HOST_FRIENDLY_NAMES[host]
    return match ? `${match.label} (${host})` : host || "Desconocido"
  }

  const handleCopyJson = () => {
    if (!selectedLog) return
    navigator.clipboard.writeText(JSON.stringify(selectedLog, null, 2))
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleCreateATRECFromLog = () => {
    if (!selectedLog) return
    const subId = getSubsystemIdFromHost(selectedLog.host)
    const errCode = selectedLog.ermes_code || ""

    // Mapear severidad del log a severidad ATREC
    let mappedSeverity: "WARNING" | "MAJOR" | "CRITICAL" = "WARNING"
    if (selectedLog.severity_code === 3 || selectedLog.severity?.includes("Fatal")) {
      mappedSeverity = "CRITICAL"
    } else if (selectedLog.severity_code === 2 || selectedLog.severity?.includes("Soft")) {
      mappedSeverity = "MAJOR"
    }

    const desc = `[Falla detectada en ${selectedLog.sr_id || "Registro"}] Host: ${getHostDisplayName(
      selectedLog.host
    )} (${selectedLog.host}).
Severidad: ${selectedLog.severity || "Nivel " + selectedLog.severity_code}.
Mensaje: ${selectedLog.message || "Sin descripción adicional"}${
      selectedLog.source_file ? `\nOrigen: ${selectedLog.source_file}:${selectedLog.source_line}` : ""
    }${selectedLog.serial_number ? `\nDispositivo/Serial: ${selectedLog.device || ""} - ${selectedLog.serial_number}` : ""}`

    if (onAutoFillATREC) {
      onAutoFillATREC({
        subsystem_id: subId,
        severity: mappedSeverity,
        error_code: errCode,
        description: desc,
        source_log_id: selectedLog.sr_id || undefined,
      })
    }
  }

  const getSeverityBadge = (severity: string, code: number) => {
    if (code === 3 || severity?.includes("Fatal")) {
      return (
        <Badge variant="destructive" className="gap-1 font-mono text-[10px]">
          <ShieldAlert className="w-3 h-3" /> FATAL / ERROR
        </Badge>
      )
    }
    if (code === 2 || severity?.includes("Soft") || severity?.includes("Pri/")) {
      return (
        <Badge variant="warning" className="gap-1 font-mono text-[10px]">
          <AlertTriangle className="w-3 h-3" /> ADVERTENCIA
        </Badge>
      )
    }
    return (
      <Badge variant="secondary" className="gap-1 font-mono text-[10px]">
        <Info className="w-3 h-3" /> INFO (Nivel {code})
      </Badge>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900/80 p-5 rounded-xl border border-slate-800">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
            <FileText className="h-5 w-5 text-blue-400" />
            Visor de Eventos y Diagnóstico GE CT99
          </h2>
          <p className="text-sm text-slate-400 mt-1">
            Exploración de registros de errores, alertas y advertencias de los 5 subsistemas de la máquina
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={loadLogs} disabled={loading} className="gap-1.5">
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Recargar
        </Button>
      </div>

      {/* Barra de Filtros y Búsqueda */}
      <Card className="border-slate-800 bg-slate-900/60 p-4">
        <form onSubmit={handleSearchSubmit} className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
            <Input
              type="text"
              placeholder="Buscar por mensaje, error o SR ID..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 bg-slate-950/60"
            />
          </div>

          <div>
            <Select
              value={severityFilter}
              onChange={(e) => {
                setSeverityFilter(e.target.value)
                setPage(1)
              }}
              className="bg-slate-950/60"
            >
              <option value="">Todas las Severidades</option>
              <option value="3">Nivel 3 (Fatal / Errores Críticos)</option>
              <option value="2">Nivel 2 (Advertencias / Pri/Soft)</option>
              <option value="1">Nivel 1 (Información / Info)</option>
            </Select>
          </div>

          <div>
            <Select
              value={hostFilter}
              onChange={(e) => {
                setHostFilter(e.target.value)
                setPage(1)
              }}
              className="bg-slate-950/60"
            >
              <option value="">Todos los Subsistemas / Hosts</option>
              <option value="tubemgr">tubemgr (Tubo RX / HVG)</option>
              <option value="scanRx">scanRx (Disparo RX)</option>
              <option value="rotmgr">rotmgr (Rotor del Gantry)</option>
              <option value="tgp">tgp (Mesa Paciente)</option>
              <option value="tablemgr">tablemgr (Posicionamiento)</option>
              <option value="dasmgr">dasmgr (Adquisición DAS)</option>
              <option value="dms">dms (Detectores)</option>
              <option value="gscb">gscb (Consola y Paro)</option>
              <option value="scanmgr">scanmgr (Escaneo)</option>
              <option value="imagecreate">imagecreate (Reconstrucción)</option>
              <option value="orp_subsys">orp_subsys (Óptica)</option>
              <option value="fwmgr">fwmgr (Firmware)</option>
            </Select>
          </div>
        </form>
      </Card>

      {/* Tabla de Logs */}
      {loading ? (
        <div className="flex justify-center p-12 text-slate-400">
          <RefreshCw className="h-6 w-6 animate-spin mr-2 text-blue-500" />
          <span>Extrayendo y parseando registros desde el servidor tomógrafo...</span>
        </div>
      ) : error ? (
        <Card className="border-rose-900 bg-rose-950/20 p-6 text-center text-rose-300">
          {error}
        </Card>
      ) : logs.length === 0 ? (
        <Card className="border-slate-800 bg-slate-900/40 p-8 text-center text-slate-400">
          No se encontraron registros de eventos con los filtros aplicados.
        </Card>
      ) : (
        <Card className="border-slate-800 bg-slate-900/80 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-slate-300">
              <thead className="bg-slate-950/80 text-xs uppercase text-slate-400 border-b border-slate-800 select-none">
                <tr>
                  {/* Registro */}
                  <th
                    onClick={() => handleSort("sr_id")}
                    className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                  >
                    <div className="flex items-center gap-1.5">
                      <span>Registro</span>
                      {renderSortIndicator("sr_id")}
                    </div>
                  </th>

                  {/* Severidad */}
                  <th
                    onClick={() => handleSort("severity")}
                    className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                  >
                    <div className="flex items-center gap-1.5">
                      <span>Severidad</span>
                      {renderSortIndicator("severity")}
                    </div>
                  </th>

                  {/* Fecha / Hora */}
                  <th
                    onClick={() => handleSort("date")}
                    className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                  >
                    <div className="flex items-center gap-1.5">
                      <span>Fecha / Hora</span>
                      {renderSortIndicator("date")}
                    </div>
                  </th>

                  {/* Componente / Host */}
                  <th
                    onClick={() => handleSort("host")}
                    className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                  >
                    <div className="flex items-center gap-1.5">
                      <span>Componente / Host</span>
                      {renderSortIndicator("host")}
                    </div>
                  </th>

                  {/* Ermes # */}
                  <th
                    onClick={() => handleSort("ermes_code")}
                    className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                  >
                    <div className="flex items-center gap-1.5">
                      <span>Ermes #</span>
                      {renderSortIndicator("ermes_code")}
                    </div>
                  </th>

                  {/* Mensaje / Detalle Técnico */}
                  <th
                    onClick={() => handleSort("message")}
                    className="px-4 py-3 cursor-pointer hover:bg-slate-900/80 hover:text-slate-200 transition-colors group"
                  >
                    <div className="flex items-center gap-1.5">
                      <span>Mensaje / Detalle Técnico</span>
                      {renderSortIndicator("message")}
                    </div>
                  </th>

                  <th className="px-4 py-3 text-right">Acción</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 font-mono text-xs">
                {sortedLogs.map((log, idx) => {
                  const friendlyInfo = HOST_FRIENDLY_NAMES[log.host]
                  const displayName = friendlyInfo ? friendlyInfo.label : log.host || "N/A"

                  return (
                    <tr
                      key={log.sr_id ? `${log.sr_id}-${idx}` : idx}
                      onClick={() => setSelectedLog(log)}
                      className="hover:bg-slate-800/50 cursor-pointer transition-colors"
                    >
                      <td className="px-4 py-2.5 font-bold text-slate-200 whitespace-nowrap">
                        {log.sr_id || `Línea #${idx + 1}`}
                      </td>
                      <td className="px-4 py-2.5 whitespace-nowrap">
                        {getSeverityBadge(log.severity, log.severity_code)}
                      </td>
                      <td className="px-4 py-2.5 text-slate-400 whitespace-nowrap">
                        {log.timestamp_formatted || log.date}
                      </td>
                      <td className="px-4 py-2.5 text-blue-300 whitespace-nowrap" title={log.host}>
                        {displayName}
                      </td>
                      <td className="px-4 py-2.5 text-amber-300 font-bold whitespace-nowrap">
                        {log.ermes_code || "-"}
                      </td>
                      <td className="px-4 py-2.5 text-slate-300 max-w-md truncate font-sans text-xs">
                        {log.message}
                      </td>
                      <td className="px-4 py-2.5 text-right whitespace-nowrap">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation()
                            setSelectedLog(log)
                          }}
                          className="h-7 px-2 text-xs text-blue-400 hover:text-blue-300"
                        >
                          <Eye className="w-3.5 h-3.5 mr-1" /> Ver
                        </Button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>

          {/* Paginación */}
          <div className="flex flex-col sm:flex-row items-center justify-between gap-3 p-4 bg-slate-950/60 border-t border-slate-800 text-xs text-slate-400">
            <div>
              Mostrando página <strong>{pagination.current_page}</strong> de{" "}
              <strong>{pagination.total_pages}</strong> ({pagination.total_records.toLocaleString()} registros totales)
              {sortField && (
                <span className="ml-2 text-blue-400 font-mono text-[11px]">
                  (Ordenado por {sortField} {sortOrder === "asc" ? "▲ Asc" : "▼ Desc"})
                </span>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.has_prev || loading}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="gap-1 h-8"
              >
                <ChevronLeft className="w-4 h-4" /> Anterior
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={!pagination.has_next || loading}
                onClick={() => setPage((p) => p + 1)}
                className="gap-1 h-8"
              >
                Siguiente <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Modal de Detalle del Log */}
      <Dialog
        open={!!selectedLog}
        onClose={() => setSelectedLog(null)}
        title={selectedLog?.sr_id || "Detalle de Evento GE CT99"}
        description={`Código Ermes: ${selectedLog?.ermes_code || "N/A"} • Host: ${selectedLog?.host || "N/A"}`}
      >
        {selectedLog && (
          <div className="space-y-4">
            <div className="flex items-center justify-between bg-slate-950 p-3 rounded-lg border border-slate-800">
              <span className="text-xs text-slate-400">Nivel de Severidad:</span>
              {getSeverityBadge(selectedLog.severity, selectedLog.severity_code)}
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs">
              <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800/80">
                <span className="text-slate-500 block mb-1">Fecha y Hora</span>
                <span className="font-mono text-slate-200">
                  {selectedLog.timestamp_formatted || selectedLog.date}
                </span>
              </div>
              <div className="bg-slate-950/60 p-3 rounded-lg border border-slate-800/80">
                <span className="text-slate-500 block mb-1">Subsistema Mapeado</span>
                <span className="font-medium text-blue-300">
                  {getHostDisplayName(selectedLog.host)}
                </span>
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                Mensaje de Diagnóstico / Detalle
              </label>
              <div className="bg-slate-950 p-3 rounded-lg border border-slate-800 font-mono text-xs text-slate-200 whitespace-pre-wrap max-h-40 overflow-y-auto">
                {selectedLog.message || "Sin mensaje técnico disponible"}
              </div>
            </div>

            {(selectedLog.source_file || selectedLog.device) && (
              <div className="bg-slate-950/40 p-3 rounded-lg border border-slate-800/60 space-y-1 text-xs font-mono text-slate-400">
                {selectedLog.source_file && (
                  <div>
                    <span className="text-slate-500">Origen:</span> {selectedLog.source_file}:
                    {selectedLog.source_line}
                  </div>
                )}
                {selectedLog.process && (
                  <div>
                    <span className="text-slate-500">Proceso:</span> {selectedLog.process}
                  </div>
                )}
                {selectedLog.serial_number && (
                  <div>
                    <span className="text-slate-500">Serie / Dev:</span> {selectedLog.device} (S/N:{" "}
                    {selectedLog.serial_number})
                  </div>
                )}
              </div>
            )}

            <div className="flex flex-col sm:flex-row items-center justify-between gap-3 pt-4 border-t border-slate-800">
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopyJson}
                className="w-full sm:w-auto gap-1.5 text-xs"
              >
                {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                {copied ? "Copiado" : "Copiar JSON"}
              </Button>

              <div className="flex items-center gap-2 w-full sm:w-auto">
                {onAutoFillATREC && (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={handleCreateATRECFromLog}
                    className="w-full sm:w-auto gap-1.5 text-xs bg-blue-600 hover:bg-blue-500 text-white"
                  >
                    <Wrench className="w-3.5 h-3.5" />
                    Auto-completar en ATREC
                  </Button>
                )}
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setSelectedLog(null)}
                  className="w-full sm:w-auto text-xs"
                >
                  Cerrar
                </Button>
              </div>
            </div>
          </div>
        )}
      </Dialog>
    </div>
  )
}
