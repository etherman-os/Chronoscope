import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { EventTimeline } from './EventTimeline';

describe('EventTimeline', () => {
  it('renders timeline with events', () => {
    const events = [
      { event_type: 'click', timestamp_ms: 1000, x: 100, y: 200, target: 'btn', payload: '' },
      { event_type: 'scroll', timestamp_ms: 2000, x: 0, y: 500, target: 'window', payload: '' },
    ];
    render(<EventTimeline events={events} currentTime={1500} />);

    // Event markers should be rendered
    const markers = screen.getAllByTitle(/click|scroll/);
    expect(markers.length).toBe(2);
  });

  it('positions click events in the top half', () => {
    const events = [
      { event_type: 'click', timestamp_ms: 1000, x: 100, y: 200, target: 'btn', payload: '' },
    ];
    const { container } = render(<EventTimeline events={events} currentTime={0} />);
    const marker = container.querySelector('[title="click — 1000ms"]');
    expect(marker).not.toBeNull();
    if (marker) {
      expect((marker as HTMLElement).style.top).toBe('0px');
    }
  });

  it('positions scroll events in the bottom half', () => {
    const events = [
      { event_type: 'scroll', timestamp_ms: 2000, x: 0, y: 500, target: 'window', payload: '' },
    ];
    const { container } = render(<EventTimeline events={events} currentTime={0} />);
    const marker = container.querySelector('[title="scroll — 2000ms"]');
    expect(marker).not.toBeNull();
    if (marker) {
      expect((marker as HTMLElement).style.top).toBe('50%');
    }
  });

  it('renders current time indicator', () => {
    const events = [
      { event_type: 'click', timestamp_ms: 1000, x: 100, y: 200, target: 'btn', payload: '' },
    ];
    const { container } = render(<EventTimeline events={events} currentTime={500} />);
    const indicator = container.querySelector('div[style*="background-color: rgb(46, 204, 113)"]');
    expect(indicator).not.toBeNull();
  });

  it('handles empty events array', () => {
    const { container } = render(<EventTimeline events={[]} currentTime={0} />);
    const indicator = container.querySelector('div[style*="background-color: rgb(46, 204, 113)"]');
    expect(indicator).not.toBeNull();
  });

  it('scales events relative to max time', () => {
    const events = [
      { event_type: 'click', timestamp_ms: 5000, x: 100, y: 200, target: 'btn', payload: '' },
    ];
    const { container } = render(<EventTimeline events={events} currentTime={2500} />);
    const marker = container.querySelector('[title="click — 5000ms"]');
    expect(marker).not.toBeNull();
    if (marker) {
      // Should be at 100% since it's the max time
      expect((marker as HTMLElement).style.left).toBe('100%');
    }
  });
});
