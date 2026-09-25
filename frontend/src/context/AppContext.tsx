"use client"

import React, { createContext, useContext, useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { ATRECInitialData, ServerConfig } from "@/types"
import { fetchActiveServerConfig, fetchServerConfigs } from "@/lib/api"

interface AppContextType {
  selectedServerId: number | undefined
  setSelectedServerId: (id?: number) => void
  serverList: ServerConfig[]
  activeServer: ServerConfig | null
  loadServerInfo: () => Promise<void>
  pendingATRECData: ATRECInitialData | null
  setPendingATRECData: (data: ATRECInitialData | null) => void
  autoFillAndNavigateToATREC: (data: ATRECInitialData) => void
  sidebarCollapsed: boolean
  setSidebarCollapsed: React.Dispatch<React.SetStateAction<boolean>>
  mobileMenuOpen: boolean
  setMobileMenuOpen: React.Dispatch<React.SetStateAction<boolean>>
}

const AppContext = createContext<AppContextType | undefined>(undefined)

export function AppProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [selectedServerId, setSelectedServerId] = useState<number | undefined>(undefined)
  const [serverList, setServerList] = useState<ServerConfig[]>([])
  const [activeServer, setActiveServer] = useState<ServerConfig | null>(null)
  const [pendingATRECData, setPendingATRECData] = useState<ATRECInitialData | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  const loadServerInfo = async () => {
    try {
      const [activeRes, listRes] = await Promise.all([
        fetchActiveServerConfig().catch(() => null),
        fetchServerConfigs().catch(() => null),
      ])
      if (activeRes?.data) {
        setActiveServer(activeRes.data)
      }
      if (listRes?.data) {
        setServerList(listRes.data)
      }
    } catch {
      // Ignorar fallback temporal
    }
  }

  useEffect(() => {
    loadServerInfo()
  }, [])

  const autoFillAndNavigateToATREC = (data: ATRECInitialData) => {
    setPendingATRECData(data)
    router.push("/atrec")
  }

  return (
    <AppContext.Provider
      value={{
        selectedServerId,
        setSelectedServerId,
        serverList,
        activeServer,
        loadServerInfo,
        pendingATRECData,
        setPendingATRECData,
        autoFillAndNavigateToATREC,
        sidebarCollapsed,
        setSidebarCollapsed,
        mobileMenuOpen,
        setMobileMenuOpen,
      }}
    >
      {children}
    </AppContext.Provider>
  )
}

export function useApp() {
  const context = useContext(AppContext)
  if (!context) {
    throw new Error("useApp must be used within an AppProvider")
  }
  return context
}
