import axios from "axios";
import { Session, SessionDetail } from "../types/session";

const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:8080/v1";

export const client = axios.create({
  baseURL: API_BASE,
  timeout: 10000,
  withCredentials: true,
  headers: {
    "Content-Type": "application/json",
  },
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
