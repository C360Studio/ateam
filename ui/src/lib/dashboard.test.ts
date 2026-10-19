import { expect, it } from 'vitest';
import { createDashboardLoader } from './dashboard';

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
