"use client"

import React from "react"
import { ServerManager } from "@/components/ServerManager"
import { useApp } from "@/context/AppContext"

export default function ServersPage() {
  const { selectedServerId, setSelectedServerId, loadServerInfo } = useApp()

  return (
    <ServerManager
      currentServerId={selectedServerId}
      onServerSelected={(id) => {
        setSelectedServerId(id)
        loadServerInfo()
      }}
    />
  )
}
