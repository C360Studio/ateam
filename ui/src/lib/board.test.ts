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
	it('groups cards across all 6 canonical columns', () => {
		const grouped = groupCardsByColumn(sampleCards);
		expect(Object.keys(grouped)).toHaveLength(BOARD_COLUMNS.length);
		expect(grouped['done']).toHaveLength(1);
		expect(grouped['in-progress']).toHaveLength(1);
		expect(grouped['needs-decision']).toHaveLength(1);
		expect(grouped['backlog']).toHaveLength(0);
	});

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
