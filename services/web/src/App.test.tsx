import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { App } from "./App";

vi.mock("react-router-dom", () => ({
  BrowserRouter: ({ children }: { children: ReactNode }) => <>{children}</>,
  Routes: ({ children }: { children: ReactNode }) => <>{children}</>,
  Route: ({ element }: { element: ReactNode }) => <>{element}</>,
}));

vi.mock("./pages/Dashboard", () => ({
  Dashboard: ({ projectId }: { projectId: string }) => (
    <div>Dashboard for {projectId}</div>
  ),
}));

describe("App", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("shows login page when no API key stored", () => {
    render(<App />);
    expect(screen.getByText(/Chronoscope/)).toBeInTheDocument();
    expect(screen.getByLabelText(/API Key/)).toBeInTheDocument();
  });

  it("shows dashboard when API key is stored", () => {
    localStorage.setItem("chronoscope_api_key", "test-key");
    localStorage.setItem("chronoscope_project_id", "test-project");
    render(<App />);
    expect(screen.getByText(/Dashboard for test-project/i)).toBeInTheDocument();
  });
});
