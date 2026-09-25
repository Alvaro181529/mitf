"use client"

import React from "react"
import { usePathname } from "next/navigation"
import { useApp } from "@/context/AppContext"
import { NAV_ITEMS } from "@/components/Sidebar"
import { Menu, HeartPulse, RefreshCw, Server } from "lucide-react"
import Link from "next/link"

export function TopNavbar() {
  const pathname = usePathname()
  const {
    selectedServerId,
    serverList,
    activeServer,
    loadServerInfo,
    setMobileMenuOpen,
  } = useApp()

  const currentConfig = selectedServerId
    ? serverList.find((s) => s.id === selectedServerId) ||
      (activeServer?.id === selectedServerId ? activeServer : null)
    : activeServer

  const displayedHost = currentConfig?.ssh_host || "192.168.122.79"
  const displayedPort = currentConfig?.ssh_port ? `:${currentConfig.ssh_port}` : ":22"
  const displayedName = currentConfig?.name ? ` • ${currentConfig.name}` : ""

  const currentNav = NAV_ITEMS.find((item) => {
    if (item.href === "/") {
      return pathname === "/" || pathname === "/hardware"
    }
    return pathname.startsWith(item.href)
  })

  return (
    <header className="sticky top-0 z-30 h-16 w-full border-b border-slate-800/80 bg-slate-950/80 backdrop-blur-md px-4 sm:px-6 lg:px-8 flex items-center justify-between">
      <div className="flex items-center gap-3">
        {/* Toggle mobile sidebar */}
        <button
          type="button"
          onClick={() => setMobileMenuOpen(true)}
          className="lg:hidden p-2 rounded-lg text-slate-400 hover:text-slate-100 hover:bg-slate-900 border border-slate-800"
          title="Abrir menú"
        >
          <Menu className="h-5 w-5" />
        </button>

        <div>
          <h1 className="text-base sm:text-lg font-bold text-slate-100 flex items-center gap-2">
            {currentNav?.label || "Tomógrafo GE CT99"}
          </h1>
          <p className="text-xs text-slate-400 hidden sm:block">
            {currentNav?.description || "Monitoreo Biomédico y Control de Calidad"}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-3">
        {/* Indicador de Servidor Activo con Link directo a Configuración */}
        <Link
          href="/servers"
          className="flex items-center gap-2 text-xs bg-slate-900/90 hover:bg-slate-800/90 border border-slate-800 px-3 py-1.5 rounded-lg font-mono text-slate-300 transition-colors shadow-sm"
          title="Gestionar Servidores SSH"
        >
          <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
          <span className="hidden sm:inline text-slate-400">Servidor:</span>
          <strong className="text-emerald-400 font-semibold">
            {displayedHost}{displayedPort}
          </strong>
          <span className="text-slate-400 hidden md:inline">{displayedName}</span>
          <Server className="w-3.5 h-3.5 text-slate-500 ml-1" />
        </Link>
      </div>
    </header>
  )
}
