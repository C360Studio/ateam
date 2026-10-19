export interface BoardCard {
	id: string;
	number?: number;
	title: string;
	column: string;
	type: string;
	labels?: string[];
	agent?: string;
	branch?: string;
	spec?: string;
	milestone?: string;
	hold_reason?: string;
	decision?: string;
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
	subject?: string;
}

export interface GitStatus {
	branch: string;
	head_sha: string;
	dirty: boolean;
	dirty_files?: string[];
	remote_url?: string;
	user_name?: string;
	user_email?: string;
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

export interface OpenSpecScenario {
	name: string;
	steps?: string[];
}

export interface OpenSpecRequirement {
	id: string;
	name: string;
	spec: string;
	statement: string;
	scenarios?: OpenSpecScenario[];
}

export interface OpenSpecSpec {
	name: string;
	path: string;
	purpose: string;
	requirements?: OpenSpecRequirement[];
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
	ateam_loc: number;
	gzip_bytes: number;
	good_passed_safety: boolean;
	bad_caught_by_gate: boolean;
	mutant_caught: boolean;
	passed: boolean;
}

export interface SuiteReport {
	timestamp: string;
	selftest_passed: boolean;
	total_tasks: number;
	passed_tasks: number;
	safety_pass_rate_pct: number;
	mutation_catch_pct: number;
	mcp_token_tax: number;
	hook_latency_us: number;
	tasks: BenchTaskResult[];
}

export interface TaskTarget {
	name: string;
	desc: string;
}

export interface RepoEntry {
	id: string;
	name: string;
	path: string;
	active: boolean;
}

export interface DashboardPayload {
	state: {
		mode: string;
		forge: ForgeConfig;
		forges?: ForgeConfig[];
		last_handoff?: Handoff;
		git: GitStatus;
		cards: BoardCard[];
		shortcuts?: ShortcutItem[];
		holds?: HoldItem[];
		openspec_queue?: OpenSpecChange[];
		openspec_specs?: OpenSpecSpec[];
		updated_at: string;
	};
	pickup_text: string;
	benchmarks?: SuiteReport;
	tasks: TaskTarget[];
	repos?: RepoEntry[];
	active_repo?: string;
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

export interface UserBadge {
	name: string;
	email: string;
	initials: string;
}

export function formatUserBadge(git?: Partial<GitStatus>): UserBadge {
	const email = git?.user_email?.trim() ?? '';
	let name = git?.user_name?.trim() ?? '';
	if (!name && email.includes('@')) {
		name = email.split('@')[0].trim();
	}
	if (!name) {
		name = 'Local User';
	}
	const parts = name.split(/[\s._-]+/).filter(Boolean);
	let initials = 'LU';
	if (parts.length >= 2) {
		initials = (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
	} else if (parts.length === 1) {
		initials = parts[0].slice(0, 2).toUpperCase();
	}
	return { name, email, initials };
}

export function resolveConfiguredForges(
	primary?: ForgeConfig,
	forges?: ForgeConfig[]
): ForgeConfig[] {
	const out: ForgeConfig[] = [];
	const seen = new Set<string>();
	const add = (f?: ForgeConfig) => {
		const p = f?.provider?.trim().toLowerCase() ?? '';
		if ((p !== 'github' && p !== 'gitea') || seen.has(p)) {
			return;
		}
		seen.add(p);
		out.push({
			provider: p,
			base_url: f?.base_url,
			owner: f?.owner,
			repo: f?.repo
		});
	};
	for (const f of forges ?? []) {
		add(f);
	}
	add(primary);
	return out;
}

export interface OpsItem {
	id: string;
	kind: 'worktree' | 'claim';
	agent: string;
	branch: string;
	head: string;
	dirty: boolean;
	path?: string;
	title: string;
	cardId?: string;
	statusColumn?: string;
	prNumber?: string;
	prUrl?: string;
	goal: string;
	addresses: string;
	constraints: string;
	doneWhen: string;
	checksToRun: string;
	writerStatus: string;
	drifted: boolean;
}

function inferAgentFromBranch(branch?: string): string {
	const b = branch?.trim().toLowerCase() ?? '';
	if (b.startsWith('claude/')) return 'claude';
	if (b.startsWith('codex/')) return 'codex';
	if (b.startsWith('gemini/')) return 'gemini';
	return '';
}

function extractPRRef(
	cards: BoardCard[],
	branch?: string,
	...texts: (string | undefined)[]
): { prNumber?: string; prUrl?: string } {
	if (branch) {
		const prCard = cards.find((c) => c.type === 'pr' && c.branch === branch);
		if (prCard) {
			const num =
				prCard.number ??
				(prCard.id.match(/(\d+)$/) ? Number(prCard.id.match(/(\d+)$/)![1]) : undefined);
			return {
				prNumber: num ? `PR #${num}` : prCard.id,
				prUrl: prCard.url
			};
		}
	}
	for (const raw of texts) {
		if (!raw) continue;
		const m = raw.match(/\bPR\s*#(\d+)\b/i) ?? raw.match(/\/pull\/(\d+)\b/i);
		const urlMatch = raw.match(/https?:\/\/[^\s)]+\/pull\/\d+/i);
		if (m) {
			return {
				prNumber: `PR #${m[1]}`,
				prUrl: urlMatch ? urlMatch[0] : undefined
			};
		}
	}
	return {};
}

export function buildOpsItems(state?: DashboardPayload['state']): OpsItem[] {
	if (!state) return [];
	const cards = state.cards ?? [];
	const handoff = state.last_handoff;
	const rawWorktrees =
		state.git?.worktrees && state.git.worktrees.length > 0
			? state.git.worktrees
			: [
					{
						path: '.',
						branch: state.git?.branch || 'main',
						head: state.git?.head_sha || 'HEAD',
						agent: inferAgentFromBranch(state.git?.branch)
					}
				];

	const matchedCardIds = new Set<string>();
	const items: OpsItem[] = [];
	const handoffMatchesAnyWorktree = Boolean(
		handoff?.branch && rawWorktrees.some((w) => w.branch === handoff.branch)
	);

	rawWorktrees.forEach((wt, idx) => {
		const branch = wt.branch || 'detached';
		const isMainWorktree = idx === 0;
		const handoffMatches = Boolean(
			handoff && (handoff.branch === branch || (!handoffMatchesAnyWorktree && isMainWorktree))
		);

		const matchedCard = cards.find(
			(c) =>
				c.column !== 'done' &&
				((c.branch && c.branch === branch) ||
					(!c.branch &&
						c.agent &&
						wt.agent &&
						c.agent === wt.agent &&
						(c.column === 'in-progress' || c.column === 'in-review')))
		);
		if (matchedCard) {
			matchedCardIds.add(matchedCard.id);
		}

		const agent =
			wt.agent ||
			inferAgentFromBranch(branch) ||
			matchedCard?.agent ||
			(handoffMatches && handoff?.agent ? handoff.agent : '') ||
			'local';

		const { prNumber, prUrl } = extractPRRef(
			cards,
			branch,
			matchedCard?.type === 'pr'
				? `PR #${matchedCard.number ?? matchedCard.id}`
				: matchedCard?.title,
			handoffMatches ? handoff?.addresses : undefined,
			handoffMatches ? handoff?.goal : undefined,
			wt.subject
		);

		const dirty = isMainWorktree ? Boolean(state.git?.dirty) : false;
		const drifted = Boolean(
			handoffMatches && handoff?.checkpoint_sha && wt.head && handoff.checkpoint_sha !== wt.head
		);

		const goal =
			(handoffMatches && handoff?.goal?.trim()) ||
			(matchedCard ? `${matchedCard.id}: ${matchedCard.title}` : '') ||
			wt.subject?.trim() ||
			`Active worktree on ${branch}`;

		const title = matchedCard
			? `${matchedCard.id}: ${matchedCard.title}`
			: handoffMatches && handoff?.goal?.trim()
				? handoff.goal.trim()
				: wt.subject?.trim()
					? wt.subject.trim()
					: `Active worktree (${branch})`;

		const addresses =
			(handoffMatches && handoff?.addresses?.trim()) ||
			(matchedCard ? matchedCard.id : '') ||
			prNumber ||
			`branch ${branch} @ ${wt.head || 'HEAD'}`;

		const constraintsParts: string[] = [];
		if (matchedCard?.hold_reason) constraintsParts.push(matchedCard.hold_reason);
		if (handoffMatches && handoff?.constraints?.trim()) {
			constraintsParts.push(handoff.constraints.trim());
		}
		if (constraintsParts.length === 0 && state.holds && state.holds.length > 0) {
			constraintsParts.push(`${state.holds.length} repo Hold marker(s)`);
		}
		const constraints = constraintsParts.length > 0 ? constraintsParts.join(' · ') : 'None';

		const doneWhen =
			(handoffMatches && handoff?.done_when?.trim()) || 'task verify green on a clean tree';
		const checksToRun = (handoffMatches && handoff?.checks_to_run?.trim()) || 'task verify';

		const statusParts = [
			`${branch}@${wt.head || 'HEAD'}`,
			dirty ? 'uncommitted changes' : 'clean tree'
		];
		if (drifted && handoff?.checkpoint_sha) {
			statusParts.push(`HEAD drifted from ${handoff.checkpoint_sha}`);
		}

		items.push({
			id: `wt:${wt.path || branch}`,
			kind: 'worktree',
			agent,
			branch,
			head: wt.head || 'HEAD',
			dirty,
			path: wt.path,
			title,
			cardId: matchedCard?.id,
			statusColumn: matchedCard?.column,
			prNumber,
			prUrl: prUrl || (matchedCard?.type === 'pr' ? matchedCard.url : undefined),
			goal,
			addresses,
			constraints,
			doneWhen,
			checksToRun,
			writerStatus: statusParts.join(' · '),
			drifted
		});
	});

	for (const card of cards) {
		if (matchedCardIds.has(card.id)) continue;
		const isActiveCol =
			card.column === 'in-progress' ||
			card.column === 'in-review' ||
			card.column === 'blocked' ||
			card.column === 'needs-decision';
		if (!isActiveCol) continue;
		if (!card.agent && !card.branch && card.column !== 'in-progress' && card.column !== 'in-review') {
			continue;
		}

		const { prNumber, prUrl } = extractPRRef(
			cards,
			card.branch,
			card.type === 'pr' ? `PR #${card.number ?? card.id}` : card.title,
			card.id
		);

		items.push({
			id: `claim:${card.id}`,
			kind: 'claim',
			agent: card.agent || inferAgentFromBranch(card.branch) || 'unassigned',
			branch: card.branch || 'no branch',
			head: card.column,
			dirty: false,
			title: `${card.id}: ${card.title}`,
			cardId: card.id,
			statusColumn: card.column,
			prNumber,
			prUrl: prUrl || card.url,
			goal: card.title,
			addresses: prNumber ? `${card.id} (${prNumber})` : card.id,
			constraints: card.hold_reason || 'None',
			doneWhen: 'PR merged and verification gates green',
			checksToRun: 'task verify',
			writerStatus: `Board claim (${card.column})${card.branch ? ` · branch ${card.branch}` : ' · no local worktree'}`,
			drifted: false
		});
	}

	return items;
}

const DOMAIN_CLUSTERS = [
	['hook', 'hooks', 'lifecycle', '5ms'],
	['mcp', 'ateam_state', 'ateam_task', '300 tokens'],
	['openspec', 'spec:queue', 'spec:check', 'spec-queue', 'spec-check'],
	['gitea', 'dual-forge'],
	['webhook', 'webhooks', 'hmac', 'merge-check'],
	['benchmark', 'bench', 'mutate', 'mutation', '-overlay', 'continuity']
];

function containsKeyword(text: string, kw: string): boolean {
	if (kw === 'hook' || kw === 'hooks') {
		return /(?:^|[^a-z])hooks?(?:$|[^a-z])/.test(text);
	}
	return text.includes(kw);
}

export function cardMatchesRequirement(card: BoardCard, req: OpenSpecRequirement): boolean {
	if (card.spec) {
		const sp = card.spec.trim().toLowerCase();
		if (sp === req.id.toLowerCase() || sp === req.name.toLowerCase()) {
			return true;
		}
	}
	const cardText = [card.id, card.title, card.branch || '', ...(card.labels || [])]
		.join(' ')
		.toLowerCase();
	const reqHeader = `${req.id} ${req.name}`.toLowerCase();
	const reqFull = `${reqHeader} ${req.statement}`.toLowerCase();

	if (reqFull.includes(card.id.toLowerCase())) {
		return true;
	}
	for (const cluster of DOMAIN_CLUSTERS) {
		const cardHit = cluster.some((kw) => containsKeyword(cardText, kw));
		const reqHit = cluster.some((kw) => containsKeyword(reqHeader, kw));
		if (cardHit && reqHit) {
			return true;
		}
	}
	return false;
}

export interface CardRelations {
	requirements: OpenSpecRequirement[];
	changes: OpenSpecChange[];
	opsItems: OpsItem[];
}

export function resolveCardRelations(
	card: BoardCard,
	state?: DashboardPayload['state'],
	opsItems: OpsItem[] = []
): CardRelations {
	if (!state) {
		return { requirements: [], changes: [], opsItems: [] };
	}
	const requirements: OpenSpecRequirement[] = [];
	for (const spec of state.openspec_specs ?? []) {
		for (const req of spec.requirements ?? []) {
			if (cardMatchesRequirement(card, req)) {
				requirements.push(req);
			}
		}
	}

	const cardText = [card.id, card.title, card.branch || '', card.spec || '', ...(card.labels || [])]
		.join(' ')
		.toLowerCase();
	const changes = (state.openspec_queue ?? []).filter((ch) => {
		const chName = ch.name.toLowerCase();
		return cardText.includes(chName) || (card.branch && card.branch.toLowerCase().includes(chName));
	});

	const cardPR = card.type === 'pr' && card.number ? `PR #${card.number}` : '';
	const matchedOps = opsItems.filter((op) => {
		if (op.cardId === card.id) return true;
		if (card.branch && op.branch === card.branch) return true;
		if (cardPR && op.prNumber === cardPR) return true;
		if (!card.branch && !card.number && card.column !== 'done' && card.agent && op.agent === card.agent) {
			return true;
		}
		return false;
	});

	return { requirements, changes, opsItems: matchedOps };
}

export function resolveRequirementCards(
	req: OpenSpecRequirement,
	cards: BoardCard[] = []
): BoardCard[] {
	return cards.filter((c) => cardMatchesRequirement(c, req));
}
