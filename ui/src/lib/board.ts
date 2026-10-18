export interface BoardCard {
	id: string;
	number?: number;
	title: string;
	column: string;
	type: string;
	labels?: string[];
	agent?: string;
	branch?: string;
	milestone?: string;
	hold_reason?: string;
	forge?: string;
	url?: string;
	updated_at?: string;
}

export interface Handoff {
	goal: string;
	addresses: string;
	constraints: string;
	done_when: string;
	checks_to_run: string;
	agent?: string;
	branch: string;
	checkpoint_sha: string;
	dirty: boolean;
	auto_compacted?: boolean;
	updated_at?: string;
}

export interface Worktree {
	path: string;
	head: string;
	branch: string;
	agent?: string;
}

export interface GitStatus {
	branch: string;
	head_sha: string;
	dirty: boolean;
	dirty_files?: string[];
	remote_url?: string;
	worktrees?: Worktree[];
}

export interface ShortcutItem {
	file: string;
	line: number;
	limit: string;
	revisit_trigger: string;
	raw: string;
}

export interface HoldItem {
	file: string;
	line: number;
	task_number?: string;
	text: string;
}

export interface OpenSpecCaveat {
	marker: string;
	line: number;
	text: string;
}

export interface OpenSpecChange {
	name: string;
	completed_tasks: number;
	total_tasks: number;
	caveats?: OpenSpecCaveat[];
	stray_holds?: number[];
}

export interface ForgeConfig {
	provider: string;
	base_url?: string;
	owner?: string;
	repo?: string;
}

export interface BenchTaskResult {
	id: string;
	tier: string;
	description: string;
	baseline_loc: number;
	ateam_loc: number;
	loc_delta_pct: number;
	gzip_bytes: number;
	good_passed_safety: boolean;
	bad_caught_by_gate: boolean;
	mutant_caught: boolean;
	baseline_turns: number;
	ateam_turns: number;
	passed: boolean;
}

export interface SuiteReport {
	timestamp: string;
	selftest_passed: boolean;
	total_tasks: number;
	passed_tasks: number;
	avg_loc_reduction_pct: number;
	safety_pass_rate_pct: number;
	mutation_catch_pct: number;
	mcp_token_tax: number;
	hook_latency_us: number;
	pickup_turns_saved: number;
	tasks: BenchTaskResult[];
}

export interface TaskTarget {
	name: string;
	desc: string;
}

export interface DashboardPayload {
	state: {
		mode: string;
		forge: ForgeConfig;
		last_handoff?: Handoff;
		git: GitStatus;
		cards: BoardCard[];
		shortcuts?: ShortcutItem[];
		holds?: HoldItem[];
		openspec_queue?: OpenSpecChange[];
		updated_at: string;
	};
	pickup_text: string;
	benchmarks: SuiteReport;
	tasks: TaskTarget[];
}

export const BOARD_COLUMNS = [
	{ id: 'needs-decision', label: 'Needs Decision', accent: '#f97316' },
	{ id: 'backlog', label: 'Backlog', accent: '#64748b' },
	{ id: 'in-progress', label: 'In Progress', accent: '#3b82f6' },
	{ id: 'in-review', label: 'In Review', accent: '#a855f7' },
	{ id: 'blocked', label: 'Blocked / Hold', accent: '#ef4444' },
	{ id: 'done', label: 'Done', accent: '#10b981' }
] as const;

export interface FilterCriteria {
	query: string;
	agent: string;
	forge: string;
	column: string;
}

export function filterCards(cards: BoardCard[], filter: FilterCriteria): BoardCard[] {
	const q = filter.query.trim().toLowerCase();
	return cards.filter((c) => {
		if (filter.agent && filter.agent !== 'all' && (c.agent || 'unassigned') !== filter.agent) {
			return false;
		}
		if (filter.forge && filter.forge !== 'all' && (c.forge || 'local') !== filter.forge) {
			return false;
		}
		if (filter.column && filter.column !== 'all' && c.column !== filter.column) {
			return false;
		}
		if (!q) {
			return true;
		}
		const haystack = [
			c.id,
			c.title,
			c.branch || '',
			c.agent || '',
			c.milestone || '',
			c.hold_reason || '',
			...(c.labels || [])
		]
			.join(' ')
			.toLowerCase();
		return haystack.includes(q);
	});
}

export function groupCardsByColumn(cards: BoardCard[]): Record<string, BoardCard[]> {
	const grouped: Record<string, BoardCard[]> = {};
	for (const col of BOARD_COLUMNS) {
		grouped[col.id] = [];
	}
	for (const card of cards) {
		const key = grouped[card.column] ? card.column : 'backlog';
		grouped[key].push(card);
	}
	return grouped;
}

export function formatFiveFieldPrompt(h: Handoff): string {
	const val = (v?: string) => (v && v.trim() ? v.trim() : 'none');
	return [
		`Goal: ${val(h.goal)}`,
		`Addresses: ${val(h.addresses)}`,
		`Constraints: ${val(h.constraints)}`,
		`Done when: ${val(h.done_when)}`,
		`Checks to run: ${val(h.checks_to_run)}`
	].join('\n');
}
