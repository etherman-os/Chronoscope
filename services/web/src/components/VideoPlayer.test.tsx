import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { beforeEach, describe, it, expect, vi } from "vitest";
import { VideoPlayer } from "./VideoPlayer";

vi.mock("../api/client", () => ({
  getSessionVideoUrl: vi.fn().mockResolvedValue("blob:session-video"),
}));

describe("VideoPlayer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders video element and session label", async () => {
    render(
      <VideoPlayer
        sessionId="sess-1"
        status="ready"
        events={[]}
        onTimeUpdate={() => {}}
      />,
    );
    expect(screen.getByText("Session: sess-1")).toBeInTheDocument();
    await waitFor(() => {
      expect(document.querySelector("video")).toBeInTheDocument();
    });
  });

  it("calls onTimeUpdate on time update", async () => {
    const handleTimeUpdate = vi.fn();
    render(
      <VideoPlayer
        sessionId="sess-1"
        status="ready"
        events={[]}
        onTimeUpdate={handleTimeUpdate}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector("video")).toBeInTheDocument();
    });
    const video = document.querySelector("video")!;
    fireEvent.timeUpdate(video);
    expect(handleTimeUpdate).toHaveBeenCalled();
  });

  it("renders visible event markers", async () => {
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
        status="ready"
        events={events}
        onTimeUpdate={() => {}}
      />,
    );
    await waitFor(() => {
      expect(document.querySelector("video")).toBeInTheDocument();
    });
    const video = document.querySelector("video")!;
    (video as HTMLVideoElement).currentTime = 1; // 1s = 1000ms
    fireEvent.timeUpdate(video);
    const marker = screen.getByTitle(/click.*1000ms/i);
    expect(marker).toBeInTheDocument();
  });

  it("shows processing state when video is not ready", () => {
    render(
      <VideoPlayer
        sessionId="sess-1"
        status="completed"
        events={[]}
        onTimeUpdate={() => {}}
      />,
    );
    expect(screen.getByText(/Video processing/i)).toBeInTheDocument();
  });
});
