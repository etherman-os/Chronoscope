import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { Dashboard } from "../pages/Dashboard";

vi.mock("../api/client", () => ({
  listSessions: vi.fn(),
  getSession: vi.fn(),
}));

import { listSessions, getSession } from "../api/client";

describe("Dashboard", () => {
  it("renders initial state", () => {
    vi.mocked(listSessions).mockReturnValue(new Promise(() => {}));
    render(<Dashboard />);
    expect(screen.getByText(/Select a session to view replay/i)).toBeInTheDocument();
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
    render(<Dashboard />);

    await waitFor(() => {
      expect(screen.getByText("user-1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("user-1"));

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: /Session: s1/i })).toBeInTheDocument();
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
    render(<Dashboard />);

    await waitFor(() => {
      expect(screen.getByText("user-1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("user-1"));

    await waitFor(() => {
      expect(screen.getByText(/Failed to load session details/i)).toBeInTheDocument();
    });
  });
});
