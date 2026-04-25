import React from "react";
import { SessionEvent } from "../types/session";
import styles from "./EventTimeline.module.css";

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
    1,
  );

  const currentPosition = (currentTime / maxTime) * 100;

  return (
    <div className={styles.container}>
      {events.map((event) => {
        const position = (event.timestamp_ms / maxTime) * 100;
        const isClick = event.event_type === "click";
        return (
          <div
            key={
              event.id ||
              `${event.event_type}-${event.timestamp_ms}-${event.x}-${event.y}`
            }
            className={isClick ? styles.click : styles.scroll}
            style={{ "--left": `${position}%` } as React.CSSProperties}
            title={`${event.event_type} — ${event.timestamp_ms}ms`}
          />
        );
      })}

      <div
        className={styles.indicator}
        style={{ "--left": `${currentPosition}%` } as React.CSSProperties}
      />
    </div>
  );
};
