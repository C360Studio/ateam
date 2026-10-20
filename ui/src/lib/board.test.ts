import { describe, expect, it } from 'vitest';
import {
	BOARD_COLUMNS,
	buildOpsItems,
	filterCards,
	formatFiveFieldPrompt,
	formatUserBadge,
	groupCardsByColumn,
	resolveCardRelations,
	resolveConfiguredForges,
	resolveRequirementCards,
	type BoardCard
} from './board';

const sampleCards: BoardCard[] = [
	{
		id: 'ATEAM-1',
		title: 'Universal CLI hooks',
		column: 'done',
		type: 'epic',
		agent: 'gemini',
		forge: 'github',
		labels: ['type:epic']
	},
	{
		id: 'ATEAM-2',
		title: 'Gitea label sync & PR status',
		column: 'in-progress',
		type: 'issue',
		agent: 'codex',
		forge: 'gitea',
		labels: ['gitea']
	},
	{
		id: 'ATEAM-3',
		title: 'Owner decision on ruleset',
		column: 'needs-decision',
		type: 'issue',
		forge: 'gitea',
		hold_reason: 'Hold: awaiting owner decision'
	}
];

describe('board helpers', () => {
	// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
	it('groups cards across all 6 canonical columns', () => {
		const grouped = groupCardsByColumn(sampleCards);
		expect(Object.keys(grouped)).toHaveLength(BOARD_COLUMNS.length);
		expect(grouped['done']).toHaveLength(1);
		expect(grouped['in-progress']).toHaveLength(1);
		expect(grouped['needs-decision']).toHaveLength(1);
		expect(grouped['backlog']).toHaveLength(0);
	});

	// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
	it('filters cards by query, agent, forge, and column', () => {
		expect(
			filterCards(sampleCards, { query: 'gitea', agent: 'all', forge: 'all', column: 'all' })
		).toHaveLength(1);
		expect(
			filterCards(sampleCards, { query: '', agent: 'gemini', forge: 'all', column: 'all' })
		).toHaveLength(1);
		expect(
			filterCards(sampleCards, { query: '', agent: 'all', forge: 'gitea', column: 'all' })
		).toHaveLength(2);
		expect(
			filterCards(sampleCards, {
				query: 'awaiting owner',
				agent: 'all',
				forge: 'gitea',
				column: 'needs-decision'
			})
		).toHaveLength(1);
	});

	// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
	it('formats the 5-field SemEngine pickup prompt deterministically', () => {
		const out = formatFiveFieldPrompt({
			goal: 'Ship SvelteKit UI',
			addresses: 'ATEAM-4',
			constraints: '',
			done_when: 'task verify green',
			checks_to_run: 'task verify',
			branch: 'gemini/ui',
			checkpoint_sha: 'abc1234',
			dirty: false
		});
		expect(out).toContain('Goal: Ship SvelteKit UI');
		expect(out).toContain('Constraints: none');
		expect(out).toContain('Checks to run: task verify');
	});

	// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
	it('derives user badge name, email, and initials from git identity', () => {
		expect(formatUserBadge({ user_name: 'Coby B', user_email: 'coby@c360.dev' })).toEqual({
			name: 'Coby B',
			email: 'coby@c360.dev',
			initials: 'CB'
		});
		expect(formatUserBadge({ user_name: '', user_email: 'alice@example.com' })).toEqual({
			name: 'alice',
			email: 'alice@example.com',
			initials: 'AL'
		});
		expect(formatUserBadge({})).toEqual({
			name: 'Local User',
			email: '',
			initials: 'LU'
		});
	});

	// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
	it('resolves configured forges without duplicates or local placeholders', () => {
		expect(
			resolveConfiguredForges(
				{ provider: 'github', owner: 'C360Studio', repo: 'ateam' },
				[{ provider: 'github', owner: 'C360Studio', repo: 'ateam' }]
			)
		).toEqual([
			{ provider: 'github', base_url: undefined, owner: 'C360Studio', repo: 'ateam' }
		]);
		expect(resolveConfiguredForges({ provider: 'local' }, [])).toEqual([]);
	});

	// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
	it('builds Current Ops items combining worktrees, agent claims, and PR numbers', () => {
		const items = buildOpsItems({
			mode: 'full',
			forge: { provider: 'github', owner: 'C360Studio', repo: 'ateam' },
			last_handoff: {
				goal: 'Ship ATeam v0.1.0 universal harness',
				addresses: 'PR #2 (https://github.com/C360Studio/ateam/pull/2)',
				constraints: 'Zero external Go deps',
				done_when: 'PR #2 merged',
				checks_to_run: 'task verify',
				agent: 'gemini',
				branch: 'gemini/ateam-v1-universal-harness',
				checkpoint_sha: 'a8fc483',
				dirty: false
			},
			git: {
				branch: 'gemini/ateam-v1-universal-harness',
				head_sha: '095cf69',
				dirty: true,
				worktrees: [
					{
						path: '/repo',
						branch: 'gemini/ateam-v1-universal-harness',
						head: '095cf69',
						agent: 'gemini'
					}
				]
			},
			cards: sampleCards,
			updated_at: '2026-04-19T12:00:00Z'
		});
		expect(items).toHaveLength(2);
		expect(items[0].kind).toBe('worktree');
		expect(items[0].agent).toBe('gemini');
		expect(items[0].prNumber).toBe('PR #2');
		expect(items[0].prUrl).toBe('https://github.com/C360Studio/ateam/pull/2');
		expect(items[0].drifted).toBe(true);
		expect(items[1].kind).toBe('claim');
		expect(items[1].cardId).toBe('ATEAM-2');
		expect(items[1].agent).toBe('codex');
	});

	// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
	it('links cards and OpenSpec requirements bidirectionally', () => {
		const reqHooks = {
			id: 'harness:1',
			name: 'Lifecycle hook normalization under 5ms',
			spec: 'harness',
			statement: 'ateam hook SHALL normalize events in under 5ms.',
			scenarios: [{ name: 'Mode switch', steps: ['WHEN /mode ultra', 'THEN updates state.json'] }]
		};
		const reqGitea = {
			id: 'harness:4',
			name: 'Dual-forge GitHub and Gitea state sync',
			spec: 'harness',
			statement: 'ateam SHALL support both GitHub and Gitea.',
			scenarios: []
		};
		const state = {
			mode: 'full',
			forge: { provider: 'github' },
			git: { branch: 'main', head_sha: 'abc', dirty: false },
			cards: sampleCards,
			openspec_specs: [
				{
					name: 'harness',
					path: 'openspec/specs/harness/spec.md',
					purpose: 'Harness spec',
					requirements: [reqHooks, reqGitea]
				}
			],
			updated_at: '2026-04-19T12:00:00Z'
		};
		const rel1 = resolveCardRelations(sampleCards[0], state, []);
		expect(rel1.requirements.map((r) => r.id)).toEqual(['harness:1']);

		const linkedCards = resolveRequirementCards(reqGitea, sampleCards);
		expect(linkedCards.map((c) => c.id)).toEqual(['ATEAM-2']);
	});
});

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
it('attributes dirtiness only to the selected checkout and compares abbreviated checkpoints', () => {
	const state = {
		mode: 'full', forge: { provider: 'local' }, updated_at: '',
		cards: [],
		git: { branch: 'codex/feature', head_sha: 'abcdef12', dirty: true, worktrees: [
			{ path: '/main', branch: 'main', head: '12345678' },
			{ path: '/feature', branch: 'codex/feature', head: 'abcdef12' }
		]},
		last_handoff: { branch: 'codex/feature', checkpoint_sha: 'abcdef1', goal: 'Feature work', addresses: '', constraints: '', done_when: '', checks_to_run: '', dirty: false }
	};
	const items = buildOpsItems(state, '/feature');
	expect(items[0].dirty).toBeNull();
	expect(items[0].writerStatus).toContain('status not inspected');
	expect(items[1].dirty).toBe(true);
	expect(items[1].drifted).toBe(false);
	state.last_handoff.checkpoint_sha = 'fedcba9';
	expect(buildOpsItems(state, '/feature')[1].drifted).toBe(true);
	state.last_handoff.branch = 'removed-branch';
	expect(buildOpsItems(state, '/feature').every((item) => item.goal !== 'Feature work')).toBe(true);
});

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
it('classifies worktree merge status across merged, open, closed-unmerged, post-merge commits, and unsynced forge states', () => {
	// Parse from JSON matching Go encoding/json omitempty output (false/0/"" fields omitted).
	const apiState = JSON.parse(
		JSON.stringify({
			mode: 'full',
			forge: { provider: 'github', owner: 'C360Studio', repo: 'ateam' },
			board_synced_at: '2026-10-20T12:00:00Z',
			git: {
				branch: 'main',
				head_sha: '3ec9dca',
				dirty: false,
				worktrees: [
					{
						path: '/wt/merged-squash',
						branch: 'claude/test-spec-attribution',
						head: 'a1b2c3d4',
						dirty: true,
						ahead_count: 1
					},
					{
						path: '/wt/open-pr',
						branch: 'codex/open-work',
						head: 'b2c3d4e5',
						dirty: false,
						ahead_count: 2
					},
					{
						path: '/wt/closed-unmerged',
						branch: 'gemini/abandoned',
						head: 'c3d4e5f6',
						dirty: false,
						ahead_count: 1
					},
					{
						path: '/wt/reused-after-merge',
						branch: 'claude/reused-branch',
						head: '99999999',
						dirty: false,
						ahead_count: 2
					},
					{
						path: '/wt/missing-pr-head-ahead',
						branch: 'codex/no-pr-sha',
						head: '77777777',
						dirty: false,
						ahead_count: 2
					},
					{
						path: '/wt/ancestor-main',
						branch: 'gemini/fast-forward-merged',
						head: '3ec9dca0',
						dirty: false,
						merged_into_main: true
					}
				]
			},
			cards: [
				{
					id: 'GH-PR-22',
					number: 22,
					title: 'feat: test-spec attribution',
					column: 'done',
					type: 'pr',
					merged: true,
					head_sha: 'a1b2c3d400000000',
					branch: 'claude/test-spec-attribution',
					forge: 'github'
				},
				{
					id: 'GH-PR-24',
					number: 24,
					title: 'feat: open work',
					column: 'in-review',
					type: 'pr',
					head_sha: 'b2c3d4e500000000',
					branch: 'codex/open-work',
					forge: 'github'
				},
				{
					id: 'GH-PR-25',
					number: 25,
					title: 'chore: abandoned experiment',
					column: 'done',
					type: 'pr',
					head_sha: 'c3d4e5f600000000',
					branch: 'gemini/abandoned',
					forge: 'github'
				},
				{
					id: 'GH-PR-26',
					number: 26,
					title: 'feat: first PR on branch',
					column: 'done',
					type: 'pr',
					merged: true,
					head_sha: '1111111100000000',
					branch: 'claude/reused-branch',
					forge: 'github'
				},
				{
					id: 'GH-PR-27',
					number: 27,
					title: 'feat: merged PR without head_sha',
					column: 'done',
					type: 'pr',
					merged: true,
					branch: 'codex/no-pr-sha',
					forge: 'github'
				}
			],
			updated_at: '2026-10-20T12:00:00Z'
		})
	);
	const items = buildOpsItems(apiState);

	const byPath = Object.fromEntries(items.filter((i) => i.kind === 'worktree').map((i) => [i.path, i]));
	// 1. Squash-merged PR with matching head_sha is marked merged while preserving local dirty=true.
	expect(byPath['/wt/merged-squash'].mergeStatus).toBe('merged');
	expect(byPath['/wt/merged-squash'].mergeEvidence).toBe('Merged via PR #22');
	expect(byPath['/wt/merged-squash'].dirty).toBe(true);
	expect(byPath['/wt/merged-squash'].writerStatus).toContain('uncommitted changes');
	expect(byPath['/wt/merged-squash'].writerStatus).toContain('merged (Merged via PR #22)');

	// 2. Open PR is active (not merged).
	expect(byPath['/wt/open-pr'].mergeStatus).toBe('active');

	// 3. Closed-unmerged PR is closed-unmerged (never marked merged).
	expect(byPath['/wt/closed-unmerged'].mergeStatus).toBe('closed-unmerged');
	expect(byPath['/wt/closed-unmerged'].mergeEvidence).toBe('PR #25 closed without merge');

	// 4. Branch with new commits after a merged PR is active (not stale/merged).
	expect(byPath['/wt/reused-after-merge'].mergeStatus).toBe('active');
	expect(byPath['/wt/reused-after-merge'].mergeEvidence).toContain('New commits ahead of merged PR #26');

	// 5. Merged PR lacking head_sha with ahead_count > 0 and omitted merged_into_main (Go omitempty false) stays active.
	expect(byPath['/wt/missing-pr-head-ahead'].mergeStatus).toBe('active');
	expect(byPath['/wt/missing-pr-head-ahead'].mergeEvidence).toBe('Unmerged commits ahead of main');

	// 6. Branch that is an ancestor of main with 0 ahead commits is marked merged.
	expect(byPath['/wt/ancestor-main'].mergeStatus).toBe('merged');
	expect(byPath['/wt/ancestor-main'].mergeEvidence).toBe('Branch ancestor of main');

	// 7. Unavailable forge evidence (unsynced remote forge and no local ancestry) reports unknown, not merged.
	const unsynced = buildOpsItems({
		mode: 'full',
		forge: { provider: 'github', owner: 'C360Studio', repo: 'ateam' },
		git: {
			branch: 'main',
			head_sha: '3ec9dca',
			dirty: false,
			worktrees: [{ path: '/wt/unsynced', branch: 'claude/unverified', head: 'deadbeef' }]
		},
		cards: [],
		updated_at: '2026-10-20T12:00:00Z'
	});
	expect(unsynced[0].mergeStatus).toBe('unknown');
});
