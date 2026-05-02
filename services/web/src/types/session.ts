export interface Session {
  id: string;
  user_id: string;
  duration_ms?: number;
  event_count?: number;
  error_count?: number;
  video_path?: string;
  status: string;
  created_at: string;
  completed_at?: string;
  metadata?: Record<string, unknown>;
}

export interface SessionEvent {
  id?: string;
  event_type: string;
  timestamp_ms: number;
  x: number;
  y: number;
  target: string;
  payload: string | Record<string, unknown> | null;
}

export interface SessionDetail {
  session: Session;
  events: SessionEvent[];
}
