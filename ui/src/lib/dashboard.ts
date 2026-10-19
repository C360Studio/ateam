import type { DashboardPayload } from './board';

type LoadResult =
	| { status: 'success'; payload: DashboardPayload }
	| { status: 'failure'; error: string }
	| { status: 'stale' };

// A repository change invalidates responses already in flight, including failures.
export function createDashboardLoader(fetcher: typeof fetch = fetch) {
	let generation = 0;
	return {
		invalidate() { generation++; },
		async load(headers: Record<string, string>): Promise<LoadResult> {
			const request = ++generation;
			try {
				const response = await fetcher('/api/state', { headers });
				if (!response.ok) throw new Error(`HTTP ${response.status}`);
				const payload: DashboardPayload = await response.json();
				return request === generation ? { status: 'success', payload } : { status: 'stale' };
			} catch (error) {
				return request === generation
					? { status: 'failure', error: error instanceof Error ? error.message : 'Failed to load state' }
					: { status: 'stale' };
			}
		}
	};
}
