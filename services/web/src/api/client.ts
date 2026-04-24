import axios from 'axios';
import { Session, SessionDetail } from '../types/session';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/v1';
const API_KEY = import.meta.env.VITE_API_KEY;
if (!API_KEY) {
  throw new Error('VITE_API_KEY is required');
}

const client = axios.create({
  baseURL: API_BASE,
  timeout: 10000,
  headers: {
    'X-API-Key': API_KEY,
    'Content-Type': 'application/json',
  },
});

export const listSessions = async (projectId: string): Promise<Session[]> => {
  const response = await client.get('/sessions', {
    params: { project_id: projectId },
  });
  return response.data.sessions as Session[];
};

export const getSession = async (sessionId: string): Promise<SessionDetail> => {
  const response = await client.get(`/sessions/${sessionId}`);
  return response.data as SessionDetail;
};
