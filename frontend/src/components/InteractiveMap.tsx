"use client"

import React, { useState, useEffect } from "react"
import { InteractiveMapResponse, InteractiveNodeComponent } from "@/types"
import { fetchInteractiveMap } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog } from "@/components/ui/dialog"
import {
  MapPin,
  RefreshCw,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Eye,
  Activity,
  Layers,
  Zap,
  Sliders,
  AlertOctagon,
} from "lucide-react"

interface InteractiveMapProps {
  serverId?: number
}

export function InteractiveMap({ serverId }: InteractiveMapProps) {
  const [data, setData] = useState<InteractiveMapResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedNode, setSelectedNode] = useState<InteractiveNodeComponent | null>(null)

  const loadMapData = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await fetchInteractiveMap(serverId)
      setData(res)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error cargando esquema interactivo"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadMapData()
  }, [serverId])

  const getNodeIcon = (nodeId: string) => {
    switch (nodeId) {
      case "node_tube_xray":
        return <Zap className="w-5 h-5 text-amber-400" />
      case "node_gantry_rotor":
        return <Activity className="w-5 h-5 text-blue-400" />
      case "node_couch_table":
        return <Sliders className="w-5 h-5 text-emerald-400" />
      case "node_das_detector":
        return <Layers className="w-5 h-5 text-purple-400" />
      case "node_estop_console":
        return <AlertOctagon className="w-5 h-5 text-rose-400" />
      default:
        return <Activity className="w-5 h-5 text-slate-400" />
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "OK":
        return (
          <Badge variant="success" className="gap-1">
            <CheckCircle2 className="w-3 h-3" /> NORMAL (OK)
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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900/80 p-5 rounded-xl border border-slate-800">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
            <MapPin className="h-5 w-5 text-blue-400" />
            Esquema Interactivo de Fallas y Salud de Componentes (GE CT99)
          </h2>
          <p className="text-sm text-slate-400 mt-1">
            Monitoreo topológico y estado físico de los 5 subsistemas del tomógrafo en tiempo real
          </p>
        </div>
        <div className="flex items-center gap-3">
          {data && (
            <div className="flex items-center gap-2 text-xs font-mono bg-slate-950 px-3 py-1.5 rounded-lg border border-slate-800">
              <span className="text-slate-400">Estado Global:</span>
              <span
                className={`font-bold px-2 py-0.5 rounded text-[11px] ${
                  data.system_status === "CRITICAL"
                    ? "bg-rose-950 text-rose-400 border border-rose-800"
                    : data.system_status === "WARNING"
                    ? "bg-amber-950 text-amber-400 border border-amber-800"
                    : "bg-emerald-950 text-emerald-400 border border-emerald-800"
                }`}
              >
                {data.system_status}
              </span>
            </div>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={loadMapData}
            disabled={loading}
            className="gap-1.5"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Actualizar
          </Button>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center p-12 text-slate-400">
          <RefreshCw className="h-6 w-6 animate-spin mr-2 text-blue-500" />
          <span>Consultando mapa de componentes físicos...</span>
        </div>
      ) : error ? (
        <Card className="border-rose-900 bg-rose-950/20 p-6 text-center text-rose-300">
          {error}
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {data?.components.map((comp) => {
            const colorBorder =
              comp.color_hex ||
              (comp.status === "CRITICAL"
                ? "#EF4444"
                : comp.status === "WARNING"
                ? "#F59E0B"
                : "#10B981")

            return (
              <Card
                key={comp.node_id}
                onClick={() => setSelectedNode(comp)}
                className="border-slate-800 bg-slate-900/80 hover:bg-slate-850 transition-all cursor-pointer group hover:scale-[1.01] hover:border-slate-700 relative overflow-hidden"
              >
                <div
                  className="absolute top-0 left-0 right-0 h-1"
                  style={{ backgroundColor: colorBorder }}
                />
                <CardHeader className="pb-3 pt-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <div className="p-2 rounded-lg bg-slate-950 border border-slate-800 group-hover:border-slate-700">
                        {getNodeIcon(comp.node_id)}
                      </div>
                      <div>
                        <CardTitle className="text-sm font-semibold text-slate-100 group-hover:text-blue-300 transition-colors">
                          {comp.label || comp.display_name || comp.node_id}
                        </CardTitle>
                        <CardDescription className="text-xs text-slate-400 font-mono mt-0.5">
                          TSM #{comp.tsm_id || comp.subsystem_code || "-"}
                        </CardDescription>
                      </div>
                    </div>
                    {getStatusBadge(comp.status)}
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {/* Métricas destacadas */}
                  {comp.metrics && (
                    <div className="grid grid-cols-1 gap-1.5 text-xs bg-slate-950/60 p-2.5 rounded-lg border border-slate-800/80 font-mono">
                      {Object.entries(comp.metrics).map(([key, val]) => (
                        <div key={key} className="flex items-center justify-between">
                          <span className="text-slate-400 capitalize text-[11px]">
                            {key.replace(/_/g, " ")}:
                          </span>
                          <span className="text-slate-200 font-medium">{String(val)}</span>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Último error o alertas */}
                  {comp.latest_error && (
                    <div className="text-[11px] text-rose-300/90 bg-rose-950/30 p-2 rounded border border-rose-900/40 line-clamp-2">
                      ⚠️ {comp.latest_error}
                    </div>
                  )}

                  <div className="flex items-center justify-between pt-2 border-t border-slate-800/80 text-xs text-slate-400">
                    <span>
                      Alertas activas:{" "}
                      <strong
                        className={
                          (comp.active_alerts_count || 0) > 0
                            ? "text-amber-400"
                            : "text-emerald-400"
                        }
                      >
                        {comp.active_alerts_count || (comp.active_alerts?.length ?? 0)}
                      </strong>
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        setSelectedNode(comp)
                      }}
                      className="h-6 px-2 text-xs gap-1 text-blue-400 hover:text-blue-300"
                    >
                      <Eye className="w-3 h-3" /> Ver Detalle
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      {/* Modal de Detalle del Nodo */}
      <Dialog
        open={!!selectedNode}
        onClose={() => setSelectedNode(null)}
        title={selectedNode?.label || selectedNode?.display_name || "Detalle de Componente Físico"}
        description={`Nodo: ${selectedNode?.node_id} • Subsistema TSM #${selectedNode?.tsm_id || selectedNode?.subsystem_code || ""}`}
      >
        {selectedNode && (
          <div className="space-y-4">
            <div className="flex items-center justify-between bg-slate-950 p-3 rounded-lg border border-slate-800">
              <span className="text-xs text-slate-400">Estado de Operación:</span>
              {getStatusBadge(selectedNode.status)}
            </div>

            {selectedNode.metrics && (
              <div className="space-y-2">
                <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                  Métricas Operativas del Nodo
                </h4>
                <div className="grid grid-cols-1 gap-2">
                  {Object.entries(selectedNode.metrics).map(([metricName, val]) => (
                    <div
                      key={metricName}
                      className="flex items-center justify-between p-2.5 rounded bg-slate-800/60 text-sm border border-slate-700/50"
                    >
                      <span className="text-slate-400 capitalize">
                        {metricName.replace(/_/g, " ")}
                      </span>
                      <span className="font-mono font-semibold text-slate-100">
                        {String(val)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {selectedNode.latest_error && (
              <div className="p-3 rounded-lg bg-rose-950/40 border border-rose-800/60 text-rose-300 text-xs">
                <div className="font-semibold mb-1 flex items-center gap-1.5">
                  <AlertTriangle className="w-3.5 h-3.5" /> Último Mensaje de Excepción Registrado:
                </div>
                <div className="font-mono whitespace-pre-wrap">{selectedNode.latest_error}</div>
              </div>
            )}

            <div className="flex justify-end pt-2 border-t border-slate-800">
              <Button size="sm" onClick={() => setSelectedNode(null)}>
                Cerrar
              </Button>
            </div>
          </div>
        )}
      </Dialog>
    </div>
  )
}
