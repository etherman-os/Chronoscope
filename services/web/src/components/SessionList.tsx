import React, { useEffect, useState } from 'react';
import { Session } from '../types/session';
import { listSessions } from '../api/client';

interface SessionListProps {
  onSelect: (session: Session) => void;
}

const PROJECT_ID = '22222222-2222-2222-2222-222222222222';

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
        setError('Failed to load sessions');
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
    <div
      style={{
        width: '300px',
        height: '100%',
        backgroundColor: '#1a1a2e',
        color: '#e0e0e0',
        display: 'flex',
        flexDirection: 'column',
        borderRight: '1px solid #333',
      }}
    >
      <div
        style={{
          padding: '16px',
          borderBottom: '1px solid #333',
          fontWeight: 600,
          fontSize: '16px',
          color: '#fff',
        }}
      >
        Sessions
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {loading && (
          <div style={{ padding: '16px', textAlign: 'center', color: '#888' }}>
            Loading...
          </div>
        )}

        {error && (
          <div style={{ padding: '16px', textAlign: 'center', color: '#e74c3c' }}>
            {error}
          </div>
        )}

        {!loading && !error && sessions.length === 0 && (
          <div style={{ padding: '16px', textAlign: 'center', color: '#888' }}>
            No sessions found
          </div>
        )}

        {sessions.map((session) => (
          <div
            key={session.id}
            onClick={() => onSelect(session)}
            style={{
              padding: '12px 16px',
              borderBottom: '1px solid #2a2a40',
              cursor: 'pointer',
              transition: 'background-color 0.15s',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLDivElement).style.backgroundColor = '#252545';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLDivElement).style.backgroundColor = 'transparent';
            }}
          >
            <div style={{ fontWeight: 500, marginBottom: '4px' }}>
              {session.user_id || 'Anonymous'}
            </div>
            <div
              style={{
                fontSize: '12px',
                color: '#aaa',
                display: 'flex',
                justifyContent: 'space-between',
              }}
            >
              <span>{formatDate(session.created_at)}</span>
              <span>{formatDuration(session.duration_ms)}</span>
            </div>
            <div style={{ fontSize: '11px', color: '#888', marginTop: '4px' }}>
              Status: {session.status}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
