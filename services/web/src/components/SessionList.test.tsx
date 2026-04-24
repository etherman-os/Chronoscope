import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { SessionList } from './SessionList';

// Mock the API client before importing the component
vi.mock('../api/client', () => ({
  listSessions: vi.fn(),
}));

import { listSessions } from '../api/client';

describe('SessionList', () => {
  it('renders loading state', () => {
    vi.mocked(listSessions).mockReturnValue(new Promise(() => {}));
    render(<SessionList onSelect={() => {}} />);
    expect(screen.getByText(/Loading/i)).toBeInTheDocument();
  });

  it('renders error state when API fails', async () => {
    vi.mocked(listSessions).mockRejectedValue(new Error('Network error'));
    render(<SessionList onSelect={() => {}} />);
    await waitFor(() => {
      expect(screen.getByText(/Failed to load sessions/i)).toBeInTheDocument();
    });
  });

  it('renders sessions after loading', async () => {
    const mockSessions = [
      {
        id: 'session-1',
        user_id: 'user-1',
        duration_ms: 15000,
        status: 'completed',
        created_at: new Date().toISOString(),
      },
      {
        id: 'session-2',
        user_id: 'user-2',
        duration_ms: 30000,
        status: 'capturing',
        created_at: new Date().toISOString(),
      },
    ];
    vi.mocked(listSessions).mockResolvedValue(mockSessions);
    render(<SessionList onSelect={() => {}} />);

    await waitFor(() => {
      expect(screen.getByText('user-1')).toBeInTheDocument();
    });
    expect(screen.getByText('user-2')).toBeInTheDocument();
  });

  it('renders empty state when no sessions', async () => {
    vi.mocked(listSessions).mockResolvedValue([]);
    render(<SessionList onSelect={() => {}} />);

    await waitFor(() => {
      expect(screen.getByText(/No sessions found/i)).toBeInTheDocument();
    });
  });

  it('calls onSelect when a session is clicked', async () => {
    const mockSessions = [
      {
        id: 'session-1',
        user_id: 'user-1',
        duration_ms: 15000,
        status: 'completed',
        created_at: new Date().toISOString(),
      },
    ];
    const handleSelect = vi.fn();
    vi.mocked(listSessions).mockResolvedValue(mockSessions);
    render(<SessionList onSelect={handleSelect} />);

    await waitFor(() => {
      expect(screen.getByText('user-1')).toBeInTheDocument();
    });

    screen.getByText('user-1').click();
    expect(handleSelect).toHaveBeenCalledWith(mockSessions[0]);
  });
});
