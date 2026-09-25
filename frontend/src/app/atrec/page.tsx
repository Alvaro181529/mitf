"use client"

import React from "react"
import { ATRECModule } from "@/components/ATRECModule"
import { useApp } from "@/context/AppContext"

export default function ATRECPage() {
  const { pendingATRECData, setPendingATRECData } = useApp()

  return (
    <ATRECModule
      initialTicketData={pendingATRECData}
      onClearInitialData={() => setPendingATRECData(null)}
    />
  )
}
