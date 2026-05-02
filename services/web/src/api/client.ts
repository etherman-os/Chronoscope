import axios from "axios";
import { Session, SessionDetail } from "../types/session";

const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:8080/v1";
const ANALYTICS_BASE =
  import.meta.env.VITE_ANALYTICS_URL || "http://localhost:8081/v1";

function getApiKey(): string {
  return localStorage.getItem("chronoscope_api_key") || "";
}

export const client = axios.create({
  baseURL: API_BASE,
  timeout: 10000,
  withCredentials: true,
  headers: {
    "Content-Type": "application/json",
  },
});

export const analyticsClient = axios.create({
  baseURL: ANALYTICS_BASE,
  timeout: 10000,
  withCredentials: true,
  headers: {
    "Content-Type": "application/json",
  },
});

client.interceptors.request.use((config) => {
  const apiKey = getApiKey();
  if (apiKey) {
    config.headers["X-API-Key"] = apiKey;
  }
  return config;
});

analyticsClient.interceptors.request.use((config) => {
  const apiKey = getApiKey();
  if (apiKey) {
    config.headers["X-API-Key"] = apiKey;
  }
  return config;
});

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      console.error(`API error ${error.response.status}:`, error.response.data);
    } else {
      console.error("API request failed:", error.message);
    }
    return Promise.reject(error);
  },
);

analyticsClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      console.error(
        `Analytics API error ${error.response.status}:`,
        error.response.data,
      );
    } else {
      console.error("Analytics API request failed:", error.message);
    }
    return Promise.reject(error);
  },
);

export const listSessions = async (projectId: string): Promise<Session[]> => {
  const response = await client.get("/sessions", {
    params: { project_id: projectId },
  });
  return response.data.sessions as Session[];
};

export const getSession = async (sessionId: string): Promise<SessionDetail> => {
  const response = await client.get(`/sessions/${sessionId}`);
  const data = response.data as SessionDetail;
  data.events = data.events.map((event) => ({
    ...event,
    id:
      event.id ||
      `${event.event_type}-${event.timestamp_ms}-${event.x}-${event.y}`,
  }));
  return data;
};

export const getSessionVideoUrl = async (sessionId: string): Promise<string> => {
  const response = await client.get(`/sessions/${sessionId}/video`, {
    responseType: "blob",
  });
  return URL.createObjectURL(response.data as Blob);
};

export interface SessionStats {
  avg_duration_ms: number;
  total_sessions: number;
  total_events: number;
  avg_events_per_session: number;
}

export interface HeatmapPoint {
  x: number;
  y: number;
  count: number;
}

export interface FunnelStage {
  stage: string;
  count: number;
}

export interface AuditLog {
  id: number;
  action: string;
  actor: string;
  details: string | Record<string, unknown> | null;
  created_at: string;
}

export interface UserExport {
  user_id: string;
  total_events: number;
  page_size: number;
  sessions: Array<Record<string, unknown>>;
}

export const getSessionStats = async (): Promise<SessionStats> => {
  const response = await analyticsClient.get("/analytics/sessions/stats");
  return response.data.stats as SessionStats;
};

export const getHeatmap = async (): Promise<HeatmapPoint[]> => {
  const response = await analyticsClient.get("/analytics/heatmap");
  return response.data.points as HeatmapPoint[];
};

export const getFunnel = async (): Promise<FunnelStage[]> => {
  const response = await analyticsClient.get("/analytics/funnel");
  return response.data.funnel as FunnelStage[];
};

export const listAuditLogs = async (): Promise<AuditLog[]> => {
  const response = await client.get("/gdpr/audit-logs", {
    params: { limit: 10, offset: 0 },
  });
  return response.data.logs as AuditLog[];
};

export const exportUserData = async (userId: string): Promise<UserExport> => {
  const response = await client.post(`/gdpr/export/${encodeURIComponent(userId)}`);
  return response.data as UserExport;
};

export const deleteUserData = async (
  userId: string,
): Promise<{ deleted_sessions: number; deleted_events: number }> => {
  const response = await client.delete(`/gdpr/delete/${encodeURIComponent(userId)}`);
  return response.data as { deleted_sessions: number; deleted_events: number };
};
