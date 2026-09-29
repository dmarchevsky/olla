import { flushSync, mount, unmount } from 'svelte';
import { describe, it, expect, afterEach, vi } from 'vitest';
import MultiSelect from './MultiSelect.svelte';

const OPTIONS = [
  { value: 'request', label: 'request' },
  { value: 'health', label: 'health' },
];

let component: ReturnType<typeof mount> | undefined;
afterEach(() => {
  if (component) unmount(component);
  document.body.innerHTML = '';
});

function mountMs(selected: Set<string>, onChange: (next: Set<string>) => void) {
  component = mount(MultiSelect, {
    target: document.body,
    props: { label: 'Event type', options: OPTIONS, selected, onChange },
  });
  flushSync();
}

function trigger(): HTMLButtonElement {
  return document.querySelector('.ms-button')!;
}

function optionInput(label: string): HTMLInputElement {
  const el = [...document.querySelectorAll<HTMLLabelElement>('.ms-option')].find((l) =>
    l.textContent?.includes(label)
  )!;
  return el.querySelector('input')!;
}

describe('MultiSelect', () => {
  it('opens the popover from the trigger and reports toggles', () => {
    const onChange = vi.fn();
    mountMs(new Set(), onChange);

    expect(document.querySelector('.ms-pop')).toBeNull();
    expect(trigger().getAttribute('aria-expanded')).toBe('false');
    expect(trigger().textContent).toContain('all');

    trigger().click();
    flushSync();
    expect(document.querySelector('.ms-pop')).toBeTruthy();
    expect(trigger().getAttribute('aria-expanded')).toBe('true');

    optionInput('health').click();
    expect(onChange).toHaveBeenCalledTimes(1);
    expect([...onChange.mock.calls[0][0]]).toEqual(['health']);
  });

  it('shows the selection count and clears via the clear button', () => {
    const onChange = vi.fn();
    mountMs(new Set(['request', 'health']), onChange);

    expect(trigger().textContent).toContain('2 selected');

    trigger().click();
    flushSync();
    document.querySelector<HTMLButtonElement>('.ms-clear')!.click();
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange.mock.calls[0][0].size).toBe(0);
  });

  it('closes when clicking outside', () => {
    mountMs(new Set(), vi.fn());
    trigger().click();
    flushSync();
    expect(document.querySelector('.ms-pop')).toBeTruthy();

    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    flushSync();
    expect(document.querySelector('.ms-pop')).toBeNull();
  });
});
