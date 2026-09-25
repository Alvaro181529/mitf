"use client"

import React from "react"
import { TubeWarmupViewer } from "@/components/TubeWarmupViewer"
import { useApp } from "@/context/AppContext"

export default function WarmupPage() {
  const { selectedServerId, autoFillAndNavigateToATREC } = useApp()

  return (
    <TubeWarmupViewer
      serverId={selectedServerId}
      onAutoFillATREC={autoFillAndNavigateToATREC}
    />
  )
}
