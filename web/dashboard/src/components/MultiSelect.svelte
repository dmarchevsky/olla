<script lang="ts" generics="T extends string">
  interface Props {
    label: string;
    options: { value: T; label: string }[];
    selected: Set<T>;
    onChange: (next: Set<T>) => void;
  }

  let { label, options, selected, onChange }: Props = $props();

  let open = $state(false);
  let root: HTMLDivElement | undefined = $state();
  let button: HTMLButtonElement | undefined = $state();

  function toggleOpen(): void {
    open = !open;
  }

  // Capture phase so a click on the trigger itself (which re-toggles via
  // onclick) is not double-handled: clicks inside the popover never close it,
  // clicks anywhere else do.
  function onDocClick(e: MouseEvent): void {
    if (open && root && !root.contains(e.target as Node)) open = false;
  }

  function onKeydown(e: KeyboardEvent): void {
    if (e.key === 'Escape' && open) {
      open = false;
      button?.focus();
    }
  }

  $effect(() => {
    if (!open) return;
    document.addEventListener('click', onDocClick, true);
    document.addEventListener('keydown', onKeydown);
    return () => {
      document.removeEventListener('click', onDocClick, true);
      document.removeEventListener('keydown', onKeydown);
    };
  });

  function toggleValue(value: T): void {
    const next = new Set(selected);
    if (next.has(value)) next.delete(value);
    else next.add(value);
    onChange(next);
  }

  function clear(): void {
    onChange(new Set());
  }

  const summary = $derived(selected.size === 0 ? 'all' : `${selected.size} selected`);
</script>

<div class="ms" bind:this={root}>
  <button
    type="button"
    class="ms-button"
    class:open
    bind:this={button}
    aria-expanded={open}
    aria-label={label}
    onclick={toggleOpen}
  >
    <span class="ms-label">{label}</span>
    <span class="ms-summary">{summary}</span>
    <span class="ms-caret" aria-hidden="true">{open ? '▴' : '▾'}</span>
  </button>
  {#if open}
    <div class="ms-pop" role="group" aria-label={label}>
      <div class="ms-options">
        {#each options as opt (opt.value)}
          <label class="ms-option">
            <input type="checkbox" checked={selected.has(opt.value)} onchange={() => toggleValue(opt.value)} />
            <span>{opt.label}</span>
          </label>
        {/each}
      </div>
      <button type="button" class="ms-clear" onclick={clear}>clear</button>
    </div>
  {/if}
</div>

<style>
  .ms {
    position: relative;
  }
  /* Same chrome as the neighbouring .log-select so the dropdown reads as
     part of the existing filter bar rather than a new control language. */
  .ms-button {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: var(--bg-elevated);
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-sm);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.72rem;
    padding: 5px 8px;
    cursor: pointer;
  }
  .ms-button:hover,
  .ms-button.open {
    border-color: var(--accent);
  }
  .ms-button:focus-visible {
    outline: var(--focus-ring);
    outline-offset: 1px;
  }
  .ms-summary {
    color: var(--text-faint);
  }
  .ms-caret {
    color: var(--text-faint);
  }
  .ms-pop {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    z-index: 30;
    min-width: 160px;
    background: var(--bg-elevated);
    border: 1px solid var(--border-strong);
    border-radius: var(--radius-sm);
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.18);
    padding: 4px;
  }
  .ms-options {
    display: flex;
    flex-direction: column;
  }
  .ms-option {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 3px 6px;
    font-family: var(--font-mono);
    font-size: 0.72rem;
    color: var(--text);
    cursor: pointer;
    white-space: nowrap;
  }
  .ms-option:hover {
    background: var(--bg-inset);
  }
  .ms-clear {
    width: 100%;
    margin-top: 2px;
    padding: 3px 6px;
    background: transparent;
    border: 0;
    border-top: 1px dashed var(--border);
    color: var(--text-faint);
    font-family: var(--font-mono);
    font-size: 0.68rem;
    text-align: left;
    cursor: pointer;
  }
  .ms-clear:hover {
    color: var(--accent);
  }
</style>
