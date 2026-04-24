-- 001_initial_schema.sql
-- Chronoscope Initial Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Organizations table
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    plan VARCHAR(50) DEFAULT 'free',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(255) UNIQUE,
    privacy_config JSONB DEFAULT '{}',
    retention_days INT DEFAULT 30,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    duration_ms INT,
    video_path VARCHAR(500),
    event_count INT DEFAULT 0,
    error_count INT DEFAULT 0,
    metadata JSONB,
    status VARCHAR(50) DEFAULT 'capturing',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ
);

-- Events table
CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    event_type VARCHAR(50),
    timestamp_ms INT,
    x INT,
    y INT,
    target VARCHAR(255),
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at);

-- Seed data
INSERT INTO organizations (name, plan) VALUES ('Chronoscope Dev', 'enterprise');

INSERT INTO projects (org_id, name, api_key_hash)
SELECT 
    id,
    'Demo App',
    'dev-api-key-12345'
FROM organizations 
WHERE name = 'Chronoscope Dev';
