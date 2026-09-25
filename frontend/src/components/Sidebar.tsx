"use client"

import React from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import { useApp } from "@/context/AppContext"
import {
  Activity,
  Flame,
  MapPin,
  ClipboardList,
  FileText,
  Radio,
  Server,
  HeartPulse,
  ChevronLeft,
  ChevronRight,
  ShieldCheck,
  X,
} from "lucide-react"

export const NAV_ITEMS = [
  {
    href: "/",
    label: "Hardware & Telemetría",
    shortLabel: "Hardware",
    icon: Activity,
    description: "KPIs Tubo RX, Gantry y Subsistemas",
  },
  {
    href: "/warmup",
    label: "Calentamiento Tubo",
    shortLabel: "Warm-up",
    icon: Flame,
    description: "Rutinas térmicas y omisiones",
  },
  {
    href: "/map",
    label: "Mapa Interactivo",
    shortLabel: "Mapa TSM",
    icon: MapPin,
    description: "Topología visual y estado de nodos",
  },
  {
    href: "/atrec",
    label: "Bitácora ATREC",
    shortLabel: "ATREC",
    icon: ClipboardList,
    description: "Mantenimiento y Calibración QA",
  },
  {
    href: "/logs",
    label: "Visor de logs",
    shortLabel: "Logs",
    icon: FileText,
    description: "log y filtros Ermes",
  },
  {
    href: "/telemetry",
    label: "Buses CAN / DAS",
    shortLabel: "Telemetría",
    icon: Radio,
    description: "jedi_can.log y das_errors.log",
  },
  {
    href: "/servers",
    label: "Servidores SSH",
    shortLabel: "Servidores",
    icon: Server,
    description: "Gestión de hosts y conexiones",
  },
] as const

export function Sidebar() {
  const pathname = usePathname()
  const {
    selectedServerId,
    serverList,
    activeServer,
    sidebarCollapsed,
    setSidebarCollapsed,
    mobileMenuOpen,
    setMobileMenuOpen,
  } = useApp()

  // Determinar servidor mostrado
  const currentConfig = selectedServerId
    ? serverList.find((s) => s.id === selectedServerId) || activeServer
    : activeServer

  const serverDisplayText = currentConfig
    ? `${currentConfig.ssh_host}:${currentConfig.ssh_port}`
    : ""

  return (
    <>
      {/* Mobile Backdrop */}
      {mobileMenuOpen && (
        <div
          onClick={() => setMobileMenuOpen(false)}
          className="fixed inset-0 z-40 bg-black/70 backdrop-blur-sm lg:hidden"
        />
      )}

      {/* Main Sidebar */}
      <aside
        className={`fixed top-0 bottom-0 left-0 z-50 flex flex-col bg-slate-950 border-r border-slate-800 transition-all duration-300 ${
          sidebarCollapsed ? "w-20" : "w-64"
        } ${mobileMenuOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"}`}
      >
        {/* Brand / Logo */}
        <div className="flex items-center justify-between h-16 px-4 border-b border-slate-800/80">
          <div className="flex items-center gap-3 overflow-hidden">
            <div className="p-2 rounded-xl bg-gradient-to-br from-blue-600 to-indigo-700 text-white shadow-md shadow-blue-500/20 shrink-0">
              <HeartPulse className="h-5 w-5" />
            </div>
            {!sidebarCollapsed && (
              <div className="flex flex-col min-w-0">
                <span className="font-bold text-sm tracking-tight text-slate-100 truncate">
                  MITFV2 <span className="text-blue-400 font-mono text-xs">CT99</span>
                </span>
                <span className="text-[10px] text-slate-400 truncate">
                  Monitoreo de Tomógrafos
                </span>
              </div>
            )}
          </div>

          {/* Close button on mobile */}
          <button
            onClick={() => setMobileMenuOpen(false)}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-200 hover:bg-slate-900 lg:hidden"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Server Status Badge */}
        <div className="p-3 border-b border-slate-900">
          <div
            className={`flex items-center gap-2 p-2 rounded-lg bg-slate-900/80 border border-slate-800/80 ${
              sidebarCollapsed ? "justify-center" : "justify-between"
            }`}
            title={`Servidor Activo: ${serverDisplayText}`}
          >
            <div className="flex items-center gap-2 min-w-0">
              <span className="relative flex h-2 w-2 shrink-0">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
              </span>
              {!sidebarCollapsed && (
                <div className="flex flex-col min-w-0">
                  <span className="text-[11px] font-mono text-emerald-400 font-semibold truncate">
                    {serverDisplayText}
                  </span>
                  <span className="text-[9px] text-slate-400 truncate">
                    {currentConfig?.name || "Tomógrafo Principal"}
                  </span>
                </div>
              )}
            </div>
            {!sidebarCollapsed && (
              <ShieldCheck className="h-3.5 w-3.5 text-emerald-400/80 shrink-0" />
            )}
          </div>
        </div>

        {/* Navigation Links */}
        <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
          {NAV_ITEMS.map((item) => {
            const Icon = item.icon
            const isActive =
              item.href === "/"
                ? pathname === "/" || pathname === "/hardware"
                : pathname?.startsWith(item.href)

            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={() => setMobileMenuOpen(false)}
                className={`group flex items-center gap-3 px-3 py-2.5 rounded-xl text-xs font-medium transition-all duration-150 ${
                  isActive
                    ? "bg-blue-600/15 text-blue-400 border border-blue-500/30 shadow-sm shadow-blue-500/10 font-semibold"
                    : "text-slate-400 hover:text-slate-200 hover:bg-slate-900/80 border border-transparent"
                } ${sidebarCollapsed ? "justify-center px-2" : ""}`}
                title={sidebarCollapsed ? item.label : undefined}
              >
                <Icon
                  className={`h-4 w-4 shrink-0 transition-colors ${
                    isActive ? "text-blue-400" : "text-slate-400 group-hover:text-slate-200"
                  }`}
                />
                {!sidebarCollapsed && (
                  <div className="flex flex-col min-w-0">
                    <span className="truncate">{item.label}</span>
                  </div>
                )}
              </Link>
            )
          })}
        </nav>

        {/* Footer / Collapse Toggle button */}
        <div className="p-3 border-t border-slate-900 hidden lg:block">
          <button
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="w-full flex items-center justify-center gap-2 p-2 rounded-lg text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-900 transition-colors"
            title={sidebarCollapsed ? "Expandir barra lateral" : "Colapsar barra lateral"}
          >
            {sidebarCollapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <>
                <ChevronLeft className="h-4 w-4" />
                <span>Colapsar Menú</span>
              </>
            )}
          </button>
        </div>
      </aside>
    </>
  )
}
