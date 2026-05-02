import React, { useEffect, useState, useCallback } from "react";
import { SessionList } from "../components/SessionList";
import { VideoPlayer } from "../components/VideoPlayer";
import { EventTimeline } from "../components/EventTimeline";
import { Session, SessionDetail } from "../types/session";
import {
  AuditLog,
  FunnelStage,
  HeatmapPoint,
  SessionStats,
  deleteUserData,
  exportUserData,
  getFunnel,
  getHeatmap,
  getSession,
  getSessionStats,
  listAuditLogs,
} from "../api/client";
import styles from "./Dashboard.module.css";

interface DashboardProps {
  projectId: string;
}

export const Dashboard: React.FC<DashboardProps> = ({ projectId }) => {
  const [selectedSession, setSelectedSession] = useState<SessionDetail | null>(
    null,
  );
  const [currentTime, setCurrentTime] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<SessionStats | null>(null);
  const [heatmap, setHeatmap] = useState<HeatmapPoint[]>([]);
  const [funnel, setFunnel] = useState<FunnelStage[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [adminMessage, setAdminMessage] = useState<string | null>(null);
  const [adminBusy, setAdminBusy] = useState(false);

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

  useEffect(() => {
    if (!selectedSession || selectedSession.session.status === "ready") {
      return;
    }

    const sessionId = selectedSession.session.id;
    const id = window.setInterval(async () => {
      try {
        const detail = await getSession(sessionId);
        setSelectedSession(detail);
      } catch (err) {
        console.error("Failed to refresh session details:", err);
      }
    }, 3000);

    return () => window.clearInterval(id);
  }, [selectedSession?.session.id, selectedSession?.session.status]);

  useEffect(() => {
    let active = true;

    const loadAdminData = async () => {
      try {
        const [statsData, heatmapData, funnelData, auditData] = await Promise.all([
          getSessionStats(),
          getHeatmap(),
          getFunnel(),
          listAuditLogs(),
        ]);
        if (!active) {
          return;
        }
        setStats(statsData);
        setHeatmap(heatmapData);
        setFunnel(funnelData);
        setAuditLogs(auditData);
      } catch (err) {
        console.error("Failed to load admin overview:", err);
      }
    };

    loadAdminData();
    const id = window.setInterval(loadAdminData, 15000);
    return () => {
      active = false;
      window.clearInterval(id);
    };
  }, [projectId]);

  const handleExportUser = useCallback(async () => {
    const userId = selectedSession?.session.user_id;
    if (!userId) {
      setAdminMessage("Select a session with a user ID first.");
      return;
    }

    try {
      setAdminBusy(true);
      const exported = await exportUserData(userId);
      setAdminMessage(
        `Exported ${exported.sessions.length} sessions and ${exported.total_events} events for ${userId}.`,
      );
    } catch (err) {
      console.error("Failed to export user data:", err);
      setAdminMessage("Failed to export user data.");
    } finally {
      setAdminBusy(false);
    }
  }, [selectedSession]);

  const handleDeleteUser = useCallback(async () => {
    const userId = selectedSession?.session.user_id;
    if (!userId) {
      setAdminMessage("Select a session with a user ID first.");
      return;
    }

    const confirmed = window.confirm(
      `Delete all Chronoscope data for user "${userId}" in this project?`,
    );
    if (!confirmed) {
      return;
    }

    try {
      setAdminBusy(true);
      const deleted = await deleteUserData(userId);
      setSelectedSession(null);
      setAdminMessage(
        `Deleted ${deleted.deleted_sessions} sessions and ${deleted.deleted_events} events for ${userId}.`,
      );
      setAuditLogs(await listAuditLogs());
    } catch (err) {
      console.error("Failed to delete user data:", err);
      setAdminMessage("Failed to delete user data.");
    } finally {
      setAdminBusy(false);
    }
  }, [selectedSession]);

  const formatMs = (ms?: number) => {
    if (!ms) {
      return "0s";
    }
    return `${Math.round(ms / 1000)}s`;
  };

  return (
    <div className={styles.container}>
      <SessionList onSelect={handleSelect} projectId={projectId} />

      <div className={styles.main}>
        <section className={styles.overview}>
          <div className={styles.metricCard}>
            <span>Total Sessions</span>
            <strong>{stats?.total_sessions ?? "—"}</strong>
          </div>
          <div className={styles.metricCard}>
            <span>Total Events</span>
            <strong>{stats?.total_events ?? "—"}</strong>
          </div>
          <div className={styles.metricCard}>
            <span>Avg Duration</span>
            <strong>{formatMs(stats?.avg_duration_ms)}</strong>
          </div>
          <div className={styles.metricCard}>
            <span>Heatmap Points</span>
            <strong>{heatmap.length}</strong>
          </div>
        </section>

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
              Session Replay
            </h2>
            <div className={styles.sessionMeta}>
              <span>User: {selectedSession.session.user_id || "Anonymous"}</span>
              <span>Status: {selectedSession.session.status}</span>
              <span>
                Events:{" "}
                {selectedSession.session.event_count || selectedSession.events.length}
              </span>
            </div>

            <VideoPlayer
              sessionId={selectedSession.session.id}
              status={selectedSession.session.status}
              events={selectedSession.events}
              onTimeUpdate={handleTimeUpdate}
            />

            <EventTimeline
              events={selectedSession.events}
              currentTime={currentTime}
            />
          </div>
        )}

        <section className={styles.adminGrid}>
          <div className={styles.panel}>
            <h3>Processing Funnel</h3>
            {funnel.length === 0 && <p>No funnel data yet.</p>}
            {funnel.map((stage) => (
              <div key={stage.stage} className={styles.rowMetric}>
                <span>{stage.stage.replace(/_/g, " ")}</span>
                <strong>{stage.count}</strong>
              </div>
            ))}
          </div>

          <div className={styles.panel}>
            <h3>GDPR Tools</h3>
            <p>
              Export or delete data for the user attached to the selected
              session.
            </p>
            <div className={styles.actions}>
              <button disabled={adminBusy} onClick={handleExportUser}>
                Export User Data
              </button>
              <button
                disabled={adminBusy}
                className={styles.danger}
                onClick={handleDeleteUser}
              >
                Delete User Data
              </button>
            </div>
            {adminMessage && <div className={styles.adminMessage}>{adminMessage}</div>}
          </div>

          <div className={styles.panel}>
            <h3>Audit Log</h3>
            {auditLogs.length === 0 && <p>No audit log entries yet.</p>}
            {auditLogs.map((log) => (
              <div key={log.id} className={styles.auditItem}>
                <strong>{log.action}</strong>
                <span>{new Date(log.created_at).toLocaleString()}</span>
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
};
