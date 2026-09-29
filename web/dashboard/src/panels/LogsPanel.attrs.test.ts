import { flushSync, mount, unmount } from 'svelte';
import { describe, it, expect, afterEach, vi } from 'vitest';
import { logs } from '../lib/stores/logs.svelte';
import LogsPanel from './LogsPanel.svelte';

// The /internal/logs API returns structured attrs (models=4, duration_ms=...)
// alongside each entry; the panel must surface them, otherwise the message
// column alone is all the operator gets.

let component: ReturnType<typeof mount> | undefined;
afterEach(() => {
  logs.setFollowing(false);
  if (component) unmount(component);
  document.body.innerHTML = '';
});

describe('LogsPanel attrs chips', () => {
  it('renders structured attrs as chips next to the message', async () => {
    global.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.startsWith('/internal/logs')) {
        return Response.json({
          entries: [
            {
              time: new Date().toISOString(),
              level: 'info',
              message: 'Discovered models',
              endpoint: 'ryzen',
              attrs: { models: '4' },
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

    component = mount(LogsPanel, { target: document.body });
    flushSync();

    await vi.waitFor(() => expect(logs.entries.length).toBe(1));
    flushSync();

    const chips = [...document.querySelectorAll('.log-attr')].map((el) => el.textContent);
    expect(chips).toContain('models=4');
    // Endpoint column populated from the entry's endpoint field.
    expect(document.body.textContent).toContain('ryzen');
  });
});
