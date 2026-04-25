import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { App } from "./App";

describe("App", () => {
  it("renders dashboard route", () => {
    render(<App />);
    expect(screen.getByText(/Select a session to view replay/i)).toBeInTheDocument();
  });
});
