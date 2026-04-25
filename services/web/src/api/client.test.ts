import { describe, it, expect, vi } from "vitest";
import { client, listSessions, getSession } from "./client";

describe("API client", () => {
  it("listSessions returns sessions", async () => {
    const mockData = { sessions: [{ id: "s1", user_id: "u1" }] };
    vi.spyOn(client, "get").mockResolvedValueOnce({ data: mockData } as unknown as { data: typeof mockData });
    const sessions = await listSessions("proj-1");
    expect(sessions).toEqual(mockData.sessions);
  });

  it("listSessions propagates errors", async () => {
    vi.spyOn(client, "get").mockRejectedValueOnce(new Error("network"));
    await expect(listSessions("proj-1")).rejects.toThrow("network");
  });

  it("getSession returns session detail with event ids", async () => {
    const mockData = {
      session: { id: "s1", user_id: "u1" },
      events: [
        {
          event_type: "click",
          timestamp_ms: 1000,
          x: 10,
          y: 20,
          target: "btn",
          payload: "",
        },
      ],
    };
    vi.spyOn(client, "get").mockResolvedValueOnce({ data: mockData } as unknown as { data: typeof mockData });
    const detail = await getSession("s1");
    expect(detail.session.id).toBe("s1");
    expect(detail.events[0].id).toBe("click-1000-10-20");
  });

  it("getSession preserves existing event ids", async () => {
    const mockData = {
      session: { id: "s1", user_id: "u1" },
      events: [
        {
          id: "existing-id",
          event_type: "click",
          timestamp_ms: 1000,
          x: 10,
          y: 20,
          target: "btn",
          payload: "",
        },
      ],
    };
    vi.spyOn(client, "get").mockResolvedValueOnce({ data: mockData } as unknown as { data: typeof mockData });
    const detail = await getSession("s1");
    expect(detail.events[0].id).toBe("existing-id");
  });

  it("getSession propagates errors", async () => {
    vi.spyOn(client, "get").mockRejectedValueOnce(new Error("network"));
    await expect(getSession("s1")).rejects.toThrow("network");
  });
});
