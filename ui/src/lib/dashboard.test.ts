import { expect, it } from 'vitest';
import { approveChange, createApprovals, emptyApprovals, type ApprovalState, createDashboardLoader, createSpecPanel, emptySpecPanel, type SpecPanelState } from './dashboard';

function deferred() {
	let resolve!: (response: Response) => void;
	let reject!: (error: Error) => void;
	const promise = new Promise<Response>((yes, no) => { resolve = yes; reject = no; });
	return { promise, resolve, reject };
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
it.each(['success', 'failure'] as const)('ignores a late %s after another repository load', async (outcome) => {
	const old = deferred();
	const current = deferred();
	const queue = [old, current];
	const loader = createDashboardLoader(() => queue.shift()!.promise);
	const first = loader.load({});
	loader.invalidate();
	const second = loader.load({});
	current.resolve(Response.json({ active_repo: '/new' }));
	expect(await second).toEqual({ status: 'success', payload: { active_repo: '/new' } });
	if (outcome === 'success') old.resolve(Response.json({ active_repo: '/old' }));
	else old.reject(new Error('old request failed'));
	expect(await first).toEqual({ status: 'stale' });
});

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
it('reports the current load failure without returning previous repository data', async () => {
	const loader = createDashboardLoader(async () => new Response('', { status: 403 }));
	expect(await loader.load({})).toEqual({ status: 'failure', error: 'HTTP 403' });
});

function specBody(name: string) {
	return Response.json({ worktree_changes: [{ name, branch: `claude/${name}`, worktree: `/wt/${name}`, completed_tasks: 0, total_tasks: 1 }] });
}

function specHarness(queue: ReturnType<typeof deferred>[]) {
	const published: SpecPanelState[] = [];
	let state = emptySpecPanel;
	const panel = createSpecPanel((next) => {
		state = next;
		published.push(next);
	}, () => queue.shift()!.promise);
	return { panel, published, current: () => state };
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
it('clears a completed repository A panel when the repository changes', async () => {
	const a = deferred();
	const { panel, current } = specHarness([a]);
	const load = panel.load({}, current());
	a.resolve(specBody('alpha'));
	await load;
	expect(current().changes.map((c) => c.name)).toEqual(['alpha']);
	panel.reset();
	expect(current()).toEqual(emptySpecPanel);
});

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
it.each(['success', 'failure'] as const)('ignores a late repository A %s after B loads', async (outcome) => {
	const a = deferred();
	const b = deferred();
	const { panel, published, current } = specHarness([a, b]);
	const first = panel.load({}, current());
	panel.reset();
	const second = panel.load({}, current());
	b.resolve(specBody('beta'));
	await second;
	const afterB = published.length;
	if (outcome === 'success') a.resolve(specBody('alpha'));
	else a.reject(new Error('A failed'));
	await first;
	expect(published.length).toBe(afterB);
	expect(current()).toEqual({ ...emptySpecPanel, changes: [expect.objectContaining({ name: 'beta' })] });
});

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
it('keeps the newer refresh when an older one completes last', async () => {
	const older = deferred();
	const newer = deferred();
	const { panel, current } = specHarness([older, newer]);
	const first = panel.load({}, current());
	const second = panel.load({}, current());
	newer.resolve(specBody('newer'));
	await second;
	older.reject(new Error('older failed'));
	await first;
	expect(current()).toEqual({ ...emptySpecPanel, changes: [expect.objectContaining({ name: 'newer' })] });
});

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
it('posts the change and branch, and returns the server error on a stale file', async () => {
	const sent: { url: string; init?: RequestInit }[] = [];
	const answers = [
		Response.json({ change: 'alpha' }),
		Response.json({ error: 'tasks.md: file changed on the forge since it was read; refresh and approve again' }, { status: 409 }),
		new Response('', { status: 502 })
	];
	const fetcher = async (url: RequestInfo | URL, init?: RequestInit) => {
		sent.push({ url: String(url), init });
		return answers.shift()!;
	};
	expect(await approveChange({ 'Content-Type': 'application/json' }, 'alpha', 'claude/alpha', fetcher)).toBe('');
	expect(sent[0].url).toBe('/api/openspec/approve');
	expect(sent[0].init?.method).toBe('POST');
	expect(JSON.parse(String(sent[0].init?.body))).toEqual({ change: 'alpha', branch: 'claude/alpha' });
	expect(await approveChange({}, 'alpha', 'claude/alpha', fetcher)).toMatch(/changed on the forge/);
	expect(await approveChange({}, 'alpha', 'claude/alpha', fetcher)).toBe('HTTP 502');
});

function approvalHarness(answers: ReturnType<typeof deferred>[]) {
	let state: ApprovalState = emptyApprovals;
	const runner = createApprovals((next) => (state = next), () => answers.shift()!.promise);
	return { runner, current: () => state };
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
it('clears a completed approval when the repository changes', async () => {
	const a = deferred();
	const { runner, current } = approvalHarness([a]);
	const done = runner.approve({}, 'alpha', 'claude/alpha', current());
	a.resolve(Response.json({ change: 'alpha' }));
	expect(await done).toBe(true);
	expect(current().done['claude/alpha/alpha']).toBe(true);
	runner.reset();
	expect(current()).toEqual(emptyApprovals);
});

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
it.each(['success', 'failure'] as const)('drops an approval %s that completes after a repository change', async (outcome) => {
	const a = deferred();
	const { runner, current } = approvalHarness([a]);
	const pending = runner.approve({}, 'alpha', 'claude/alpha', current());
	expect(current().pending).toBe('claude/alpha/alpha');
	runner.reset();
	if (outcome === 'success') a.resolve(Response.json({ change: 'alpha' }));
	else a.resolve(Response.json({ error: 'stale' }, { status: 409 }));
	expect(await pending).toBe(false);
	expect(current()).toEqual(emptyApprovals);
});
