"use client"

import React from "react"
import { TelemetryViewer } from "@/components/TelemetryViewer"
import { useApp } from "@/context/AppContext"

export default function TelemetryPage() {
  const { selectedServerId } = useApp()

  return <TelemetryViewer serverId={selectedServerId} />
}
