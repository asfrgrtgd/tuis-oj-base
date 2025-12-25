export interface RunRequest {
  language: string
  source: string
  stdin?: string
  timeLimitMs?: number
  memoryLimitMb?: number
}

export interface RunFiles {
  stdout?: string
  stderr?: string
}

export interface RunStageResult {
  status?: string | { status?: string; code?: string; message?: string }
  time_ms?: number
  memory_kb?: number
  exit_code?: number
  files?: RunFiles
  message?: string
}

export interface RunResponse {
  compile?: RunStageResult
  run?: RunStageResult
  limits?: {
    timeLimitMs?: number
    memoryLimitMb?: number
  }
}

export interface QueueDepth {
  pending?: number
  processing?: number
  running?: number
  failed?: number
}
