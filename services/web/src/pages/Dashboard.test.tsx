import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { beforeEach, describe, it, expect, vi } from "vitest";
import { Dashboard } from "../pages/Dashboard";

vi.mock("../api/client", () => ({
  listSessions: vi.fn(),
  getSession: vi.fn(),
  getSessionVideoUrl: vi.fn(),
  getSessionStats: vi.fn().mockResolvedValue({
    avg_duration_ms: 1000,
    total_sessions: 1,
    total_events: 1,
    avg_events_per_session: 1,
  }),
  getHeatmap: vi.fn().mockResolvedValue([]),
  getFunnel: vi.fn().mockResolvedValue([]),
  listAuditLogs: vi.fn().mockResolvedValue([]),
  exportUserData: vi.fn(),
  deleteUserData: vi.fn(),
}));

import {
  getFunnel,
  getHeatmap,
  getSession,
  getSessionStats,
  listAuditLogs,
  listSessions,
} from "../api/client";

describe("Dashboard", () => {
  beforeEach(() => {
    vi.mocked(getSessionStats).mockResolvedValue({
      avg_duration_ms: 1000,
      total_sessions: 1,
      total_events: 1,
      avg_events_per_session: 1,
    });
    vi.mocked(getHeatmap).mockResolvedValue([]);
    vi.mocked(getFunnel).mockResolvedValue([]);
    vi.mocked(listAuditLogs).mockResolvedValue([]);
  });

  it("renders initial state", async () => {
    vi.mocked(listSessions).mockReturnValue(new Promise(() => {}));
    render(<Dashboard projectId="test-project" />);
    expect(screen.getByText(/Select a session to view replay/i)).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("Total Sessions")).toBeInTheDocument();
    });
  });

  it("shows loading and then session details on select", async () => {
    const sessions = [
      {
        id: "s1",
        user_id: "user-1",
        duration_ms: 10000,
        status: "completed",
        created_at: new Date().toISOString(),
      },
    ];
    const detail = {
      session: sessions[0],
      events: [
        {
          id: "e1",
          event_type: "click",
          timestamp_ms: 1000,
          x: 10,
          y: 20,
          target: "btn",
          payload: "",
        },
      ],
    };
    vi.mocked(listSessions).mockResolvedValue(sessions);
    vi.mocked(getSession).mockResolvedValue(detail);
    render(<Dashboard projectId="test-project" />);

    await waitFor(() => {
      expect(screen.getByText("user-1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("user-1"));

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /Session Replay/i })).toBeInTheDocument();
      expect(screen.getByText("Session: s1")).toBeInTheDocument();
    });
  });

  it("shows error when getSession fails", async () => {
    const sessions = [
      {
        id: "s1",
        user_id: "user-1",
        duration_ms: 10000,
        status: "completed",
        created_at: new Date().toISOString(),
      },
    ];
    vi.mocked(listSessions).mockResolvedValue(sessions);
    vi.mocked(getSession).mockRejectedValue(new Error("fail"));
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<Dashboard projectId="test-project" />);

    await waitFor(() => {
      expect(screen.getByText("user-1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("user-1"));

    await waitFor(() => {
      expect(screen.getByText(/Failed to load session details/i)).toBeInTheDocument();
    });
    consoleError.mockRestore();
  });
});
