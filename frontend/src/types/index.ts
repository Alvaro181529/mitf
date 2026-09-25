// Tipos y Modelos TypeScript alineados con el Backend en Go (MITFV2)

export interface ServerInfo {
  host: string
  port: number
  user: string
  id?: number
  name?: string
  is_active?: boolean
}

export interface TubeHealthData {
  accumulated_mas: number
  limit_nominal_mas: number
  usage_percentage: number
  anode_temperature_celsius: number
  anode_thermal_capacity_mhu: number
  status: "NORMAL" | "WARNING" | "CRITICAL" | string
}

export interface GantryStatsData {
  accumulated_revolutions: number
  limit_revolutions: number
  usage_percentage: number
  status: "OPERATIONAL" | "WARNING" | "CRITICAL" | string
}

export interface TSMSubsystemHealth {
  id: number
  name: string
  associated_processes?: string[]
  status: "NORMAL" | "WARNING" | "CRITICAL" | "OK" | string
  active_warnings?: number
  active_alerts_count?: number
  last_alert_message?: string
}

export interface HardwareSummaryData {
  tube_rx: TubeHealthData
  gantry_rotor: GantryStatsData
  subsystems?: TSMSubsystemHealth[]
  tsm_subsystems_health?: TSMSubsystemHealth[]
  active_warnings_count?: number
}

// -------------------------------------------------------------
// Rutinas de Calentamiento de Tubo (Warm-up Routines)
// -------------------------------------------------------------
export interface WarmupRoutineCycle {
  id: number
  routine_type: "COLD_TUBE_WARMUP" | "WARMUP_II" | "FAST_CALIBRATION" | "SKIPPED_WARMUP" | string
  status: "COMPLETED" | "INCOMPLETE" | string
  start_time: string
  start_time_iso?: string
  start_epoch?: number
  end_time?: string
  end_time_iso?: string
  end_epoch?: number
  duration_seconds: number
  initial_temp_celsius: number
  final_temp_celsius: number
  temp_rise_celsius: number
  process: string
  start_ermes_code: string
  end_ermes_code: string
  start_sr_id: string
  end_sr_id: string
  details?: string
}

export interface SkippedWarmupEvent {
  sr_id: string
  date: string
  timestamp_iso?: string
  epoch?: number
  exam_id: string
  max_ma_120kv: number
  max_ma_140kv: number
  severity: string
  ermes_code: string
  process: string
  message: string
}

export interface WarmupSummary {
  total_completed_routines: number
  total_skipped_events: number
  last_routine_date?: string
  last_routine_date_iso?: string
  last_initial_temp_celsius: number
  last_final_temp_celsius: number
  last_temp_rise_celsius: number
  average_temp_rise_celsius: number
  average_duration_seconds: number
  tube_thermal_status: "READY" | "NEEDS_WARMUP" | "OVERHEATED" | string
  compliance_status: "COMPLIANT" | "ATTENTION_REQUIRED" | string
  latest_skipped_event?: SkippedWarmupEvent
}

export interface TubeWarmupResponse {
  status: string
  timestamp: string
  server_info?: ServerInfo
  summary: WarmupSummary
  routines: WarmupRoutineCycle[]
  skipped_events: SkippedWarmupEvent[]
}

// -------------------------------------------------------------
// Bus CAN (jedi_can_at_error.log)
// -------------------------------------------------------------
export interface CANFrame {
  timestamp_ms: number
  direction: "T" | "R" | string
  can_id: string
  dlc: number
  payload: string[]
  is_heartbeat: boolean
  subsystem_id: number
}

export interface JediCanLogsResponse {
  status: string
  timestamp: string
  server_info?: ServerInfo
  file: string
  lines: number
  subsystem_id: number
  subsystem_name: string
  total_frames: number
  frames: CANFrame[]
  content: string
}

export interface InteractiveNodeComponent {
  node_id: string
  display_name?: string
  label?: string
  subsystem_code?: string
  tsm_id?: string | number
  status_color?: string
  color_hex?: string
  status: "OK" | "WARNING" | "CRITICAL" | string
  metrics?: Record<string, any>
  active_alerts?: string[]
  active_alerts_count?: number
  latest_error?: string
  tooltip_info?: {
    recommended_action?: string
    technical_description?: string
  }
}

export type InteractiveMapComponent = InteractiveNodeComponent

export interface InteractiveMapResponse {
  timestamp: string
  system_status: "OK" | "WARNING" | "CRITICAL" | string
  components: InteractiveMapComponent[]
  server_info?: ServerInfo
}

export interface PostRepairQA {
  z_alignment_passed: boolean
  phantom_iq_passed: boolean
  detector_flatfield_calibrated: boolean
  qa_sign_off_by?: string
  resolution_type?: string
}

export interface TicketATREC {
  ticket_id: string
  created_at: string
  updated_at: string
  tsm_subsystem_id: number
  severity?: "WARNING" | "MAJOR" | "CRITICAL" | string
  source_error_code: string
  description: string
  assigned_technician: string
  status: "OPEN" | "IN_PROGRESS" | "PENDING_CALIBRATION" | "QA_VERIFIED" | "CLOSED"
  post_repair_qa: PostRepairQA
}

export interface LogEntry {
  sr_id?: string
  date: string
  timestamp_formatted?: string
  timestamp_epoch?: number
  ermes_code: string
  severity: string
  severity_code: number
  host: string
  process: string
  source_file: string
  source_line: number
  message: string
  device?: string
  serial_number?: string
  tube_health_index?: string
  accumulated_scans?: number
  slice_count?: number
  tube_heat_load?: string
}

export interface LogPagination {
  current_page: number
  total_pages: number
  total_records: number
  limit: number
  has_next: boolean
  has_prev: boolean
}

export interface PaginatedLogsResponse {
  data: LogEntry[]
  pagination: LogPagination
  server_info?: ServerInfo
}

export interface SSHTestResponse {
  success: boolean
  message: string
  data?: {
    server_info?: string
    ssh_host?: string
    ssh_port?: number
    ssh_user?: string
    server_id?: number
    server_name?: string
  }
  raw_output?: string
  server_info?: {
    host?: string
    port?: number
    user?: string
  }
}

export interface ServerConfig {
  id: number
  name: string
  ssh_host: string
  ssh_port: number
  ssh_user: string
  ssh_password?: string
  ssh_key_path?: string
  ssh_key_passphrase?: string
  remote_log_path: string
  server_port?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface ATRECInitialData {
  subsystem_id: number
  severity?: "WARNING" | "MAJOR" | "CRITICAL" | string
  error_code: string
  description: string
  source_log_id?: string
}
