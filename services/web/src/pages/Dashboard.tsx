import React, { useState, useCallback } from "react";
import { SessionList } from "../components/SessionList";
import { VideoPlayer } from "../components/VideoPlayer";
import { EventTimeline } from "../components/EventTimeline";
import { Session, SessionDetail } from "../types/session";
import { getSession } from "../api/client";
import styles from "./Dashboard.module.css";

export const Dashboard: React.FC = () => {
  const [selectedSession, setSelectedSession] = useState<SessionDetail | null>(
    null,
  );
  const [currentTime, setCurrentTime] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSelect = useCallback(async (session: Session) => {
    try {
      setLoading(true);
      setError(null);
      const detail = await getSession(session.id);
      setSelectedSession(detail);
      setCurrentTime(0);
    } catch (err) {
      console.error("Failed to load session details:", err);
      setError("Failed to load session details");
    } finally {
      setLoading(false);
    }
  }, []);

  const handleTimeUpdate = useCallback((timeMs: number) => {
    setCurrentTime(timeMs);
  }, []);

  return (
    <div className={styles.container}>
      <SessionList onSelect={handleSelect} />

      <div className={styles.main}>
        {!selectedSession && !loading && (
          <div className={styles.centerMessage}>
            Select a session to view replay
          </div>
        )}

        {loading && (
          <div className={styles.centerMessage}>Loading session details...</div>
        )}

        {error && (
          <div className={`${styles.centerMessage} ${styles.error}`}>
            {error}
          </div>
        )}

        {selectedSession && !loading && (
          <div>
            <h2 className={styles.title}>
              Session: {selectedSession.session.id}
            </h2>

            <VideoPlayer
              sessionId={selectedSession.session.id}
              events={selectedSession.events}
              onTimeUpdate={handleTimeUpdate}
            />

            <EventTimeline
              events={selectedSession.events}
              currentTime={currentTime}
            />
          </div>
        )}
      </div>
    </div>
  );
};
