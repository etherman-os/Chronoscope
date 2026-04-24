import React, { useState, useCallback } from 'react';
import { SessionList } from '../components/SessionList';
import { VideoPlayer } from '../components/VideoPlayer';
import { EventTimeline } from '../components/EventTimeline';
import { Session, SessionDetail } from '../types/session';
import { getSession } from '../api/client';

export const Dashboard: React.FC = () => {
  const [selectedSession, setSelectedSession] = useState<SessionDetail | null>(null);
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
      setError('Failed to load session details');
    } finally {
      setLoading(false);
    }
  }, []);

  const handleTimeUpdate = useCallback((timeMs: number) => {
    setCurrentTime(timeMs);
  }, []);

  return (
    <div
      style={{
        display: 'flex',
        height: '100vh',
        fontFamily: 'system-ui, sans-serif',
      }}
    >
      <SessionList onSelect={handleSelect} />

      <div
        style={{
          flex: 1,
          backgroundColor: '#f5f5f5',
          padding: '24px',
          overflowY: 'auto',
        }}
      >
        {!selectedSession && !loading && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: '#666',
              fontSize: '18px',
            }}
          >
            Select a session to view replay
          </div>
        )}

        {loading && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: '#666',
              fontSize: '18px',
            }}
          >
            Loading session details...
          </div>
        )}

        {error && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: '#e74c3c',
              fontSize: '18px',
            }}
          >
            {error}
          </div>
        )}

        {selectedSession && !loading && (
          <div>
            <h2
              style={{
                margin: '0 0 16px 0',
                fontSize: '20px',
                fontWeight: 600,
                color: '#333',
              }}
            >
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
