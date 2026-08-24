// /internal/logs poller backing the Logs panel. Unlike models/endpoints
// (fixed URL, createPollStore), this store has mutable filters and
// cursor-append semantics - a fresh filter can't be applied to entries
// already fetched under the old filter, since filtering happens server-side -
// so it's bespoke, but still drives its recurring tick through the single
// shared pollScheduler rather than a second timer mechanism.
import type { LogEntry, LogLevel, LogsResponse, PollStatus } from '../types';
import { pollScheduler, STALE_MULTIPLIER } from '../poll-scheduler';

const LOGS_JOB_NAME = 'logs';
const FOLLOW_INTERVAL_MS = 2000;
// Client-side cap on accumulated entries, mirroring the server's default ring
// buffer capacity, so a long-running follow session can't grow unbounded.
const CLIENT_ENTRY_CAP = 5000;

export type TimePreset = '15m' | '1h' | '6h' | '24h' | 'all';

export interface LogFilters {
  preset: TimePreset;
  levels: Set<LogLevel>;
  endpoint: string; // '' = all endpoints
}

const PRESET_MS: Record<Exclude<TimePreset, 'all'>, number> = {
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
};

function buildUrl(filters: LogFilters, since: number): string {
  const params = new URLSearchParams();
  if (since > 0) params.set('since', String(since));
  if (filters.preset !== 'all') {
    const from = new Date(Date.now() - PRESET_MS[filters.preset]);
    params.set('from', from.toISOString());
  }
  if (filters.levels.size > 0) params.set('level', [...filters.levels].join(','));
  if (filters.endpoint) params.set('endpoint', filters.endpoint);
  return `/internal/logs?${params.toString()}`;
}

class LogsStore {
  #entries: LogEntry[] = $state([]);
  #sinceSeq = 0;
  #status: PollStatus = $state('loading');
  #error: Error | null = $state(null);
  #lastUpdated: Date | null = $state(null);
  #lastSuccessAt = 0;
  #following = $state(false);
  #truncated = $state(false);
  #filters: LogFilters = $state({ preset: '1h', levels: new Set(), endpoint: '' });

  get name(): string {
    return 'logs';
  }
  get entries(): LogEntry[] {
    return this.#entries;
  }
  get status(): PollStatus {
    return this.#status;
  }
  get error(): Error | null {
    return this.#error;
  }
  get lastUpdated(): Date | null {
    return this.#lastUpdated;
  }
  get hasData(): boolean {
    return this.#status !== 'loading';
  }
  get following(): boolean {
    return this.#following;
  }
  get filters(): LogFilters {
    return this.#filters;
  }
  get truncated(): boolean {
    return this.#truncated;
  }

  constructor() {
    pollScheduler.register(LOGS_JOB_NAME, FOLLOW_INTERVAL_MS, (signal) => this.#tick(signal));
  }

  setFilters(filters: LogFilters): void {
    this.#filters = filters;
    this.#resetAndRefetch();
  }

  setFollowing(on: boolean): void {
    this.#following = on;
    if (on) {
      pollScheduler.start(LOGS_JOB_NAME);
    } else {
      pollScheduler.stop(LOGS_JOB_NAME);
    }
  }

  /** One-off fetch of the current filtered window. Called on panel mount
   *  regardless of follow state, so opening the tab always shows current
   *  data even with follow off. Safe to call while following too (mirrors
   *  "retry now" on the other panels). */
  refresh(): void {
    pollScheduler.refresh(LOGS_JOB_NAME);
  }

  #resetAndRefetch(): void {
    this.#entries = [];
    this.#sinceSeq = 0;
    this.#truncated = false;
    this.refresh();
  }

  async #tick(signal: AbortSignal): Promise<void> {
    const url = buildUrl(this.#filters, this.#sinceSeq);
    let resp: Response;
    try {
      resp = await fetch(url, { signal, cache: 'no-store', headers: { Accept: 'application/json' } });
    } catch (e) {
      if (e instanceof Error && e.name === 'AbortError') return;
      this.#onFailure(e instanceof Error ? e : new Error(String(e || 'network error')));
      return;
    }
    if (!resp.ok) {
      this.#onFailure(new Error(`HTTP ${resp.status}`));
      return;
    }
    let body: LogsResponse;
    try {
      body = (await resp.json()) as LogsResponse;
    } catch {
      this.#onFailure(new Error('invalid JSON from logs API'));
      return;
    }

    if (body.truncated) this.#truncated = true;
    if (body.entries.length > 0) {
      this.#entries = [...this.#entries, ...body.entries].slice(-CLIENT_ENTRY_CAP);
    }
    this.#sinceSeq = body.next_since;

    this.#error = null;
    this.#status = 'ok';
    this.#lastUpdated = new Date();
    this.#lastSuccessAt = Date.now();
  }

  #onFailure(err: Error): void {
    this.#error = err;
    const hasPrior = this.#entries.length > 0;
    const ageMs = Date.now() - this.#lastSuccessAt;
    if (hasPrior && ageMs > FOLLOW_INTERVAL_MS * STALE_MULTIPLIER) {
      this.#status = 'stale';
    } else {
      this.#status = 'error';
    }
  }
}

export const logs: LogsStore = new LogsStore();
