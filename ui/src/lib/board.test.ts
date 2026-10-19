import { describe, expect, it } from 'vitest';
import {
	BOARD_COLUMNS,
	filterCards,
	formatFiveFieldPrompt,
	groupCardsByColumn,
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
});
