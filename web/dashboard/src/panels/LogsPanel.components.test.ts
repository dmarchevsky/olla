import { flushSync, mount, unmount } from 'svelte';
import { describe, it, expect, afterEach, vi } from 'vitest';
import { logs } from '../lib/stores/logs.svelte';
import LogsPanel from './LogsPanel.svelte';

// Event-type filtering: the panel exposes component pills that narrow the
// server-side query via component=, and a Source column showing each entry's
// component (or "—" when untagged).

let component: ReturnType<typeof mount> | undefined;
afterEach(() => {
  logs.setFollowing(false);
  if (component) unmount(component);
  document.body.innerHTML = '';
});

function mockFetch() {
  const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input);
    if (url.startsWith('/internal/logs')) {
      return Response.json({
        entries: [
          {
            time: new Date().toISOString(),
            level: 'info',
            message: 'Endpoint recovered: ryzen is Healthy',
            endpoint: 'ryzen',
            component: 'health',
            seq: 1,
          },
        ],
        next_since: 1,
        head_seq: 1,
        capacity: 5000,
        truncated: false,
      });
    }
    if (url.startsWith('/internal/status/endpoints')) {
      return Response.json({ endpoints: [] });
    }
    return Response.json({});
  });
  global.fetch = fetchMock;
  return fetchMock;
}

describe('LogsPanel component filter', () => {
  it('shows the component in the Source column', async () => {
    mockFetch();
    component = mount(LogsPanel, { target: document.body });
    flushSync();
    await vi.waitFor(() => expect(logs.entries.length).toBe(1));
    flushSync();

    const source = document.querySelector('.log-source');
    expect(source?.textContent?.trim()).toBe('health');
  });

  it('applies component= to the query when a pill is toggled', async () => {
    const fetchMock = mockFetch();
    component = mount(LogsPanel, { target: document.body });
    flushSync();
    await vi.waitFor(() => expect(logs.entries.length).toBe(1));

    const pills = [...document.querySelectorAll<HTMLButtonElement>('.pill-toggle')];
    const healthPill = pills.find((b) => b.textContent?.includes('health'));
    expect(healthPill).toBeTruthy();
    healthPill!.click();
    flushSync();

    await vi.waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('component=health'), expect.anything()),
    );
  });
});
