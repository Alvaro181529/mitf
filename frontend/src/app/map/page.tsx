"use client"

import React from "react"
import { InteractiveMap } from "@/components/InteractiveMap"
import { useApp } from "@/context/AppContext"

export default function MapPage() {
  const { selectedServerId } = useApp()
  return <InteractiveMap serverId={selectedServerId} />
}
