"use client"

import React from "react"
import { useApp } from "@/context/AppContext"
import { Sidebar } from "@/components/Sidebar"
import { TopNavbar } from "@/components/TopNavbar"

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const { sidebarCollapsed } = useApp()

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col">
      {/* Sidebar Fija */}
      <Sidebar />

      {/* Contenedor Principal adaptable a la barra lateral */}
      <div
        className={`flex-1 flex flex-col transition-all duration-300 ease-in-out ${
          sidebarCollapsed ? "lg:pl-20" : "lg:pl-64"
        }`}
      >
        <TopNavbar />

        <main className="flex-1 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          {children}
        </main>
      </div>
    </div>
  )
}
