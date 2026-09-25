"use client"

import React, { useState, useEffect } from "react"
import { TicketATREC, ATRECInitialData } from "@/types"
import {
  fetchATRECTickets,
  createATRECTicket,
  updateATRECCalibration,
  closeATRECTicket,
} from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input, Select } from "@/components/ui/input"
import { Dialog } from "@/components/ui/dialog"
import { formatDate } from "@/lib/utils"
import {
  ClipboardList,
  PlusCircle,
  FileCheck2,
  AlertTriangle,
  Clock,
  ShieldCheck,
  Lock,
  User,
  Wrench,
  RefreshCw,
  Sparkles,
  Flame,
  ShieldAlert,
  Info,
  CheckCircle2,
} from "lucide-react"

interface ATRECModuleProps {
  initialTicketData?: ATRECInitialData | null
  onClearInitialData?: () => void
}

// Opciones de filtro de estado
const STATUS_OPTIONS = [
  { value: "", label: "Todos los Estados" },
  { value: "OPEN", label: "Abierto" },
  { value: "IN_PROGRESS", label: "En Reparación" },
  { value: "PENDING_CALIBRATION", label: "Calibración Pendiente" },
  { value: "QA_VERIFIED", label: "QA Aprobado (Listo)" },
  { value: "CLOSED", label: "Cerrado" },
]

// Tipos de resolución post-reparación
export const RESOLUTION_TYPES = [
  { value: "REVISION_GENERAL", label: "Revisión General" },
  { value: "DIAGNOSTICO_CAMPO", label: "Diagnóstico Técnico de Campo" },
  { value: "TROUBLESHOOTING_L1", label: "Troubleshooting Nivel L1 Ejecutado" },
  { value: "ESCALADO_L2", label: "Escalado a Nivel L2 (Especialista)" },
  { value: "REQUIERE_REPUESTOS", label: "Requiere Repuestos" },
]

export function ATRECModule({ initialTicketData, onClearInitialData }: ATRECModuleProps) {
  const [tickets, setTickets] = useState<TicketATREC[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<string>("")

  // Modales
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [calibModalOpen, setCalibModalOpen] = useState(false)
  const [selectedTicket, setSelectedTicket] = useState<TicketATREC | null>(null)

  // Formulario nuevo ticket
  const [newSubsystem, setNewSubsystem] = useState(1)
  const [newSeverity, setNewSeverity] = useState<string>("WARNING")
  const [newErrorCode, setNewErrorCode] = useState("")
  const [newDescription, setNewDescription] = useState("")
  const [newTechnician, setNewTechnician] = useState("")
  const [submitting, setSubmitting] = useState(false)

  // Autocompletado desde prop externa (LogsViewer u otro)
  useEffect(() => {
    if (initialTicketData) {
      setNewSubsystem(initialTicketData.subsystem_id || 1)
      if (initialTicketData.severity) {
        setNewSeverity(initialTicketData.severity)
      }
      setNewErrorCode(initialTicketData.error_code || "")
      setNewDescription(initialTicketData.description || "")
      setNewTechnician("Técnico Biomédico de Guardia")
      setCreateModalOpen(true)
      if (onClearInitialData) {
        onClearInitialData()
      }
    }
  }, [initialTicketData, onClearInitialData])

  // Formulario calibración
  const [zAlignment, setZAlignment] = useState(false)
  const [phantomIQ, setPhantomIQ] = useState(false)
  const [flatfield, setFlatfield] = useState(false)
  const [qaSignOff, setQaSignOff] = useState("")
  const [resolutionType, setResolutionType] = useState("REVISION_GENERAL")

  // Error 412 específico
  const [rule412Error, setRule412Error] = useState<{
    message: string
    missingChecks: string[]
  } | null>(null)

  const loadTickets = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await fetchATRECTickets(statusFilter || undefined)
      setTickets(res.data || [])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error cargando bitácora ATREC"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadTickets()
  }, [statusFilter])

  const handleOpenCalibration = (t: TicketATREC) => {
    setSelectedTicket(t)
    setZAlignment(t.post_repair_qa.z_alignment_passed)
    setPhantomIQ(t.post_repair_qa.phantom_iq_passed)
    setFlatfield(t.post_repair_qa.detector_flatfield_calibrated)
    setQaSignOff(t.post_repair_qa.qa_sign_off_by || "")
    setResolutionType(t.post_repair_qa.resolution_type || "REVISION_GENERAL")
    setCalibModalOpen(true)
  }

  const applyPresetCase = (caseType: "z_only" | "iq_only" | "all_passed" | "reset") => {
    switch (caseType) {
      case "z_only":
        setZAlignment(true)
        setPhantomIQ(false)
        setFlatfield(false)
        break
      case "iq_only":
        setZAlignment(false)
        setPhantomIQ(true)
        setFlatfield(false)
        break
      case "all_passed":
        setZAlignment(true)
        setPhantomIQ(true)
        setFlatfield(true)
        break
      case "reset":
        setZAlignment(false)
        setPhantomIQ(false)
        setFlatfield(false)
        break
    }
  }

  const handleSaveCalibration = async () => {
    if (!selectedTicket) return
    try {
      setSubmitting(true)
      await updateATRECCalibration(selectedTicket.ticket_id, {
        z_alignment_passed: zAlignment,
        phantom_iq_passed: phantomIQ,
        detector_flatfield_calibrated: flatfield,
        qa_sign_off_by: qaSignOff,
        resolution_type: resolutionType,
      })
      setCalibModalOpen(false)
      loadTickets()
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Error guardando calibración")
    } finally {
      setSubmitting(false)
    }
  }

  const handleCreateTicket = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      setSubmitting(true)
      await createATRECTicket({
        tsm_subsystem_id: Number(newSubsystem),
        severity: newSeverity,
        source_error_code: newErrorCode,
        description: newDescription,
        assigned_technician: newTechnician,
      })
      setCreateModalOpen(false)
      // Limpiar form
      setNewErrorCode("")
      setNewDescription("")
      setNewTechnician("")
      setNewSeverity("WARNING")
      loadTickets()
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Error creando ticket")
    } finally {
      setSubmitting(false)
    }
  }

  const handleCloseTicket = async (t: TicketATREC) => {
    try {
      setRule412Error(null)
      await closeATRECTicket(t.ticket_id)
      loadTickets()
    } catch (err: any) {
      if (err.status === 412 || err.missing_checks) {
        setRule412Error({
          message: err.message,
          missingChecks: err.missing_checks || [
            "Pruebas de calibración post-reparación incompletas.",
          ],
        })
      } else {
        alert(err.message || "Error cerrando ticket")
      }
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "OPEN":
        return (
          <Badge variant="warning" className="gap-1">
            <Clock className="w-3 h-3" /> ABIERTO
          </Badge>
        )
      case "IN_PROGRESS":
        return (
          <Badge variant="default" className="gap-1 bg-amber-600">
            <Wrench className="w-3 h-3" /> EN REPARACIÓN
          </Badge>
        )
      case "PENDING_CALIBRATION":
        return (
          <Badge variant="secondary" className="gap-1 text-purple-300 border-purple-500/40">
            <FileCheck2 className="w-3 h-3" /> CALIBRACIÓN PENDIENTE
          </Badge>
        )
      case "QA_VERIFIED":
        return (
          <Badge variant="success" className="gap-1">
            <ShieldCheck className="w-3 h-3" /> QA APROBADO (LISTO)
          </Badge>
        )
      case "CLOSED":
        return (
          <Badge variant="outline" className="gap-1 text-slate-400">
            <Lock className="w-3 h-3" /> CERRADO
          </Badge>
        )
      default:
        return <Badge variant="secondary">{status}</Badge>
    }
  }

  const getSeverityBadge = (severity?: string) => {
    switch (severity) {
      case "CRITICAL":
        return (
          <Badge variant="destructive" className="gap-1 font-semibold text-[11px]">
            <ShieldAlert className="w-3 h-3" /> Critical (Crítica)
          </Badge>
        )
      case "MAJOR":
        return (
          <Badge variant="default" className="gap-1 bg-amber-600/90 text-amber-100 border-amber-500/40 text-[11px]">
            <Flame className="w-3 h-3" /> Major (Mayor)
          </Badge>
        )
      case "WARNING":
      default:
        return (
          <Badge variant="warning" className="gap-1 text-[11px]">
            <AlertTriangle className="w-3 h-3" /> Warning (Menor)
          </Badge>
        )
    }
  }

  const getResolutionTypeLabel = (resType?: string) => {
    const item = RESOLUTION_TYPES.find((r) => r.value === resType)
    return item ? item.label : resType || "No especificada"
  }

  const getSubsystemName = (id: number) => {
    switch (id) {
      case 1:
        return "Generador y Tubo RX"
      case 2:
        return "Gantry y Rotor"
      case 3:
        return "Mesa del Paciente (Couch)"
      case 4:
        return "Adquisición y Detectores (DAS)"
      case 5:
        return "Consola y Paros de Emergencia"
      default:
        return `Subsistema #${id}`
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900/80 p-5 rounded-xl border border-slate-800">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
            <ClipboardList className="h-5 w-5 text-purple-400" />
            Bitácora de Mantenimiento ATREC y Control de Calibración
          </h2>
          <p className="text-sm text-slate-400 mt-1">
            Atención Técnica y Registro de Calibración • Regla estricta: Ningún ticket puede cerrarse sin validación completa de QA
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button onClick={() => setCreateModalOpen(true)} className="gap-1.5 bg-purple-600 hover:bg-purple-700">
            <PlusCircle className="w-4 h-4" /> Crear Ticket ATREC
          </Button>
          <Button variant="outline" size="sm" onClick={loadTickets} disabled={loading} className="gap-1.5">
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Actualizar
          </Button>
        </div>
      </div>

      {/* Alerta de Validación 412 si falló el cierre de ticket */}
      {rule412Error && (
        <Card className="border-rose-800 bg-rose-950/40 animate-in fade-in duration-300">
          <CardContent className="p-4 flex items-start gap-3">
            <AlertTriangle className="h-6 w-6 text-rose-400 shrink-0 mt-0.5" />
            <div className="space-y-1.5">
              <div className="flex items-center gap-2">
                <span className="font-bold text-rose-300 text-sm">
                  REGLA DE NEGOCIO ATREC BLOQUEADA (HTTP 412 Precondition Failed)
                </span>
              </div>
              <p className="text-xs text-rose-200">{rule412Error.message}</p>
              <ul className="text-xs text-rose-300 list-disc list-inside space-y-0.5 mt-1 font-mono">
                {rule412Error.missingChecks.map((check, idx) => (
                  <li key={idx}>{check}</li>
                ))}
              </ul>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setRule412Error(null)}
                className="mt-2 text-xs h-7 border-rose-700 hover:bg-rose-900/40 text-rose-200"
              >
                Entendido
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Filtros de Tickets con Nombres en Español */}
      <div className="flex flex-wrap items-center justify-between gap-3 bg-slate-900/50 p-3 rounded-lg border border-slate-800">
        <div className="flex flex-wrap items-center gap-2 text-sm text-slate-300">
          <span className="font-medium text-xs text-slate-400">Filtrar por estado:</span>
          <div className="flex flex-wrap gap-1.5">
            {STATUS_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                onClick={() => setStatusFilter(opt.value)}
                className={`text-xs px-2.5 py-1 rounded-md font-medium transition-colors ${
                  statusFilter === opt.value
                    ? "bg-blue-600 text-white shadow-sm"
                    : "bg-slate-800 text-slate-400 hover:bg-slate-700 hover:text-slate-200"
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Lista de Tickets */}
      {loading ? (
        <div className="flex justify-center p-12 text-slate-400">
          <RefreshCw className="h-6 w-6 animate-spin mr-2 text-purple-500" />
          <span>Cargando tickets ATREC...</span>
        </div>
      ) : error ? (
        <Card className="border-rose-900 bg-rose-950/20 p-6 text-center text-rose-300">
          {error}
        </Card>
      ) : tickets.length === 0 ? (
        <Card className="border-slate-800 bg-slate-900/40 p-8 text-center text-slate-400">
          No hay tickets de mantenimiento registrados con este filtro.
        </Card>
      ) : (
        <div className="space-y-4">
          {tickets.map((t) => {
            const qa = t.post_repair_qa
            const isClosed = t.status === "CLOSED"

            return (
              <Card
                key={t.ticket_id}
                className={`border-slate-800 bg-slate-900/80 transition-all ${
                  t.status === "QA_VERIFIED"
                    ? "border-emerald-500/40"
                    : t.status === "CLOSED"
                    ? "opacity-60"
                    : ""
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                    <div className="flex flex-wrap items-center gap-3">
                      <span className="font-mono text-sm font-bold text-purple-400">
                        {t.ticket_id}
                      </span>
                      {getStatusBadge(t.status)}
                      {getSeverityBadge(t.severity)}
                      <span className="text-xs text-slate-400">
                        {getSubsystemName(t.tsm_subsystem_id)}
                      </span>
                    </div>
                    <div className="text-xs text-slate-400 font-mono">
                      Registrado: {formatDate(t.created_at)}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="text-sm text-slate-200">
                    <p className="font-medium whitespace-pre-wrap">{t.description}</p>
                    <div className="flex flex-wrap items-center gap-4 mt-2 text-xs text-slate-400">
                      <span className="flex items-center gap-1">
                        <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
                        Ermes: <strong className="text-slate-300 font-mono">{t.source_error_code || "N/A"}</strong>
                      </span>
                      <span className="flex items-center gap-1">
                        <User className="w-3.5 h-3.5 text-blue-400" />
                        Técnico: <strong className="text-slate-300">{t.assigned_technician || "Sin asignar"}</strong>
                      </span>
                      {qa.resolution_type && (
                        <span className="flex items-center gap-1 text-purple-300">
                          <Wrench className="w-3.5 h-3.5" />
                          Resolución: <strong className="text-purple-200">{getResolutionTypeLabel(qa.resolution_type)}</strong>
                        </span>
                      )}
                      {qa.qa_sign_off_by && (
                        <span className="flex items-center gap-1 text-emerald-300">
                          <CheckCircle2 className="w-3.5 h-3.5" />
                          QA: <strong className="text-emerald-200">{qa.qa_sign_off_by}</strong>
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Checklist QA de Calibración */}
                  <div className="bg-slate-950/70 p-3.5 rounded-lg border border-slate-800 flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div className="space-y-1.5 text-xs flex-1">
                      <div className="font-semibold text-slate-300 flex items-center gap-1.5">
                        <ShieldCheck className="w-4 h-4 text-emerald-400" />
                        Estado de Pruebas de Calibración Post-Reparación (QA Checklist)
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">1. Alineación Z (Geometría):</span>
                        {qa.z_alignment_passed ? (
                          <span className="text-emerald-400 font-medium">Aprobada</span>
                        ) : (
                          <span className="text-rose-400 font-medium">Pendiente</span>
                        )}
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">2. Phantom IQ (Ruido/HU):</span>
                        {qa.phantom_iq_passed ? (
                          <span className="text-emerald-400 font-medium">Aprobada</span>
                        ) : (
                          <span className="text-rose-400 font-medium">Pendiente</span>
                        )}
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-slate-400">3. Detector Flatfield:</span>
                        {qa.detector_flatfield_calibrated ? (
                          <span className="text-emerald-400 font-medium">Aprobada</span>
                        ) : (
                          <span className="text-rose-400 font-medium">Pendiente</span>
                        )}
                      </div>
                    </div>

                    {/* Acciones */}
                    <div className="flex sm:flex-col gap-2 shrink-0">
                      {!isClosed && (
                        <>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleOpenCalibration(t)}
                            className="gap-1 text-xs"
                          >
                            <FileCheck2 className="w-3.5 h-3.5 text-blue-400" /> Calibrar
                          </Button>
                          <Button
                            variant="success"
                            size="sm"
                            onClick={() => handleCloseTicket(t)}
                            className="gap-1 text-xs"
                          >
                            <Lock className="w-3.5 h-3.5" /> Cerrar Ticket
                          </Button>
                        </>
                      )}
                      {isClosed && (
                        <div className="text-center text-xs text-slate-500 font-mono py-1">
                          Equipo en Servicio
                        </div>
                      )}
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      {/* Modal: Nuevo Ticket ATREC */}
      <Dialog
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title="Registrar Nuevo Ticket de Falla Técnica (ATREC)"
        description="Apertura formal de mantenimiento con seguimiento de ciclo de vida"
      >
        <form onSubmit={handleCreateTicket} className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-slate-400 mb-1 block">Subsistema Físico TSM</label>
              <Select
                value={newSubsystem}
                onChange={(e) => setNewSubsystem(Number(e.target.value))}
              >
                <option value={1}>1 - Generador y Tubo RX</option>
                <option value={2}>2 - Gantry y Rotor</option>
                <option value={3}>3 - Mesa del Paciente (Couch)</option>
                <option value={4}>4 - Adquisición y Detectores (DAS)</option>
                <option value={5}>5 - Consola y Paro de Emergencia</option>
              </Select>
            </div>

            <div>
              <label className="text-xs text-slate-400 mb-1 block">Nivel de Severidad *</label>
              <Select
                value={newSeverity}
                onChange={(e) => setNewSeverity(e.target.value)}
              >
                <option value="WARNING">Warning (Menor)</option>
                <option value="MAJOR">Major (Mayor)</option>
                <option value="CRITICAL">Critical (Crítica)</option>
              </Select>
            </div>
          </div>

          <div>
            <label className="text-xs text-slate-400 mb-1 block">Código de Error / Ermes #</label>
            <Input
              placeholder="Ej: 260199001 o 260114056"
              value={newErrorCode}
              onChange={(e) => setNewErrorCode(e.target.value)}
            />
          </div>

          <div>
            <label className="text-xs text-slate-400 mb-1 block">Descripción de la Falla Técnica *</label>
            <textarea
              required
              rows={3}
              placeholder="Detalle de anomalía, síntomas o pieza a sustituir..."
              className="w-full rounded-md border border-slate-700 bg-slate-800 p-2 text-sm text-slate-100"
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
            />
          </div>

          <div>
            <label className="text-xs text-slate-400 mb-1 block">Técnico Asignado *</label>
            <Input
              required
              placeholder="Nombre del Ingeniero o Especialista de Servicio"
              value={newTechnician}
              onChange={(e) => setNewTechnician(e.target.value)}
            />
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t border-slate-800">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setCreateModalOpen(false)}
            >
              Cancelar
            </Button>
            <Button type="submit" size="sm" disabled={submitting}>
              {submitting ? "Creando..." : "Crear Ticket"}
            </Button>
          </div>
        </form>
      </Dialog>

      {/* Modal: Calibración y Pruebas QA */}
      <Dialog
        open={calibModalOpen}
        onClose={() => setCalibModalOpen(false)}
        title={`Registro de Calibración Post-Reparación • ${selectedTicket?.ticket_id}`}
        description="Marque las pruebas obligatorias ejecutadas exitosamente antes de cerrar el ticket"
      >
        <div className="space-y-4">
          {/* Presets Rápidos de Casos de Calibración */}
          <div className="bg-slate-950 p-3 rounded-lg border border-slate-800">
            <div className="flex items-center gap-1.5 text-xs text-purple-400 font-semibold mb-2">
              <Sparkles className="w-3.5 h-3.5" /> Casos Rápidos de Simulación / Prueba:
            </div>
            <div className="flex flex-wrap gap-1.5">
              <button
                type="button"
                onClick={() => applyPresetCase("z_only")}
                className="text-[11px] px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-colors"
                title="Caso: Solo Z Aprobada, IQ y Flatfield Pendientes (Gatilla 412)"
              >
                Caso 1: Z Aprobada (IQ y Flatfield Pendientes)
              </button>
              <button
                type="button"
                onClick={() => applyPresetCase("iq_only")}
                className="text-[11px] px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-colors"
                title="Caso: Solo IQ Aprobada (Gatilla 412)"
              >
                Caso 2: Solo Phantom IQ
              </button>
              <button
                type="button"
                onClick={() => applyPresetCase("all_passed")}
                className="text-[11px] px-2.5 py-1 rounded bg-emerald-950/60 hover:bg-emerald-900/60 text-emerald-300 border border-emerald-700/60 transition-colors"
                title="Caso: Todas Aprobadas (Permite Cierre 200 OK)"
              >
                Caso 3: Todas Aprobadas (QA OK)
              </button>
              <button
                type="button"
                onClick={() => applyPresetCase("reset")}
                className="text-[11px] px-2.5 py-1 rounded bg-rose-950/60 hover:bg-rose-900/60 text-rose-300 border border-rose-800/60 transition-colors"
                title="Resetear todas a Pendientes"
              >
                Resetear Todo
              </button>
            </div>
          </div>

          {/* Checklist de las 3 pruebas QA obligatorias */}
          <div className="space-y-3 bg-slate-950 p-4 rounded-lg border border-slate-800">
            <div className="text-xs font-semibold text-slate-300 flex items-center gap-1.5 mb-1">
              <Info className="w-3.5 h-3.5 text-blue-400" />
              Pruebas Técnicas Obligatorias de Calibración:
            </div>

            {/* 1. Alineación Z */}
            <label className="flex items-start gap-3 cursor-pointer p-2 rounded hover:bg-slate-900/60 transition-colors">
              <input
                type="checkbox"
                checked={zAlignment}
                onChange={(e) => setZAlignment(e.target.checked)}
                className="h-4 w-4 mt-1 rounded border-slate-700 bg-slate-800 text-blue-600 focus:ring-blue-500"
              />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-slate-200 block">
                    1. Alineación Geométrica del Eje Z
                  </span>
                  <Badge variant={zAlignment ? "success" : "secondary"} className="text-[10px] py-0">
                    {zAlignment ? "Aprobada" : "Pendiente"}
                  </Badge>
                </div>
                <span className="text-xs text-slate-400 block mt-0.5">
                  Laser alignment and collimator position verified within ±0.5mm
                </span>
              </div>
            </label>

            {/* 2. Phantom IQ */}
            <label className="flex items-start gap-3 cursor-pointer p-2 rounded hover:bg-slate-900/60 transition-colors">
              <input
                type="checkbox"
                checked={phantomIQ}
                onChange={(e) => setPhantomIQ(e.target.checked)}
                className="h-4 w-4 mt-1 rounded border-slate-700 bg-slate-800 text-blue-600 focus:ring-blue-500"
              />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-slate-200 block">
                    2. Phantom Image Quality (IQ)
                  </span>
                  <Badge variant={phantomIQ ? "success" : "secondary"} className="text-[10px] py-0">
                    {phantomIQ ? "Aprobada" : "Pendiente"}
                  </Badge>
                </div>
                <span className="text-xs text-slate-400 block mt-0.5">
                  Nivel de ruido, uniformidad y números CT (HU) en agua dentro de rango normal
                </span>
              </div>
            </label>

            {/* 3. Detector Flatfield */}
            <label className="flex items-start gap-3 cursor-pointer p-2 rounded hover:bg-slate-900/60 transition-colors">
              <input
                type="checkbox"
                checked={flatfield}
                onChange={(e) => setFlatfield(e.target.checked)}
                className="h-4 w-4 mt-1 rounded border-slate-700 bg-slate-800 text-blue-600 focus:ring-blue-500"
              />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-slate-200 block">
                    3. Calibración Detector Flatfield / Offset (ADF)
                  </span>
                  <Badge variant={flatfield ? "success" : "secondary"} className="text-[10px] py-0">
                    {flatfield ? "Aprobada" : "Pendiente"}
                  </Badge>
                </div>
                <span className="text-xs text-slate-400 block mt-0.5">
                  Compensación de ganancia y canales muertos en DAS sin artefactos de anillo
                </span>
              </div>
            </label>
          </div>

          {/* Tipo de Resolución y Firma QA */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-slate-400 mb-1 block font-medium">
                Tipo de Resolución Técnica *
              </label>
              <Select
                value={resolutionType}
                onChange={(e) => setResolutionType(e.target.value)}
              >
                {RESOLUTION_TYPES.map((rt) => (
                  <option key={rt.value} value={rt.value}>
                    {rt.label}
                  </option>
                ))}
              </Select>
            </div>

            <div>
              <label className="text-xs text-slate-400 mb-1 block font-medium">
                Firma Digital / Nombre del Responsable QA *
              </label>
              <Input
                placeholder="Ingeniero Biomédico / QA Sign-Off"
                value={qaSignOff}
                onChange={(e) => setQaSignOff(e.target.value)}
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t border-slate-800">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setCalibModalOpen(false)}
            >
              Cancelar
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={handleSaveCalibration}
              disabled={submitting}
              className="bg-purple-600 hover:bg-purple-700"
            >
              {submitting ? "Guardando..." : "Guardar Calibración"}
            </Button>
          </div>
        </div>
      </Dialog>
    </div>
  )
}
