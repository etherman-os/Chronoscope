export interface Session {
  id: string;
  user_id: string;
  duration_ms: number;
  status: string;
  created_at: string;
  metadata?: Record<string, unknown>;
}

export interface SessionEvent {
  event_type: string;
  timestamp_ms: number;
  x: number;
  y: number;
  target: string;
  payload: string;
}

export interface SessionDetail {
  session: Session;
  events: SessionEvent[];
}
