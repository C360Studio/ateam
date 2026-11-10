import type { DashboardPayload, WorktreeChange } from './board';

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

export interface SpecPanelState {
	changes: WorktreeChange[];
	skipped: string[];
	loading: boolean;
	error: string;
}

export const emptySpecPanel: SpecPanelState = { changes: [], skipped: [], loading: false, error: '' };

// Like the dashboard loader: reset() on a repository change drops every request in flight, and only the latest
// load may publish rows, loading or an error, so one repository's proposals never show under another (#181).
export function createSpecPanel(publish: (state: SpecPanelState) => void, fetcher: typeof fetch = fetch) {
	let generation = 0;
	return {
		reset() {
			generation++;
			publish(emptySpecPanel);
		},
		async load(headers: Record<string, string>, current: SpecPanelState) {
			const request = ++generation;
			publish({ ...current, loading: true, error: '' });
			try {
				const response = await fetcher('/api/openspec', { headers });
				if (!response.ok) throw new Error(`HTTP ${response.status}`);
				const body: { worktree_changes?: WorktreeChange[]; worktree_skipped?: string[] } = await response.json();
				if (request !== generation) return;
				publish({ changes: body.worktree_changes ?? [], skipped: body.worktree_skipped ?? [], loading: false, error: '' });
			} catch (error) {
				if (request !== generation) return;
				publish({
					...emptySpecPanel,
					error: error instanceof Error ? error.message : 'Failed to load in-flight changes'
				});
			}
		}
	};
}
