import React from 'react';
import { SessionEvent } from '../types/session';

interface EventTimelineProps {
  events: SessionEvent[];
  currentTime: number;
}

export const EventTimeline: React.FC<EventTimelineProps> = ({
  events,
  currentTime,
}) => {
  const maxTime = Math.max(
    ...events.map((e) => e.timestamp_ms),
    currentTime,
    1
  );

  const currentPosition = (currentTime / maxTime) * 100;

  return (
    <div
      style={{
        height: '60px',
        backgroundColor: '#e0e0e0',
        borderRadius: '4px',
        position: 'relative',
        overflow: 'hidden',
        width: '100%',
        maxWidth: '960px',
      }}
    >
      {events.map((event, index) => {
        const position = (event.timestamp_ms / maxTime) * 100;
        const isClick = event.event_type === 'click';
        return (
          <div
            key={`${event.timestamp_ms}-${index}`}
            style={{
              position: 'absolute',
              left: `${position}%`,
              top: isClick ? '0' : '50%',
              width: '4px',
              height: '50%',
              backgroundColor: isClick ? '#e74c3c' : '#3498db',
              transform: 'translateX(-50%)',
              borderRadius: '2px',
              cursor: 'pointer',
            }}
            title={`${event.event_type} — ${event.timestamp_ms}ms`}
          />
        );
      })}

      <div
        style={{
          position: 'absolute',
          left: `${currentPosition}%`,
          top: 0,
          bottom: 0,
          width: '2px',
          backgroundColor: '#2ecc71',
          transform: 'translateX(-50%)',
          zIndex: 5,
        }}
      />
    </div>
  );
};
