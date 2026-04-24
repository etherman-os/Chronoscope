import React, { useEffect, useRef, useState } from 'react';
import { SessionEvent } from '../types/session';

interface VideoPlayerProps {
  sessionId: string;
  events: SessionEvent[];
  onTimeUpdate?: (timeMs: number) => void;
}

export const VideoPlayer: React.FC<VideoPlayerProps> = ({
  sessionId,
  events,
  onTimeUpdate,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [currentTime, setCurrentTime] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      if (videoRef.current) {
        const timeMs = videoRef.current.currentTime * 1000;
        setCurrentTime(timeMs);
        onTimeUpdate?.(timeMs);
      }
    }, 100);

    return () => clearInterval(interval);
  }, [onTimeUpdate]);

  const visibleEvents = events.filter(
    (event) =>
      event.timestamp_ms >= currentTime - 500 &&
      event.timestamp_ms <= currentTime + 500
  );

  return (
    <div style={{ marginBottom: '16px' }}>
      <div
        style={{
          position: 'relative',
          width: '100%',
          maxWidth: '960px',
          aspectRatio: '16 / 9',
          backgroundColor: '#000',
          borderRadius: '4px',
          overflow: 'hidden',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <video
          ref={videoRef}
          controls
          style={{
            width: '100%',
            height: '100%',
            objectFit: 'contain',
          }}
        >
          {/* <source src={`/v1/sessions/${sessionId}/video`} type="video/mp4" /> */}
          Your browser does not support the video tag.
        </video>

        {visibleEvents.map((event, index) => {
          const isClick = event.event_type === 'click';
          return (
            <div
              key={`${event.timestamp_ms}-${index}`}
              style={{
                position: 'absolute',
                left: `${event.x}px`,
                top: `${event.y}px`,
                width: '16px',
                height: '16px',
                borderRadius: '50%',
                backgroundColor: isClick ? 'rgba(231, 76, 60, 0.8)' : 'rgba(52, 152, 219, 0.8)',
                transform: 'translate(-50%, -50%)',
                pointerEvents: 'none',
                zIndex: 10,
              }}
              title={`${event.event_type} at ${event.timestamp_ms}ms`}
            />
          );
        })}

        <div
          style={{
            position: 'absolute',
            top: '16px',
            left: '16px',
            color: '#fff',
            backgroundColor: 'rgba(0, 0, 0, 0.6)',
            padding: '8px 12px',
            borderRadius: '4px',
            fontSize: '14px',
            zIndex: 20,
          }}
        >
          Session: {sessionId}
        </div>
      </div>
    </div>
  );
};
