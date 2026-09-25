"use client"

import React from "react"
import { HardwareDashboard } from "@/components/HardwareDashboard"
import { useApp } from "@/context/AppContext"

export default function HomePage() {
  const { selectedServerId } = useApp()
  return <HardwareDashboard serverId={selectedServerId} />
}
