import { render, screen } from "@testing-library/react";
import { describe, it, expect, beforeEach } from "vitest";
import { App } from "./App";

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
    expect(screen.getByText(/Select a session to view replay/i)).toBeInTheDocument();
  });
});
