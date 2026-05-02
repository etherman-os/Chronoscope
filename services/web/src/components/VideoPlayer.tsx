import React, { useEffect, useRef, useState } from "react";
import { getSessionVideoUrl } from "../api/client";
import { SessionEvent } from "../types/session";
import styles from "./VideoPlayer.module.css";

interface VideoPlayerProps {
  sessionId: string;
  status: string;
  events: SessionEvent[];
  onTimeUpdate?: (timeMs: number) => void;
}

export const VideoPlayer: React.FC<VideoPlayerProps> = ({
  sessionId,
  status,
  events,
  onTimeUpdate,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [currentTime, setCurrentTime] = useState(0);
  const [videoUrl, setVideoUrl] = useState<string | null>(null);
  const [loadingVideo, setLoadingVideo] = useState(false);
  const [videoError, setVideoError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    let objectUrl: string | null = null;

    if (status !== "ready") {
      setVideoUrl(null);
      return;
    }

    setLoadingVideo(true);
    setVideoError(null);
    getSessionVideoUrl(sessionId)
      .then((url) => {
        objectUrl = url;
        if (active) {
          setVideoUrl(url);
        } else if (URL.revokeObjectURL) {
          URL.revokeObjectURL(url);
        }
      })
      .catch((err) => {
        console.error("Failed to load session video:", err);
        if (active) {
          setVideoError("Processed video is not available yet.");
        }
      })
      .finally(() => {
        if (active) {
          setLoadingVideo(false);
        }
      });

    return () => {
      active = false;
      if (objectUrl && URL.revokeObjectURL) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [sessionId, status]);

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
        {status !== "ready" && (
          <div className={styles.statePanel}>
            <div className={styles.stateTitle}>Video processing</div>
            <div className={styles.stateText}>
              Session status is <strong>{status}</strong>. The processor will
              publish replay video when the session becomes ready.
            </div>
          </div>
        )}

        {status === "ready" && loadingVideo && (
          <div className={styles.statePanel}>Loading processed video...</div>
        )}

        {status === "ready" && videoError && (
          <div className={styles.statePanel}>{videoError}</div>
        )}

        {status === "ready" && videoUrl && (
          <video
            ref={videoRef}
            controls
            onTimeUpdate={handleTimeUpdate}
            className={styles.video}
            src={videoUrl}
          >
            Your browser does not support the video tag.
          </video>
        )}

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
