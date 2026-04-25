import React, { useState, useEffect } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Dashboard } from "./pages/Dashboard";
import { Login } from "./pages/Login";

export const App: React.FC = () => {
  const [authenticated, setAuthenticated] = useState(false);
  const [projectId, setProjectId] = useState("");

  useEffect(() => {
    const storedKey = localStorage.getItem("chronoscope_api_key");
    const storedProject = localStorage.getItem("chronoscope_project_id");
    if (storedKey && storedProject) {
      setAuthenticated(true);
      setProjectId(storedProject);
    }
  }, []);

  const handleAuthenticated = (_apiKey: string, pid: string) => {
    setAuthenticated(true);
    setProjectId(pid);
  };

  if (!authenticated) {
    return <Login onAuthenticated={handleAuthenticated} />;
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Dashboard projectId={projectId} />} />
      </Routes>
    </BrowserRouter>
  );
};
