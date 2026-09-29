<script lang="ts">
  import { logs, type LogFilters, type LogComponent, type TimePreset } from '../lib/stores/logs.svelte';
  import { endpoints } from '../lib/stores/endpoints.svelte';
  import StatusBanner from '../components/StatusBanner.svelte';
  import SortableTable from '../components/SortableTable.svelte';
  import MultiSelect from '../components/MultiSelect.svelte';
  import type { Column } from '../components/SortableTable.svelte';
  import { fmtLocalTime } from '../lib/format';
  import type { LogEntry, LogLevel } from '../lib/types';

  // Mapped-type alias (not a plain intersection) so it carries the implicit
  // index signature SortableTable's `Row extends Record<string, unknown>`
  // constraint needs - mirrors ModelsPanel's ModelRow for the same reason.
  type LogRow = Omit<LogEntry, never>;

  const LEVELS: LogLevel[] = ['debug', 'info', 'warn', 'error'];
  const LEVEL_OPTIONS = LEVELS.map((l) => ({ value: l, label: l }));
  const COMPONENTS: LogComponent[] = ['request', 'health', 'discovery', 'registry', 'config', 'system'];
  const COMPONENT_OPTIONS = COMPONENTS.map((c) => ({ value: c, label: c }));
  const LEVEL_GLYPH: Record<LogLevel, string> = { debug: '○', info: '●', warn: '◐', error: '○' };
  const LEVEL_COLOUR: Record<LogLevel, string> = { debug: 'neutral', info: 'green', warn: 'amber', error: 'red' };
  const PRESETS: { value: TimePreset; label: string }[] = [
    { value: '15m', label: 'Last 15m' },
    { value: '1h', label: 'Last 1h' },
    { value: '6h', label: 'Last 6h' },
    { value: '24h', label: 'Last 24h' },
    { value: 'all', label: 'All' },
  ];

  // This panel owns both the logs store's follow-poll lifecycle and, like
  // EndpointsPanel/OverviewPanel, the endpoints store's - it's the source for
  // the endpoint filter's option list. Only one panel is ever mounted at a
  // time (App.svelte routes conditionally), so duplicating start()/stop()
  // across panels is the established, safe pattern here.
  $effect(() => {
    logs.refresh();
    endpoints.start();
    return () => {
      logs.setFollowing(false);
      endpoints.stop();
    };
  });

  const loading = $derived(logs.status === 'loading');
  const endpointOptions = $derived(endpoints.data?.endpoints.map((e) => e.name) ?? []);

  let preset: TimePreset = $state('1h');
  let selectedLevels: Set<LogLevel> = $state(new Set());
  let selectedComponents: Set<LogComponent> = $state(new Set());
  let endpointFilter: string = $state('');

  function applyFilters(): void {
    logs.setFilters({
      preset,
      levels: new Set(selectedLevels),
      components: new Set(selectedComponents),
      endpoint: endpointFilter,
    } satisfies LogFilters);
  }

  function onLevelsChange(next: Set<LogLevel>): void {
    selectedLevels = next;
    applyFilters();
  }

  function onComponentsChange(next: Set<LogComponent>): void {
    selectedComponents = next;
    applyFilters();
  }

  function onPresetChange(): void {
    applyFilters();
  }

  function onEndpointChange(): void {
    applyFilters();
  }

  function onFollowToggle(): void {
    logs.setFollowing(!logs.following);
  }

  const columns: Column[] = [
    { key: 'time', label: 'Time', sortable: true, sticky: true },
    { key: 'level', label: 'Level', sortable: true },
    { key: 'component', label: 'Source', sortable: true },
    { key: 'endpoint', label: 'Endpoint', sortable: true },
    { key: 'message', label: 'Message', sortable: false },
  ];

  function rowId(e: LogRow): string {
    return String(e.seq);
  }

  // Auto-scroll to newest while following, unless the operator has scrolled
  // up to read older entries - standard tail -f UX. Wraps SortableTable
  // (whose own .table-scroll only scrolls horizontally) in a vertically
  // capped container so a live tail has a bounded viewport instead of
  // growing the whole page.
  let logScroll: HTMLElement | undefined = $state();
  let stickToBottom = $state(true);

  // Size the scroll container to exactly the remaining viewport height so
  // the page itself never scrolls on this tab. A fixed 60vh cap left the
  // page scrollbar and the container scrollbar competing whenever the
  // banner or filter bar changed height.
  let scrollMaxHeight = $state(320);

  function measureScroll(): void {
    if (!logScroll) return;
    // Document-relative top so the measurement holds even if the page is
    // scrolled at measure time; the bottom gap is main's real padding, not
    // a guess - a constant under-count left a few px of page overflow and
    // therefore a permanent page scrollbar.
    const mainEl = logScroll.closest('main');
    const padBottom = mainEl ? parseFloat(getComputedStyle(mainEl).paddingBottom) || 0 : 0;
    const top = logScroll.getBoundingClientRect().top + window.scrollY;
    const h = Math.max(240, window.innerHeight - top - padBottom);
    if (h !== scrollMaxHeight) scrollMaxHeight = h;
  }

  function onScroll(): void {
    if (!logScroll) return;
    const gap = logScroll.scrollHeight - logScroll.scrollTop - logScroll.clientHeight;
    stickToBottom = gap < 40;
  }

  $effect(() => {
    // Re-run whenever entries change (Svelte tracks logs.entries via this read).
    const count = logs.entries.length;
    const status = logs.status;
    const truncated = logs.truncated;
    if (count > 0 && logs.following && stickToBottom && logScroll) {
      logScroll.scrollTop = logScroll.scrollHeight;
    }
    // Re-measure after any layout shift: banner appearing/clearing, the
    // truncated note, or the filter bar wrapping on a narrow viewport.
    void status;
    void truncated;
    measureScroll();
  });

  $effect(() => {
    window.addEventListener('resize', measureScroll);
    return () => window.removeEventListener('resize', measureScroll);
  });
</script>

<div
  id="panel-logs"
  class="panel is-active"
  role="tabpanel"
  aria-labelledby="tab-logs"
  tabindex="0"
>
  <StatusBanner store={logs} />

  <p class="panel-intro">Recent log activity captured from Olla's own process, in-memory only.</p>

  <div class="logs-filterbar">
    <select class="log-select" bind:value={preset} onchange={onPresetChange} aria-label="Time period">
      {#each PRESETS as p (p.value)}
        <option value={p.value}>{p.label}</option>
      {/each}
    </select>

    <MultiSelect label="Severity" options={LEVEL_OPTIONS} selected={selectedLevels} onChange={onLevelsChange} />

    <MultiSelect label="Event type" options={COMPONENT_OPTIONS} selected={selectedComponents} onChange={onComponentsChange} />

    <select class="log-select" bind:value={endpointFilter} onchange={onEndpointChange} aria-label="Endpoint filter">
      <option value="">All endpoints</option>
      {#each endpointOptions as name (name)}
        <option value={name}>{name}</option>
      {/each}
    </select>

    <button type="button" class="theme-toggle" class:active={logs.following} onclick={onFollowToggle}>
      {logs.following ? '● following' : '○ follow'}
    </button>
  </div>

  {#if logs.truncated}
    <p class="logs-truncated-note">Some older entries may have been dropped from the in-memory buffer.</p>
  {/if}

  <div class="panel-data" data-state={logs.status === 'error' || logs.status === 'stale' ? logs.status : null}>
    {#if loading}
      <div class="table-scroll">
        <div class="scroll-hint">loading…</div>
        {#each Array(6) as _, i (i)}<div class="skeleton row-skel" style="margin:6px 10px"></div>{/each}
      </div>
    {:else if logs.entries.length}
      <div class="log-scroll" bind:this={logScroll} onscroll={onScroll} style="max-height: {scrollMaxHeight}px">
        <SortableTable
          {columns}
          rows={logs.entries as LogRow[]}
          initialSort={null}
          {rowId}
          showScrollHint={false}
        >
          {#snippet rowSnippet({ row: e })}
            <td class="col-sticky log-time">{fmtLocalTime(e.time)}</td>
            <td>
              <span class="glyph g-{LEVEL_COLOUR[e.level as LogLevel] ?? 'neutral'}" aria-hidden="true"
                >{LEVEL_GLYPH[e.level as LogLevel] ?? '○'}</span
              >
              {e.level}
            </td>
            <td class="log-source">{e.component || '—'}</td>
            <td>{e.endpoint || '—'}</td>
            <td class="log-message">
              {e.message}
              {#if e.attrs}
                {#each Object.entries(e.attrs) as [key, value] (key)}
                  <span class="log-attr">{key}={value}</span>
                {/each}
              {/if}
            </td>
          {/snippet}
        </SortableTable>
      </div>
    {:else if logs.hasData}
      <p class="panel-intro">No log entries match the current filters.</p>
    {/if}
  </div>
</div>

<style>
  .logs-filterbar {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
    margin-bottom: var(--space-3);
  }

  /* New control, but built from the app's existing tokens (matching
     .theme-toggle's chrome) rather than a browser-default select. */
  .log-select {
    background: var(--bg-elevated);
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.72rem;
    padding: 5px 8px;
  }
  .log-select:focus-visible {
    outline: var(--focus-ring);
    outline-offset: 1px;
  }

  .theme-toggle.active {
    border-color: var(--accent);
    color: var(--accent);
  }

  .logs-truncated-note {
    color: var(--amber);
    font-size: 0.72rem;
    margin: 0 0 var(--space-2);
  }

  /* Vertical scroll around SortableTable (whose own .table-scroll only
     scrolls horizontally); max-height is set inline from JS to exactly the
     remaining viewport height, so the page itself never gains a second
     scrollbar. */
  .log-scroll {
    overflow-y: auto;
  }
  /* The global table min-width (860px) forces a horizontal scrollbar on
     narrow viewports; log columns fit without it since the message cell
     wraps. */
  #panel-logs :global(table) {
    min-width: 0;
  }
  .log-time {
    white-space: nowrap;
    color: var(--text-dim);
    font-family: var(--font-mono);
  }
  .log-message {
    word-break: break-word;
  }
  .log-source {
    white-space: nowrap;
    color: var(--text-dim);
    font-family: var(--font-mono);
  }
  /* Structured attrs (models=4, duration_ms=120, ...) rendered inline so the
     detail the API already returns is visible without an expand affordance. */
  .log-attr {
    display: inline-block;
    margin-left: 4px;
    padding: 0 5px;
    background: var(--bg-inset);
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-sm);
    color: var(--text-dim);
    font-family: var(--font-mono);
    font-size: 0.68rem;
    white-space: nowrap;
  }
</style>
