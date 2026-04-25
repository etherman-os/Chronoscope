import React, { useEffect, useState } from "react";
import { Session } from "../types/session";
import { listSessions } from "../api/client";
import styles from "./SessionList.module.css";

interface SessionListProps {
  onSelect: (session: Session) => void;
}

const PROJECT_ID = import.meta.env.VITE_PROJECT_ID || "";
if (!PROJECT_ID) {
  throw new Error("VITE_PROJECT_ID is required");
}

export const SessionList: React.FC<SessionListProps> = ({ onSelect }) => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSessions = async () => {
      try {
        setLoading(true);
        const data = await listSessions(PROJECT_ID);
        setSessions(data);
      } catch (err) {
        console.error("Failed to load sessions:", err);
        setError("Failed to load sessions");
      } finally {
        setLoading(false);
      }
    };

    fetchSessions();
  }, []);

  const formatDate = (dateStr: string): string => {
    const date = new Date(dateStr);
    return date.toLocaleString();
  };

  const formatDuration = (ms: number): string => {
    const seconds = Math.round(ms / 1000);
    return `${seconds}s`;
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>Sessions</div>

      <div className={styles.scrollArea}>
        {loading && <div className={styles.loading}>Loading...</div>}

        {error && <div className={styles.error}>{error}</div>}

        {!loading && !error && sessions.length === 0 && (
          <div className={styles.empty}>No sessions found</div>
        )}

        {sessions.map((session) => (
          <div
            key={session.id}
            onClick={() => onSelect(session)}
            role="button"
            tabIndex={0}
            aria-label={`Session for ${session.user_id || "Anonymous"}`}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onSelect(session);
              }
            }}
            className={styles.row}
          >
            <div className={styles.userId}>
              {session.user_id || "Anonymous"}
            </div>
            <div className={styles.meta}>
              <span>{formatDate(session.created_at)}</span>
              <span>{formatDuration(session.duration_ms)}</span>
            </div>
            <div className={styles.status}>Status: {session.status}</div>
          </div>
        ))}
      </div>
    </div>
  );
};
