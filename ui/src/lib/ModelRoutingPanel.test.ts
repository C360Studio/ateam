import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import ModelRoutingPanel from './ModelRoutingPanel.svelte';
import type { ModelRouting } from './board';

const roles = {
 developer: { model: 'dev-exact-id', provider: 'codex', effort: 'custom-effort' },
 reviewer: { model: 'review-exact-id', provider: 'claude', effort: 'high' }
};

describe('model routing panel', () => {
 // spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
 it.each([
  ['eligible', 'Eligible route'], ['blocked', 'Routing blocked'],
  ['unconfigured', 'Not configured'], ['invalid', 'Invalid policy'], ['unavailable', 'Routing unavailable']
 ] as const)('renders %s with honest scope and no edit controls', (status, label) => {
  const routing: ModelRouting = { status, reason: 'A specific explanation', ...roles };
  const html = render(ModelRoutingPanel, { props: { routing } }).body;
  expect(html).toContain(label);
  expect(html).toContain('A specific explanation');
  expect(html).toContain('Configured routes do not identify running models or approve a PR.');
  expect(html).toContain('Reasoning effort is separate from model strength and Ponytail mode.');
  expect(html).not.toMatch(/<(button|input|select|textarea)\b/);
  if (status === 'eligible' || status === 'blocked') {
   for (const text of ['Developer', 'Reviewer', 'dev-exact-id', 'review-exact-id', 'codex', 'claude', 'custom-effort', 'high']) {
    expect(html).toContain(text);
   }
  } else {
   expect(html).not.toContain('dev-exact-id');
   expect(html).not.toContain('review-exact-id');
  }
  if (status === 'unconfigured') expect(html).toContain('ateam review setup --policy FILE');
 });

 // spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
 it('treats absent metadata as unavailable rather than unconfigured or eligible', () => {
  const html = render(ModelRoutingPanel, { props: {} }).body;
  expect(html).toContain('Routing unavailable');
  expect(html).toContain('server that supports routing');
  expect(html).not.toContain('Not configured');
  expect(html).not.toContain('Eligible route');
 });

 // spec: openspec/specs/model-review/spec.md#Requirement: Read-only dashboard model routing
 it('escapes policy text and retains blocked routes despite a stronger effort label', () => {
  const html = render(ModelRoutingPanel, { props: { routing: {
   status: 'blocked', reason: '<script>unsafe()</script>',
   developer: { model: '<img src=x>', provider: 'codex', effort: 'low' },
   reviewer: { model: '<img src=x>', provider: 'codex', effort: 'ultra' }
  } } }).body;
  expect(html).toContain('Routing blocked');
  expect(html).toContain('&lt;script>unsafe()&lt;/script>');
  expect(html).toContain('&lt;img src=x>');
  expect(html).toContain('ultra');
  expect(html).not.toContain('<script>');
  expect(html).not.toContain('<img');
  expect(html).not.toContain('Eligible route');
 });
});
