import {
  HardwareSummaryData,
  InteractiveMapResponse,
  PaginatedLogsResponse,
  TicketATREC,
  ServerConfig,
  SSHTestResponse,
  TubeWarmupResponse,
  JediCanLogsResponse,
} from "@/types"

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

function buildUrl(path: string, params?: Record<string, string | number | undefined>): string {
  const url = new URL(`${BASE_URL}${path}`)
  if (params) {
    Object.entries(params).forEach(([key, val]) => {
      if (val !== undefined && val !== null && val !== "") {
        url.searchParams.append(key, String(val))
      }
    })
  }
  return url.toString()
}

// 1. Hardware & Métricas
export async function fetchHardwareSummary(serverId?: number): Promise<{ data: HardwareSummaryData }> {
  const res = await fetch(buildUrl("/api/v1/hardware/summary", { server_id: serverId }), {
    cache: "no-store",
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando hardware summary`)
  return res.json()
}

export async function fetchInteractiveMap(serverId?: number): Promise<InteractiveMapResponse> {
  const res = await fetch(buildUrl("/api/v1/hardware/interactive-map", { server_id: serverId }), {
    cache: "no-store",
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando esquema interactivo`)
  return res.json()
}

// 1.1 Rutinas de Calentamiento de Tubo RX (Warm-up Routines)
export async function fetchTubeWarmup(
  serverId?: number,
  limit: number = 50
): Promise<TubeWarmupResponse> {
  const res = await fetch(buildUrl("/api/v1/hardware/tube-warmup", { server_id: serverId, limit }), {
    cache: "no-store",
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando rutinas de calentamiento de tubo`)
  return res.json()
}

// 2. Telemetría Buses & Detectores
export async function fetchJediCanLogs(
  serverId?: number,
  lines: number = 100
): Promise<JediCanLogsResponse> {
  const res = await fetch(buildUrl("/api/v1/logs/jedi-can", { server_id: serverId, lines }), {
    cache: "no-store",
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando jedi_can.log`)
  return res.json()
}

export async function fetchDasErrorsLogs(
  serverId?: number,
  lines: number = 100
): Promise<{ content: string }> {
  const res = await fetch(buildUrl("/api/v1/logs/das-errors", { server_id: serverId, lines }), {
    cache: "no-store",
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando das_errors.log`)
  return res.json()
}

// 3. Bitácora ATREC
export async function fetchATRECTickets(status?: string): Promise<{ data: TicketATREC[] }> {
  const res = await fetch(buildUrl("/api/v1/atrec/tickets", { status }), {
    cache: "no-store",
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando bitácora ATREC`)
  return res.json()
}

export async function createATRECTicket(data: {
  tsm_subsystem_id: number
  severity?: string
  source_error_code?: string
  description: string
  assigned_technician: string
}): Promise<{ data: TicketATREC }> {
  const res = await fetch(buildUrl("/api/v1/atrec/tickets"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `HTTP ${res.status}: Error creando ticket ATREC`)
  }
  return res.json()
}

export async function updateATRECCalibration(
  ticketId: string,
  qaData: {
    z_alignment_passed: boolean
    phantom_iq_passed: boolean
    detector_flatfield_calibrated: boolean
    qa_sign_off_by: string
    resolution_type?: string
  }
): Promise<{ data: TicketATREC; calibration_fully_passed: boolean }> {
  const res = await fetch(buildUrl(`/api/v1/atrec/tickets/${ticketId}/calibration`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(qaData),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `HTTP ${res.status}: Error actualizando calibración`)
  }
  return res.json()
}

export async function closeATRECTicket(ticketId: string): Promise<{ data: TicketATREC }> {
  const res = await fetch(buildUrl(`/api/v1/atrec/tickets/${ticketId}/close`), {
    method: "PATCH",
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    const errorObj = new Error(err.message || `HTTP ${res.status}: Error al cerrar ticket`)
    // @ts-expect-error adding custom properties
    errorObj.status = res.status
    // @ts-expect-error custom property
    errorObj.missing_checks = err.missing_checks
    throw errorObj
  }
  return res.json()
}

// 4. Logs Viewer
export async function fetchLogs(params: {
  server_id?: number
  severity?: string
  host?: string
  search?: string
  device?: string
  page?: number
  limit?: number
}): Promise<PaginatedLogsResponse> {
  const res = await fetch(buildUrl("/api/logs", params), { cache: "no-store" })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando logs`)
  return res.json()
}

// 5. Configuración de Servidores & SSH
export async function fetchServerConfigs(): Promise<{ data: ServerConfig[] }> {
  const res = await fetch(buildUrl("/api/v1/configs"), { cache: "no-store" })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando servidores`)
  return res.json()
}

export async function fetchActiveServerConfig(): Promise<{ data: ServerConfig }> {
  const res = await fetch(buildUrl("/api/v1/configs/active"), { cache: "no-store" })
  if (!res.ok) throw new Error(`HTTP ${res.status}: Error consultando servidor activo`)
  return res.json()
}

export async function createServerConfig(data: Partial<ServerConfig>): Promise<{ data: ServerConfig }> {
  const res = await fetch(buildUrl("/api/v1/configs"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `HTTP ${res.status}: Error registrando servidor`)
  }
  return res.json()
}

export async function activateServerConfig(id: number): Promise<{ message: string }> {
  const res = await fetch(buildUrl(`/api/v1/configs/${id}/activate`), {
    method: "POST",
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.message || `HTTP ${res.status}: Error activando servidor`)
  }
  return res.json()
}

export async function testSSH(serverId?: number): Promise<SSHTestResponse> {
  const res = await fetch(buildUrl("/api/ssh/test", { server_id: serverId }), {
    cache: "no-store",
  })
  return res.json()
}
