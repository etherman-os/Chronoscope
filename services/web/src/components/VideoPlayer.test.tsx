import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { VideoPlayer } from "./VideoPlayer";

describe("VideoPlayer", () => {
  it("renders video element and session label", () => {
    render(
      <VideoPlayer
        sessionId="sess-1"
        events={[]}
        onTimeUpdate={() => {}}
      />,
    );
    expect(screen.getByText("Session: sess-1")).toBeInTheDocument();
    expect(document.querySelector("video")).toBeInTheDocument();
  });

  it("calls onTimeUpdate on time update", () => {
    const handleTimeUpdate = vi.fn();
    render(
      <VideoPlayer
        sessionId="sess-1"
        events={[]}
        onTimeUpdate={handleTimeUpdate}
      />,
    );
    const video = document.querySelector("video")!;
    fireEvent.timeUpdate(video);
    expect(handleTimeUpdate).toHaveBeenCalled();
  });

  it("renders visible event markers", () => {
    const events = [
      {
        id: "e1",
        event_type: "click",
        timestamp_ms: 1000,
        x: 50,
        y: 60,
        target: "btn",
        payload: "",
      },
    ];
    render(
      <VideoPlayer
        sessionId="sess-1"
        events={events}
        onTimeUpdate={() => {}}
      />,
    );
    const video = document.querySelector("video")!;
    (video as HTMLVideoElement).currentTime = 1; // 1s = 1000ms
    fireEvent.timeUpdate(video);
    const marker = screen.getByTitle(/click.*1000ms/i);
    expect(marker).toBeInTheDocument();
  });
});
