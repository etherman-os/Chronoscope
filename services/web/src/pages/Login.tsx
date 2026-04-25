import React, { useState } from "react";

interface LoginProps {
  onAuthenticated: (apiKey: string, projectId: string) => void;
}

export const Login: React.FC<LoginProps> = ({ onAuthenticated }) => {
  const [apiKey, setApiKey] = useState("");
  const [projectId, setProjectId] = useState(
    () => import.meta.env.VITE_PROJECT_ID || "",
  );
  const [error, setError] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!apiKey.trim()) {
      setError("API key is required");
      return;
    }
    if (!projectId.trim()) {
      setError("Project ID is required");
      return;
    }
    localStorage.setItem("chronoscope_api_key", apiKey.trim());
    localStorage.setItem("chronoscope_project_id", projectId.trim());
    setError("");
    onAuthenticated(apiKey.trim(), projectId.trim());
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>Chronoscope</h1>
        <p>Session replay infrastructure</p>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="api-key">API Key</label>
            <input
              id="api-key"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="Enter your API key"
              autoComplete="current-password"
            />
          </div>
          <div className="form-group">
            <label htmlFor="project-id">Project ID</label>
            <input
              id="project-id"
              type="text"
              value={projectId}
              onChange={(e) => setProjectId(e.target.value)}
              placeholder="Enter project ID"
            />
          </div>
          {error && <div className="form-error">{error}</div>}
          <button type="submit" className="login-button">
            Connect
          </button>
        </form>
      </div>
    </div>
  );
};
