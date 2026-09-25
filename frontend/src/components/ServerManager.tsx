"use client"

import React, { useState, useEffect } from "react"
import { ServerConfig, SSHTestResponse } from "@/types"
import {
  fetchServerConfigs,
  createServerConfig,
  activateServerConfig,
  testSSH,
} from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Dialog } from "@/components/ui/dialog"
import { formatDate } from "@/lib/utils"
import {
  Server,
  PlusCircle,
  CheckCircle2,
  Zap,
  Terminal,
  RefreshCw,
  HardDrive,
  Key,
} from "lucide-react"

interface ServerManagerProps {
  currentServerId?: number
  onServerSelected: (id?: number) => void
}

export function ServerManager({ currentServerId, onServerSelected }: ServerManagerProps) {
  const [servers, setServers] = useState<ServerConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  // Test SSH
  const [testOutput, setTestOutput] = useState<string | null>(null)
  const [testingSSH, setTestingSSH] = useState(false)

  // Formulario nuevo servidor
  const [name, setName] = useState("")
  const [host, setHost] = useState("")
  const [port, setPort] = useState(22)
  const [user, setUser] = useState("server")
  const [password, setPassword] = useState("")
  const [remoteLogPath, setRemoteLogPath] = useState("/home/server/log")

  const loadServers = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await fetchServerConfigs()
      setServers(res.data || [])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error cargando servidores"
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadServers()
  }, [])

  const handleCreateServer = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      setSubmitting(true)
      await createServerConfig({
        name,
        ssh_host: host,
        ssh_port: port,
        ssh_user: user,
        ssh_password: password,
        remote_log_path: remoteLogPath,
      })
      setModalOpen(false)
      setName("")
      setHost("")
      setPassword("")
      loadServers()
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Error guardando servidor")
    } finally {
      setSubmitting(false)
    }
  }

  const handleActivate = async (id: number) => {
    try {
      await activateServerConfig(id)
      onServerSelected(id)
      loadServers()
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Error activando servidor")
    }
  }

  const handleTestSSH = async (id?: number) => {
    try {
      setTestingSSH(true)
      setTestOutput(null)
      const res: SSHTestResponse = await testSSH(id)
      
      const serverUser = res.data?.ssh_user || "server"
      const serverHost = res.data?.ssh_host || "192.168.122.79"
      const serverPort = res.data?.ssh_port || 22
      const unameOutput = res.data?.server_info || res.raw_output

      const banner = `✓ Conexión SSH exitosa con ${serverUser}@${serverHost}:${serverPort}`
      const detail = unameOutput ? `\n\n[Diagnóstico de Linux Remoto / Uname]\n${unameOutput}` : ""

      setTestOutput(res.message ? `${res.message}${detail}` : `${banner}${detail}`)
    } catch (err: unknown) {
      setTestOutput(err instanceof Error ? err.message : "Fallo en conexión SSH")
    } finally {
      setTestingSSH(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900/80 p-5 rounded-xl border border-slate-800">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
            <Server className="h-5 w-5 text-blue-400" />
            Administrador de Servidores SSH y Tomógrafos
          </h2>
          <p className="text-sm text-slate-400 mt-1">
            Gestión dinámica multi-host para conectar y auditar múltiples equipos en red clínica
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button onClick={() => setModalOpen(true)} className="gap-1.5">
            <PlusCircle className="w-4 h-4" /> Agregar Servidor
          </Button>
          <Button variant="outline" size="sm" onClick={loadServers} disabled={loading} className="gap-1.5">
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Actualizar
          </Button>
        </div>
      </div>

      {/* Consola de Diagnóstico SSH */}
      {testOutput && (
        <Card className="border-blue-900/60 bg-blue-950/30 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold text-blue-300 flex items-center gap-1.5">
              <Terminal className="w-4 h-4 text-blue-400" /> Diagnóstico de Conexión SSH
            </span>
            <button
              onClick={() => setTestOutput(null)}
              className="text-xs text-slate-400 hover:text-slate-200"
            >
              Cerrar
            </button>
          </div>
          <pre className="text-xs font-mono text-slate-200 bg-slate-950/80 p-3 rounded overflow-x-auto whitespace-pre-wrap">
            {testOutput}
          </pre>
        </Card>
      )}

      {/* Lista de Servidores Registrados */}
      {loading ? (
        <div className="flex justify-center p-12 text-slate-400">
          <RefreshCw className="h-6 w-6 animate-spin mr-2 text-blue-500" />
          <span>Cargando configuraciones de servidores...</span>
        </div>
      ) : error ? (
        <Card className="border-rose-900 bg-rose-950/20 p-6 text-center text-rose-300">
          {error}
        </Card>
      ) : servers.length === 0 ? (
        <Card className="border-slate-800 bg-slate-900/40 p-8 text-center text-slate-400">
          No hay servidores SSH registrados en la base de datos.
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {servers.map((s) => {
            const isCurrentlySelected = currentServerId === s.id
            const isActiveInDB = s.is_active

            return (
              <Card
                key={s.id}
                className={`border-slate-800 bg-slate-900/80 transition-all ${
                  isActiveInDB
                    ? "border-emerald-500/50 shadow-md shadow-emerald-950/20"
                    : isCurrentlySelected
                    ? "border-blue-500/50"
                    : ""
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base text-slate-100 flex items-center gap-2">
                      <HardDrive className="h-4 w-4 text-blue-400" />
                      {s.name}
                    </CardTitle>
                    {isActiveInDB ? (
                      <Badge variant="success" className="gap-1">
                        <CheckCircle2 className="w-3 h-3" /> ACTIVO GLOBAL
                      </Badge>
                    ) : isCurrentlySelected ? (
                      <Badge variant="default" className="gap-1 bg-blue-600">
                        SELECCIONADO
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="text-slate-400">
                        ID #{s.id}
                      </Badge>
                    )}
                  </div>
                  <CardDescription className="font-mono text-xs text-slate-400">
                    {s.ssh_user}@{s.ssh_host}:{s.ssh_port}
                  </CardDescription>
                </CardHeader>

                <CardContent className="space-y-4">
                  <div className="text-xs space-y-1 font-mono text-slate-300 bg-slate-950/60 p-2.5 rounded border border-slate-800/60">
                    <div className="truncate">
                      <span className="text-slate-500">Ruta Logs:</span> {s.remote_log_path}
                    </div>
                    <div>
                      <span className="text-slate-500">Autenticación:</span>{" "}
                      {s.ssh_password ? "Contraseña" : s.ssh_key_path ? "Llave SSH" : "Por defecto"}
                    </div>
                    <div className="text-slate-500 text-[10px]">
                      Registrado: {formatDate(s.created_at)}
                    </div>
                  </div>

                  <div className="flex flex-wrap items-center gap-2 pt-2 border-t border-slate-800">
                    {!isActiveInDB && (
                      <Button
                        size="sm"
                        variant="default"
                        onClick={() => handleActivate(s.id)}
                        className="text-xs gap-1 flex-1 bg-emerald-600 hover:bg-emerald-700"
                      >
                        <Zap className="w-3.5 h-3.5" /> Activar
                      </Button>
                    )}
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={testingSSH}
                      onClick={() => handleTestSSH(s.id)}
                      className="text-xs gap-1 flex-1"
                    >
                      <Terminal className="w-3.5 h-3.5" /> Probar SSH
                    </Button>
                    {!isActiveInDB && !isCurrentlySelected && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => onServerSelected(s.id)}
                        className="text-xs px-2 text-blue-400 hover:text-blue-300"
                      >
                        Auditar
                      </Button>
                    )}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      {/* Modal: Agregar Servidor */}
      <Dialog
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title="Registrar Nuevo Servidor SSH / Tomógrafo"
        description="Agregue los parámetros de conexión SSH para explorar sus logs y telemetría"
      >
        <form onSubmit={handleCreateServer} className="space-y-4">
          <div>
            <label className="text-xs text-slate-400 mb-1 block">Nombre / Alias del Equipo *</label>
            <Input
              required
              placeholder="Ej: Tomógrafo GE Sala 2 (Urgencias)"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <label className="text-xs text-slate-400 mb-1 block">Dirección IP o Host SSH *</label>
              <Input
                required
                placeholder="192.168.122.80"
                value={host}
                onChange={(e) => setHost(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-slate-400 mb-1 block">Puerto SSH *</label>
              <Input
                type="number"
                required
                value={port}
                onChange={(e) => setPort(Number(e.target.value))}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-xs text-slate-400 mb-1 block">Usuario SSH *</label>
              <Input
                required
                placeholder="server"
                value={user}
                onChange={(e) => setUser(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-slate-400 mb-1 block">Contraseña SSH</label>
              <Input
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
          </div>

          <div>
            <label className="text-xs text-slate-400 mb-1 block">Ruta Remota de Logs *</label>
            <Input
              required
              placeholder="/home/server/log"
              value={remoteLogPath}
              onChange={(e) => setRemoteLogPath(e.target.value)}
            />
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t border-slate-800">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setModalOpen(false)}
            >
              Cancelar
            </Button>
            <Button type="submit" size="sm" disabled={submitting}>
              {submitting ? "Guardando..." : "Guardar Servidor"}
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  )
}
