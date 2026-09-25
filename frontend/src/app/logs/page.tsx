"use client"

import React from "react"
import { LogsViewer } from "@/components/LogsViewer"
import { useApp } from "@/context/AppContext"

export default function LogsPage() {
  const { selectedServerId, autoFillAndNavigateToATREC } = useApp()

  return (
    <LogsViewer
      serverId={selectedServerId}
      onAutoFillATREC={autoFillAndNavigateToATREC}
    />
  )
}
