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

// Approves a change holding on OPEN-Q through POST /api/openspec/approve (#181) and returns "" on success, else the
// server's error text: a 409 means a file changed on the forge after it was read.
export async function approveChange(
	headers: Record<string, string>,
	change: string,
	branch: string,
	fetcher: typeof fetch = fetch
): Promise<string> {
	try {
		const response = await fetcher('/api/openspec/approve', {
			method: 'POST',
			headers,
			body: JSON.stringify({ change, branch })
		});
		if (response.ok) return '';
		const body: { error?: string } = await response.json().catch(() => ({}));
		return body.error || `HTTP ${response.status}`;
	} catch (error) {
		return error instanceof Error ? error.message : 'Approve failed';
	}
}

export interface ApprovalState {
	pending: string; // "<branch>/<change>" while its request runs
	done: Record<string, boolean>; // approved on the forge, waiting for that worktree's pull
	errors: Record<string, string>;
}

export const emptyApprovals: ApprovalState = { pending: '', done: {}, errors: {} };

// Approval results belong to the repository they were sent for: reset() on a repository change clears them and
// drops any approval still in flight, like createSpecPanel.
export function createApprovals(publish: (state: ApprovalState) => void, fetcher: typeof fetch = fetch) {
	let generation = 0;
	return {
		reset() {
			generation++;
			publish(emptyApprovals);
		},
		async approve(headers: Record<string, string>, change: string, branch: string, current: ApprovalState) {
			const request = generation;
			const key = `${branch}/${change}`;
			publish({ ...current, pending: key });
			const error = await approveChange(headers, change, branch, fetcher);
			if (request !== generation) return false;
			publish({ pending: '', done: { ...current.done, [key]: !error }, errors: { ...current.errors, [key]: error } });
			return !error;
		}
	};
}
