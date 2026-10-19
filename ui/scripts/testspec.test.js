import { test } from 'vitest';
import assert from 'node:assert/strict';
import { scanSource as scan } from '../../internal/testspec/vitest.mjs';
import { fileURLToPath } from 'node:url';
const root = fileURLToPath(new URL('../../', import.meta.url));
const scanSource = (file, source) => scan(file, source, root);

// spec: openspec/specs/test-attribution/spec.md#Requirement: Attached test citations
test('finds imported aliases and modifiers without inheriting suite comments', () => {
  const source = `import { it as check, test, describe } from 'vitest';
// suite citation
describe('suite', () => {
  // spec: first
  check('first', () => {});
  test.skip.each([1, 2])('case %s', () => {});
});`;
  assert.deepEqual(scanSource('a.test.ts', source), [
    { file: 'a.test.ts', line: 5, name: 'first', comments: [{ line: 4, text: '// spec: first' }] },
    { file: 'a.test.ts', line: 6, name: 'case %s', comments: [] }
  ]);
});

// spec: openspec/specs/test-attribution/spec.md#Requirement: Attached test citations
test('recognizes namespace, table templates, globals, and conditional declarations', () => {
  const source = `import * as vi from 'vitest';
vi.test.concurrent('namespace', () => {});
vi.it.each\`a | b\n1 | 2\`('template', () => {});
it.runIf(true)('global', () => {});
vi.test.each([1]).skipIf(false)('combined', () => {});`;
  assert.deepEqual(scanSource('a.test.ts', source).map(({ name }) => name), ['namespace', 'template', 'global', 'combined']);
});

// spec: openspec/specs/test-attribution/spec.md#Requirement: Attached test citations
test('rejects aliases, computed access, unknown modifiers, and parse errors', () => {
  for (const source of [
    `import { test } from 'vitest'; const alias = test; alias('hidden', () => {});`,
    `import * as v from 'vitest'; v['test']('hidden', () => {});`,
    `import { test } from 'vitest'; test.something('hidden', () => {});`,
    `import { test } from 'vitest'; test('broken', () => {`,
    `const { test } = require('vitest'); test('hidden', () => {});`,
    `const v = await import('vitest'); v.test('hidden', () => {});`,
    `import v from 'vitest'; v.test('hidden', () => {});`
  ]) {
    assert.ok(scanSource('a.test.ts', source).some(record => record.error), source);
  }
});

// spec: openspec/specs/test-attribution/spec.md#Requirement: Attached test citations
test('ignores unrelated calls, strings, nested comments, and shadowed identifiers', () => {
  const source = `import { test } from 'vitest';
const text = "test('fake')";
function helper(test) { test('unrelated'); }
test('real', () => {
  // spec: body does not attribute declaration
});`;
  assert.deepEqual(scanSource('a.test.ts', source), [{ file: 'a.test.ts', line: 4, name: 'real', comments: [] }]);
});

// spec: openspec/specs/test-attribution/spec.md#Requirement: Attached test citations
test('blank lines detach comments and only the nearest contiguous group attaches', () => {
  const source = `import { test } from 'vitest';
// spec: detached

test('none', () => {});
// spec: older

// spec: attached
// second line
test('nearest', () => {});`;
  assert.deepEqual(scanSource('a.test.ts', source), [
    { file: 'a.test.ts', line: 4, name: 'none', comments: [] },
    { file: 'a.test.ts', line: 9, name: 'nearest', comments: [
      { line: 7, text: '// spec: attached' },
      { line: 8, text: '// second line' }
    ] }
  ]);
});
