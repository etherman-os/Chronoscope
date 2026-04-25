import React, { useRef, useState } from "react";
import { SessionEvent } from "../types/session";
import styles from "./VideoPlayer.module.css";

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

  const handleTimeUpdate = () => {
    if (videoRef.current) {
      const timeMs = videoRef.current.currentTime * 1000;
      setCurrentTime(timeMs);
      onTimeUpdate?.(timeMs);
    }
  };

  const visibleEvents = events.filter(
    (event) =>
      event.timestamp_ms >= currentTime - 500 &&
      event.timestamp_ms <= currentTime + 500,
  );

  return (
    <div className={styles.container}>
      <div className={styles.wrapper}>
        <video
          ref={videoRef}
          controls
          onTimeUpdate={handleTimeUpdate}
          className={styles.video}
        >
          {/* <source src={`/v1/sessions/${sessionId}/video`} type="video/mp4" /> */}
          Your browser does not support the video tag.
        </video>

        {visibleEvents.map((event) => {
          const isClick = event.event_type === "click";
          return (
            <div
              key={
                event.id ||
                `${event.event_type}-${event.timestamp_ms}-${event.x}-${event.y}`
              }
              className={`${styles.marker} ${isClick ? styles.clickMarker : styles.scrollMarker}`}
              style={
                {
                  "--x": `${event.x}px`,
                  "--y": `${event.y}px`,
                } as React.CSSProperties
              }
              title={`${event.event_type} at ${event.timestamp_ms}ms`}
            />
          );
        })}

        <div className={styles.label}>Session: {sessionId}</div>
      </div>
    </div>
  );
};
