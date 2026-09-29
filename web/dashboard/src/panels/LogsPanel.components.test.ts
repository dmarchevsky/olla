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

  it('applies component= to the query when an event type is selected', async () => {
    const fetchMock = mockFetch();
    component = mount(LogsPanel, { target: document.body });
    flushSync();
    await vi.waitFor(() => expect(logs.entries.length).toBe(1));

    const dropdowns = [...document.querySelectorAll<HTMLButtonElement>('.ms-button')];
    const typeButton = dropdowns.find((b) => b.textContent?.includes('Event type'));
    expect(typeButton).toBeTruthy();
    typeButton!.click();
    flushSync();

    const option = [...document.querySelectorAll<HTMLLabelElement>('.ms-option')].find((l) =>
      l.textContent?.includes('health')
    );
    expect(option).toBeTruthy();
    option!.querySelector('input')!.click();
    flushSync();

    await vi.waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('component=health'), expect.anything()),
    );
  });
});
