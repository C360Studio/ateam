<script lang="ts">
	import { onMount } from 'svelte';
	import ModelRoutingPanel from '$lib/ModelRoutingPanel.svelte';
	import {
		BOARD_COLUMNS,
		filterCards,
		groupCardsByColumn,
		type BoardCard,
		type DashboardPayload
	} from '$lib/board';

	let payload = $state<DashboardPayload | null>(null);
	let loading = $state(true);
	let errorMsg = $state('');
	let statusNotice = $state('');
	let apiToken = $state('');

	let activeTab = $state<'board' | 'table' | 'openspec' | 'benchmarks'>('board');
	let searchQuery = $state('');
	let filterAgent = $state('all');
	let filterForge = $state('all');
	let filterColumn = $state('all');

	let showNewCardModal = $state(false);
	let newCardTitle = $state('');
	let newCardColumn = $state('backlog');
	let newCardType = $state('issue');
	let newCardAgent = $state('');
	let newCardForge = $state('github');
	let newCardLabels = $state('');
	let newCardHold = $state('');

	let showHandoffModal = $state(false);
	let hoGoal = $state('');
	let hoAddresses = $state('');
	let hoConstraints = $state('');
	let hoDoneWhen = $state('task verify green on a clean tree');
	let hoChecks = $state('task verify && ateam doctor --check');

	const cards = $derived(payload?.state?.cards ?? []);
	const filtered = $derived(
		filterCards(cards, {
			query: searchQuery,
			agent: filterAgent,
			forge: filterForge,
			column: filterColumn
		})
	);
	const grouped = $derived(groupCardsByColumn(filtered));
	const shortcuts = $derived(payload?.state?.shortcuts ?? []);
	const holds = $derived(payload?.state?.holds ?? []);
	const openspecQueue = $derived(payload?.state?.openspec_queue ?? []);
	const worktrees = $derived(payload?.state?.git?.worktrees ?? []);

	function authHeaders(json = false): Record<string, string> {
		const h: Record<string, string> = {};
		if (json) {
			h['Content-Type'] = 'application/json';
		}
		if (apiToken.trim()) {
			h['X-ATeam-Token'] = apiToken.trim();
		}
		return h;
	}

	function saveApiToken(val: string) {
		apiToken = val;
		try {
			if (val.trim()) {
				localStorage.setItem('ateam_api_token', val.trim());
			} else {
				localStorage.removeItem('ateam_api_token');
			}
		} catch {
			// localStorage unavailable
		}
		loadDashboard();
	}

	async function loadDashboard() {
		loading = true;
		errorMsg = '';
		try {
			const res = await fetch('/api/state', { headers: authHeaders(false) });
			if (!res.ok) {
				throw new Error(`HTTP ${res.status}`);
			}
			const data: DashboardPayload = await res.json();
			payload = data;
			if (data.state.last_handoff) {
				hoGoal = data.state.last_handoff.goal;
				hoAddresses = data.state.last_handoff.addresses;
				hoConstraints = data.state.last_handoff.constraints;
				hoDoneWhen = data.state.last_handoff.done_when;
				hoChecks = data.state.last_handoff.checks_to_run;
			}
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : 'Failed to load state';
		} finally {
			loading = false;
		}
	}

	async function updateMode(mode: string) {
		try {
			const res = await fetch('/api/mode', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({ mode })
			});
			if (!res.ok) {
				throw new Error(`HTTP ${res.status}`);
			}
			if (payload) {
				payload.state.mode = mode;
				flashNotice(`Ponytail mode switched to ${mode.toUpperCase()}`);
			}
		} catch (err) {
			errorMsg = err instanceof Error ? `Failed to switch Ponytail mode (${err.message})` : 'Failed to switch Ponytail mode';
		}
	}

	async function moveCard(id: string, column: string, agent?: string) {
		try {
			const res = await fetch('/api/cards/move', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({ id, column, agent })
			});
			if (!res.ok) {
				throw new Error(`Failed to move card (HTTP ${res.status})`);
			}
			const updated: BoardCard = await res.json();
			if (payload) {
				payload.state.cards = payload.state.cards.map((c) => (c.id === updated.id ? updated : c));
			}
			flashNotice(`Moved ${id} -> ${column}`);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : 'Failed to move card';
		}
	}

	async function createCard(e: SubmitEvent) {
		e.preventDefault();
		if (!newCardTitle.trim()) return;
		const labels = newCardLabels
			.split(',')
			.map((s) => s.trim())
			.filter(Boolean);
		try {
			const res = await fetch('/api/cards', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({
					title: newCardTitle.trim(),
					column: newCardColumn,
					type: newCardType,
					agent: newCardAgent.trim(),
					forge: newCardForge,
					labels,
					hold_reason: newCardHold.trim()
				})
			});
			if (!res.ok) {
				throw new Error(`Failed to create card (HTTP ${res.status})`);
			}
			const created: BoardCard = await res.json();
			if (payload) {
				payload.state.cards = [...payload.state.cards, created];
			}
			newCardTitle = '';
			newCardLabels = '';
			newCardHold = '';
			showNewCardModal = false;
			flashNotice(`Created card ${created.id}`);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : 'Failed to create card';
		}
	}

	async function saveHandoff(e: SubmitEvent) {
		e.preventDefault();
		if (!hoGoal.trim()) return;
		try {
			const res = await fetch('/api/handoff', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({
					goal: hoGoal.trim(),
					addresses: hoAddresses.trim(),
					constraints: hoConstraints.trim(),
					done_when: hoDoneWhen.trim(),
					checks_to_run: hoChecks.trim()
				})
			});
			if (!res.ok) {
				throw new Error(`Failed to record handoff (HTTP ${res.status})`);
			}
			await loadDashboard();
			showHandoffModal = false;
			flashNotice('Saved 5-field handoff checkpoint');
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : 'Failed to record handoff';
		}
	}

	async function syncForge(provider: string) {
		try {
			const res = await fetch('/api/sync', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({ provider })
			});
			const data = await res.json();
			if (!res.ok) {
				flashNotice(`Forge sync note (${provider}): ${data.error || 'offline fallback active'}`);
				return;
			}
			await loadDashboard();
			flashNotice(`Synced ${data.synced ?? 0} items from ${provider}`);
		} catch {
			flashNotice('Using cached local board state (remote forge unreachable)');
		}
	}

	function flashNotice(msg: string) {
		statusNotice = msg;
		setTimeout(() => {
			if (statusNotice === msg) statusNotice = '';
		}, 3500);
	}

	onMount(() => {
		try {
			apiToken = localStorage.getItem('ateam_api_token') ?? '';
		} catch {
			apiToken = '';
		}
		loadDashboard();
	});
</script>

<div class="shell">
	<header class="topbar">
		<div class="brand">
			<span class="logo-badge">ATeam</span>
			<div>
				<h1>Universal Agentic Harness & PM Board</h1>
				<p class="subtitle">
					Claude Code · Codex CLI · Gemini CLI · OpenSpec · GitHub & Gitea · Ponytail Ladder
				</p>
			</div>
		</div>

		<div class="top-controls">
			{#if payload}
				<div class="pill-group" role="group" aria-label="Ponytail discipline mode">
					<span class="pill-label">Ponytail:</span>
					{#each ['lite', 'full', 'ultra', 'off'] as m}
						<button
							type="button"
							class="mode-btn {payload.state.mode === m ? 'active' : ''}"
							onclick={() => updateMode(m)}
						>
							{m}
						</button>
					{/each}
				</div>

				<div class="status-chip">
					<span class="dot {payload.state.git.dirty ? 'warn' : 'ok'}"></span>
					<code>{payload.state.git.branch || 'main'}@{payload.state.git.head_sha || 'HEAD'}</code>
				</div>

				<div class="status-chip">
					<span>Forge:</span>
					<strong>{payload.state.forge.provider || 'local'}</strong>
					<button type="button" class="link-btn" onclick={() => syncForge('github')}>GH Sync</button>
					<button type="button" class="link-btn" onclick={() => syncForge('gitea')}>Gitea Sync</button>
				</div>
			{/if}

			<div class="status-chip">
				<label for="api-token-input">Token:</label>
				<input
					id="api-token-input"
					type="password"
					placeholder="ATEAM_API_TOKEN"
					value={apiToken}
					onchange={(e) => saveApiToken((e.currentTarget as HTMLInputElement).value)}
					aria-label="API token"
				/>
			</div>

			<button type="button" class="btn secondary" onclick={() => (showHandoffModal = true)}>
				Checkpoint Handoff
			</button>
			<button type="button" class="btn primary" onclick={() => (showNewCardModal = true)}>
				+ New Card
			</button>
		</div>
	</header>

	{#if statusNotice}
		<div class="banner info" role="status">{statusNotice}</div>
	{/if}
	{#if errorMsg}
		<div class="banner error" role="alert">{errorMsg}</div>
	{/if}

	{#if payload}
		<ModelRoutingPanel routing={payload.model_routing} />
		<section class="strip" aria-label="Active worktrees and handoff summary">
			<div class="strip-card">
				<div class="strip-header">
					<strong>Active Worktrees & Agent Claims ({worktrees.length})</strong>
					<span class="muted">Dedicated agent branches (`claude/*`, `codex/*`, `gemini/*`)</span>
				</div>
				<div class="worktree-list">
					{#if worktrees.length === 0}
						<span class="chip">main worktree ({payload.state.git.branch})</span>
					{:else}
						{#each worktrees as wt}
							<span class="chip">
								{#if wt.agent}
									<strong class="agent-tag {wt.agent}">{wt.agent}</strong>
								{/if}
								<code>{wt.branch || 'detached'}</code>
								<small>({wt.head})</small>
							</span>
						{/each}
					{/if}
				</div>
			</div>

			<div class="strip-card">
				<div class="strip-header">
					<strong>5-Field SemEngine Pickup State</strong>
					{#if payload.state.last_handoff}
						{#if payload.state.git.head_sha && payload.state.last_handoff.checkpoint_sha && payload.state.last_handoff.checkpoint_sha !== payload.state.git.head_sha}
							<span class="badge warn">
								HEAD DRIFTED ({payload.state.last_handoff.checkpoint_sha} → {payload.state.git.head_sha})
							</span>
						{:else}
							<span class="badge ok">Checkpoint @ {payload.state.last_handoff.checkpoint_sha}</span>
						{/if}
					{:else}
						<span class="badge">Live Git / OpenSpec Reconciliation</span>
					{/if}
				</div>
				<pre class="pickup-pre">{payload.pickup_text}</pre>
			</div>

			<div class="strip-card metrics-mini">
				<div class="metric-box">
					<span class="metric-val">{payload.benchmarks.safety_pass_rate_pct}% / {payload.benchmarks.mutation_catch_pct}%</span>
					<span class="metric-lbl">Safety &amp; Mutant Catch</span>
				</div>
				<div class="metric-box">
					<span class="metric-val">{payload.benchmarks.mcp_token_tax} tok</span>
					<span class="metric-lbl">MCP Schema (&lt;300)</span>
				</div>
				<div class="metric-box">
					<span class="metric-val">{payload.benchmarks.hook_latency_us} µs</span>
					<span class="metric-lbl">Hook Latency (&lt;5ms)</span>
				</div>
				<div class="metric-box">
					<span class="metric-val">{shortcuts.length}</span>
					<span class="metric-lbl">// shortcut: Debt</span>
				</div>
			</div>
		</section>

		<nav class="tabs-bar" aria-label="Project views">
			<div class="tab-buttons">
				<button
					type="button"
					class="tab-btn {activeTab === 'board' ? 'active' : ''}"
					onclick={() => (activeTab = 'board')}
				>
					Board View ({filtered.length})
				</button>
				<button
					type="button"
					class="tab-btn {activeTab === 'table' ? 'active' : ''}"
					onclick={() => (activeTab = 'table')}
				>
					Table View
				</button>
				<button
					type="button"
					class="tab-btn {activeTab === 'openspec' ? 'active' : ''}"
					onclick={() => (activeTab = 'openspec')}
				>
					OpenSpec & Shortcut Debt ({openspecQueue.length} specs · {shortcuts.length} shortcuts)
				</button>
				<button
					type="button"
					class="tab-btn {activeTab === 'benchmarks' ? 'active' : ''}"
					onclick={() => (activeTab = 'benchmarks')}
				>
					Benchmarks & DoD Gates ({payload.benchmarks.passed_tasks}/{payload.benchmarks.total_tasks} PASS)
				</button>
			</div>

			{#if activeTab === 'board' || activeTab === 'table'}
				<div class="filters">
					<input
						type="search"
						placeholder="Filter cards, branches, labels, holds..."
						bind:value={searchQuery}
						aria-label="Filter cards"
					/>
					<select bind:value={filterAgent} aria-label="Filter by agent">
						<option value="all">All Agents</option>
						<option value="claude">Claude</option>
						<option value="codex">Codex</option>
						<option value="gemini">Gemini</option>
						<option value="unassigned">Unassigned</option>
					</select>
					<select bind:value={filterForge} aria-label="Filter by forge">
						<option value="all">All Forges (GitHub + Gitea + Local)</option>
						<option value="github">GitHub</option>
						<option value="gitea">Gitea</option>
						<option value="local">Local</option>
					</select>
				</div>
			{/if}
		</nav>

		<main class="main-area">
			{#if activeTab === 'board'}
				<div class="kanban-grid">
					{#each BOARD_COLUMNS as col}
						{@const colCards = grouped[col.id] ?? []}
						<section class="kanban-col" aria-label={col.label}>
							<header class="col-header" style="border-top-color: {col.accent}">
								<span class="col-title">{col.label}</span>
								<span class="col-count">{colCards.length}</span>
							</header>
							<div class="col-body">
								{#each colCards as card (card.id)}
									<article class="card">
										<div class="card-top">
											<span class="card-id">{card.id}</span>
											<div class="card-badges">
												<span class="badge forge-{card.forge || 'local'}">{card.forge || 'local'}</span>
												<span class="badge type-{card.type}">{card.type}</span>
												{#if card.agent}
													<span class="agent-tag {card.agent}">{card.agent}</span>
												{/if}
											</div>
										</div>
										<h3 class="card-title">{card.title}</h3>
										{#if card.branch}
											<div class="card-branch"><code>{card.branch}</code></div>
										{/if}
										{#if card.hold_reason}
											<div class="hold-alert">{card.hold_reason}</div>
										{/if}
										{#if card.labels && card.labels.length > 0}
											<div class="label-row">
												{#each card.labels as lbl}
													<span class="lbl">{lbl}</span>
												{/each}
											</div>
										{/if}
										<div class="card-actions">
											<label class="sr-only" for="move-{card.id}">Move {card.id}</label>
											<select
												id="move-{card.id}"
												value={card.column}
												onchange={(e) => moveCard(card.id, (e.currentTarget as HTMLSelectElement).value, card.agent)}
											>
												{#each BOARD_COLUMNS as dest}
													<option value={dest.id}>{dest.label}</option>
												{/each}
											</select>
											<label class="sr-only" for="agent-{card.id}">Assign agent for {card.id}</label>
											<select
												id="agent-{card.id}"
												value={card.agent || ''}
												onchange={(e) => moveCard(card.id, card.column, (e.currentTarget as HTMLSelectElement).value)}
											>
												<option value="">Unassigned</option>
												<option value="claude">claude</option>
												<option value="codex">codex</option>
												<option value="gemini">gemini</option>
											</select>
										</div>
									</article>
								{/each}
							</div>
						</section>
					{/each}
				</div>
			{:else if activeTab === 'table'}
				<div class="panel">
					<table class="data-table">
						<thead>
							<tr>
								<th>ID</th>
								<th>Title</th>
								<th>Status Column</th>
								<th>Type</th>
								<th>Forge</th>
								<th>Agent Claim</th>
								<th>Branch / Milestone</th>
								<th>Labels / Hold</th>
							</tr>
						</thead>
						<tbody>
							{#each filtered as card (card.id)}
								<tr>
									<td><code>{card.id}</code></td>
									<td><strong>{card.title}</strong></td>
									<td>
										<select
											aria-label="Column for {card.id}"
											value={card.column}
											onchange={(e) => moveCard(card.id, (e.currentTarget as HTMLSelectElement).value, card.agent)}
										>
											{#each BOARD_COLUMNS as dest}
												<option value={dest.id}>{dest.label}</option>
											{/each}
										</select>
									</td>
									<td><span class="badge type-{card.type}">{card.type}</span></td>
									<td><span class="badge forge-{card.forge || 'local'}">{card.forge || 'local'}</span></td>
									<td>
										<select
											aria-label="Agent for {card.id}"
											value={card.agent || ''}
											onchange={(e) => moveCard(card.id, card.column, (e.currentTarget as HTMLSelectElement).value)}
										>
											<option value="">Unassigned</option>
											<option value="claude">claude</option>
											<option value="codex">codex</option>
											<option value="gemini">gemini</option>
										</select>
									</td>
									<td>
										{#if card.branch}<code>{card.branch}</code>{/if}
										{#if card.milestone}<span class="muted"> · {card.milestone}</span>{/if}
									</td>
									<td>
										{#if card.hold_reason}
											<span class="hold-inline">{card.hold_reason}</span>
										{/if}
										{#each card.labels ?? [] as lbl}
											<span class="lbl">{lbl}</span>
										{/each}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else if activeTab === 'openspec'}
				<div class="split-grid">
					<div class="panel">
						<h2>OpenSpec In-Flight Queue (`task spec:queue`)</h2>
						<p class="muted">
							Native Go parser for `openspec/changes/*/tasks.md` — surfaces `WONTDO`, `HALT`, `RED`, `BLOCKED`, `OPEN-Q`, and `Hold:` items without Node.
						</p>
						{#if openspecQueue.length === 0}
							<div class="empty-box">
								Queue is empty (`0 change(s) in flight`). Canonical specs under `openspec/specs/harness/spec.md` are active and validated by `task spec:check`.
							</div>
						{:else}
							{#each openspecQueue as ch}
								<div class="spec-card">
									<div class="spec-head">
										<strong>{ch.name}</strong>
										<span>{ch.completed_tasks}/{ch.total_tasks} tasks complete</span>
									</div>
									{#if ch.caveats && ch.caveats.length > 0}
										<ul class="caveat-list">
											{#each ch.caveats as cav}
												<li>
													<span class="badge warn">{cav.marker}</span>
													<code>L{cav.line}</code>
													{cav.text}
												</li>
											{/each}
										</ul>
									{:else}
										<p class="muted">ok — no halt/hold/deliberate marker in open tasks</p>
									{/if}
								</div>
							{/each}
						{/if}

						{#if holds.length > 0}
							<h3>Active `Hold:` Markers ({holds.length})</h3>
							<ul class="caveat-list">
								{#each holds as h}
									<li>
										<code>{h.file}:{h.line}</code> — {h.text}
									</li>
								{/each}
							</ul>
						{/if}
					</div>

					<div class="panel">
						<h2>Ponytail `// shortcut:` Tech Debt Inventory ({shortcuts.length})</h2>
						<p class="muted">
							Scanned live across the repository by `ateam state debt`. Every intentional deferral records its limit and upgrade trigger.
						</p>
						<table class="data-table">
							<thead>
								<tr>
									<th>Location</th>
									<th>Known Limit</th>
									<th>When to Upgrade</th>
								</tr>
							</thead>
							<tbody>
								{#each shortcuts as item}
									<tr>
										<td><code>{item.file}:{item.line}</code></td>
										<td>{item.limit}</td>
										<td>{item.revisit_trigger}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{:else if activeTab === 'benchmarks'}
				<div class="split-grid">
					<div class="panel">
						<h2>Ponytail + Continuity Benchmark Scorecard (`ateam bench --selftest`)</h2>
						<p class="muted">
							Verifies reference fixture LOC/gzip size, adversarial AST safety gates, injected mutant detection, and cross-agent handoff continuity.
						</p>
						<table class="data-table">
							<thead>
								<tr>
									<th>Task ID</th>
									<th>Tier</th>
									<th>Fixture LOC</th>
									<th>Gzip Bytes</th>
									<th>Safety Gate</th>
									<th>Mutant Caught</th>
									<th>Status</th>
								</tr>
							</thead>
							<tbody>
								{#each payload.benchmarks.tasks as t}
									<tr>
										<td><code>{t.id}</code></td>
										<td><span class="badge">{t.tier}</span></td>
										<td><strong>{t.ateam_loc}</strong></td>
										<td>{t.gzip_bytes} B</td>
										<td>{t.good_passed_safety && t.bad_caught_by_gate ? 'PASS' : 'FAIL'}</td>
										<td>{t.mutant_caught ? 'CAUGHT' : 'MISSED'}</td>
										<td class="text-ok">{t.passed ? 'PASS' : 'FAIL'}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					<div class="panel">
						<h2>Standardized `Taskfile.yml` Verification Gates</h2>
						<p class="muted">
							Exposed to Claude, Codex, and Gemini via the low-tax `ateam_task` MCP tool.
						</p>
						<table class="data-table">
							<thead>
								<tr>
									<th>Target</th>
									<th>Description</th>
								</tr>
							</thead>
							<tbody>
								{#each payload.tasks as target}
									<tr>
										<td><code>task {target.name}</code></td>
										<td>{target.desc}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{/if}
		</main>
	{:else if loading}
		<div class="loading">Loading ATeam harness state...</div>
	{/if}

	{#if showNewCardModal}
		<div class="modal-backdrop" role="dialog" aria-modal="true" aria-label="New board card">
			<form class="modal" onsubmit={createCard}>
				<h2>Create PM Board Card</h2>
				<label>
					Title
					<input type="text" required bind:value={newCardTitle} placeholder="e.g., Add Gitea PR webhook verification" />
				</label>
				<div class="form-row">
					<label>
						Column
						<select bind:value={newCardColumn}>
							{#each BOARD_COLUMNS as col}
								<option value={col.id}>{col.label}</option>
							{/each}
						</select>
					</label>
					<label>
						Type
						<select bind:value={newCardType}>
							<option value="issue">issue</option>
							<option value="epic">epic</option>
							<option value="pr">pr</option>
							<option value="spec">spec</option>
						</select>
					</label>
				</div>
				<div class="form-row">
					<label>
						Forge
						<select bind:value={newCardForge}>
							<option value="github">github</option>
							<option value="gitea">gitea</option>
							<option value="local">local</option>
						</select>
					</label>
					<label>
						Assigned Agent
						<select bind:value={newCardAgent}>
							<option value="">Unassigned</option>
							<option value="claude">claude</option>
							<option value="codex">codex</option>
							<option value="gemini">gemini</option>
						</select>
					</label>
				</div>
				<label>
					Labels (comma-separated)
					<input type="text" bind:value={newCardLabels} placeholder="openspec, gitea, type:epic" />
				</label>
				<label>
					Optional Hold Reason
					<input type="text" bind:value={newCardHold} placeholder="Hold: waiting on #42" />
				</label>
				<div class="modal-actions">
					<button type="button" class="btn secondary" onclick={() => (showNewCardModal = false)}>Cancel</button>
					<button type="submit" class="btn primary">Create Card</button>
				</div>
			</form>
		</div>
	{/if}

	{#if showHandoffModal}
		<div class="modal-backdrop" role="dialog" aria-modal="true" aria-label="Record 5-field handoff checkpoint">
			<form class="modal" onsubmit={saveHandoff}>
				<h2>Record 5-Field SemEngine Handoff Checkpoint</h2>
				<label>
					Goal (1 sentence)
					<input type="text" required bind:value={hoGoal} placeholder="Complete Gitea issue sync and run task verify" />
				</label>
				<label>
					Addresses (Issues / PRs / OpenSpec change)
					<input type="text" bind:value={hoAddresses} placeholder="#3, openspec/changes/dual-forge" />
				</label>
				<label>
					Constraints & Active Holds
					<input type="text" bind:value={hoConstraints} placeholder="Keep MCP schema < 300 tokens; zero external Go deps" />
				</label>
				<label>
					Done When
					<input type="text" bind:value={hoDoneWhen} />
				</label>
				<label>
					Checks to Run
					<input type="text" bind:value={hoChecks} />
				</label>
				<div class="modal-actions">
					<button type="button" class="btn secondary" onclick={() => (showHandoffModal = false)}>Cancel</button>
					<button type="submit" class="btn primary">Save Handoff Checkpoint</button>
				</div>
			</form>
		</div>
	{/if}
</div>

<style>
	:global(body) {
		margin: 0;
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
		background: #0b0f19;
		color: #e2e8f0;
	}
	.shell {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}
	.topbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 1rem;
		padding: 0.9rem 1.4rem;
		background: #111827;
		border-bottom: 1px solid #1e293b;
	}
	.brand {
		display: flex;
		align-items: center;
		gap: 0.85rem;
	}
	.logo-badge {
		background: linear-gradient(135deg, #3b82f6, #8b5cf6);
		color: #fff;
		font-weight: 800;
		font-size: 0.95rem;
		padding: 0.45rem 0.7rem;
		border-radius: 0.5rem;
		letter-spacing: 0.04em;
	}
	h1 {
		margin: 0;
		font-size: 1.1rem;
		font-weight: 700;
	}
	.subtitle {
		margin: 0.15rem 0 0;
		font-size: 0.78rem;
		color: #94a3b8;
	}
	.top-controls {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.65rem;
	}
	.pill-group {
		display: flex;
		align-items: center;
		background: #1e293b;
		padding: 0.2rem;
		border-radius: 0.45rem;
		gap: 0.2rem;
	}
	.pill-label {
		font-size: 0.75rem;
		color: #94a3b8;
		padding: 0 0.4rem;
	}
	.mode-btn {
		background: transparent;
		border: none;
		color: #cbd5e1;
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.28rem 0.55rem;
		border-radius: 0.35rem;
		cursor: pointer;
		text-transform: uppercase;
	}
	.mode-btn.active {
		background: #3b82f6;
		color: #fff;
	}
	.status-chip {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		background: #1e293b;
		padding: 0.32rem 0.65rem;
		border-radius: 0.45rem;
		font-size: 0.78rem;
	}
	.dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		display: inline-block;
	}
	.dot.ok {
		background: #10b981;
	}
	.dot.warn {
		background: #f59e0b;
	}
	.link-btn {
		background: #334155;
		border: none;
		color: #e2e8f0;
		font-size: 0.72rem;
		padding: 0.18rem 0.45rem;
		border-radius: 0.3rem;
		cursor: pointer;
	}
	.btn {
		border: none;
		border-radius: 0.45rem;
		padding: 0.45rem 0.85rem;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
	}
	.btn.primary {
		background: #3b82f6;
		color: #fff;
	}
	.btn.secondary {
		background: #1e293b;
		color: #e2e8f0;
		border: 1px solid #334155;
	}
	.banner {
		padding: 0.55rem 1.4rem;
		font-size: 0.82rem;
		font-weight: 600;
	}
	.banner.info {
		background: #1e3a8a;
		color: #dbeafe;
	}
	.banner.error {
		background: #7f1d1d;
		color: #fecaca;
	}
	.strip {
		display: grid;
		grid-template-columns: 1.1fr 1.4fr 1fr;
		gap: 0.85rem;
		padding: 0.9rem 1.4rem;
		background: #0f172a;
		border-bottom: 1px solid #1e293b;
	}
	.strip-card {
		background: #111827;
		border: 1px solid #1e293b;
		border-radius: 0.55rem;
		padding: 0.75rem 0.9rem;
	}
	.strip-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.8rem;
		margin-bottom: 0.45rem;
	}
	.worktree-list {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		background: #1e293b;
		padding: 0.25rem 0.55rem;
		border-radius: 0.35rem;
		font-size: 0.76rem;
	}
	.pickup-pre {
		margin: 0;
		font-size: 0.74rem;
		line-height: 1.38;
		color: #cbd5e1;
		white-space: pre-wrap;
		max-height: 4.8rem;
		overflow-y: auto;
	}
	.metrics-mini {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.5rem;
	}
	.metric-box {
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.4rem;
		padding: 0.45rem 0.6rem;
		display: flex;
		flex-direction: column;
	}
	.metric-val {
		font-size: 1rem;
		font-weight: 800;
		color: #38bdf8;
	}
	.metric-lbl {
		font-size: 0.7rem;
		color: #94a3b8;
	}
	.tabs-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
		padding: 0.65rem 1.4rem;
		background: #111827;
		border-bottom: 1px solid #1e293b;
	}
	.tab-buttons {
		display: flex;
		gap: 0.4rem;
		flex-wrap: wrap;
	}
	.tab-btn {
		background: transparent;
		border: 1px solid transparent;
		color: #94a3b8;
		padding: 0.42rem 0.8rem;
		border-radius: 0.4rem;
		font-size: 0.82rem;
		font-weight: 600;
		cursor: pointer;
	}
	.tab-btn.active {
		background: #1e293b;
		color: #fff;
		border-color: #334155;
	}
	.filters {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}
	input,
	select {
		background: #0b0f19;
		border: 1px solid #334155;
		color: #e2e8f0;
		padding: 0.38rem 0.6rem;
		border-radius: 0.38rem;
		font-size: 0.78rem;
	}
	.main-area {
		padding: 1rem 1.4rem;
		flex: 1;
	}
	.kanban-grid {
		display: grid;
		grid-template-columns: repeat(6, minmax(220px, 1fr));
		gap: 0.8rem;
		overflow-x: auto;
		padding-bottom: 0.5rem;
	}
	.kanban-col {
		background: #111827;
		border: 1px solid #1e293b;
		border-radius: 0.55rem;
		display: flex;
		flex-direction: column;
		min-height: 420px;
	}
	.col-header {
		padding: 0.65rem 0.8rem;
		border-top: 3px solid #64748b;
		border-bottom: 1px solid #1e293b;
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-weight: 700;
		font-size: 0.82rem;
	}
	.col-count {
		background: #1e293b;
		color: #cbd5e1;
		font-size: 0.72rem;
		padding: 0.12rem 0.45rem;
		border-radius: 999px;
	}
	.col-body {
		padding: 0.65rem;
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
	}
	.card {
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.48rem;
		padding: 0.7rem;
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.card-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.3rem;
	}
	.card-id {
		font-family: monospace;
		font-size: 0.74rem;
		color: #94a3b8;
		font-weight: 700;
	}
	.card-badges {
		display: flex;
		gap: 0.25rem;
		flex-wrap: wrap;
	}
	.card-title {
		margin: 0;
		font-size: 0.84rem;
		line-height: 1.35;
		color: #f8fafc;
	}
	.card-branch {
		font-size: 0.72rem;
		color: #38bdf8;
	}
	.hold-alert {
		background: rgba(239, 68, 68, 0.14);
		border-left: 3px solid #ef4444;
		color: #fca5a5;
		padding: 0.3rem 0.45rem;
		font-size: 0.73rem;
		border-radius: 0.25rem;
	}
	.label-row {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem;
	}
	.lbl {
		background: #1e293b;
		color: #cbd5e1;
		font-size: 0.68rem;
		padding: 0.12rem 0.4rem;
		border-radius: 0.25rem;
	}
	.card-actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.35rem;
		margin-top: 0.2rem;
	}
	.card-actions select {
		font-size: 0.72rem;
		padding: 0.24rem 0.35rem;
	}
	.badge {
		font-size: 0.68rem;
		font-weight: 700;
		padding: 0.12rem 0.4rem;
		border-radius: 0.25rem;
		background: #1e293b;
		color: #cbd5e1;
		text-transform: uppercase;
	}
	.badge.ok {
		background: rgba(16, 185, 129, 0.2);
		color: #6ee7b7;
	}
	.badge.warn {
		background: rgba(245, 158, 11, 0.2);
		color: #fcd34d;
	}
	.badge.forge-github {
		background: #1e293b;
		color: #f8fafc;
	}
	.badge.forge-gitea {
		background: rgba(16, 185, 129, 0.2);
		color: #6ee7b7;
	}
	.agent-tag {
		font-size: 0.68rem;
		font-weight: 700;
		padding: 0.12rem 0.4rem;
		border-radius: 0.25rem;
		text-transform: uppercase;
	}
	.agent-tag.claude {
		background: rgba(217, 119, 6, 0.25);
		color: #fcd34d;
	}
	.agent-tag.codex {
		background: rgba(16, 185, 129, 0.25);
		color: #6ee7b7;
	}
	.agent-tag.gemini {
		background: rgba(59, 130, 246, 0.25);
		color: #93c5fd;
	}
	.panel {
		background: #111827;
		border: 1px solid #1e293b;
		border-radius: 0.55rem;
		padding: 1rem;
	}
	.panel h2 {
		margin: 0 0 0.35rem;
		font-size: 1rem;
	}
	.split-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.8rem;
		margin-top: 0.6rem;
	}
	.data-table th,
	.data-table td {
		text-align: left;
		padding: 0.55rem 0.65rem;
		border-bottom: 1px solid #1e293b;
	}
	.data-table th {
		color: #94a3b8;
		font-size: 0.74rem;
		text-transform: uppercase;
	}
	.text-ok {
		color: #34d399;
		font-weight: 700;
	}
	.muted {
		color: #94a3b8;
		font-size: 0.78rem;
	}
	.empty-box {
		background: #0b0f19;
		border: 1px dashed #334155;
		padding: 0.9rem;
		border-radius: 0.45rem;
		font-size: 0.82rem;
		color: #cbd5e1;
		margin-top: 0.65rem;
	}
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 50;
	}
	.modal {
		background: #111827;
		border: 1px solid #334155;
		border-radius: 0.6rem;
		padding: 1.25rem;
		width: 100%;
		max-width: 480px;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	.modal h2 {
		margin: 0;
		font-size: 1.05rem;
	}
	.modal label {
		display: flex;
		flex-direction: column;
		gap: 0.3rem;
		font-size: 0.78rem;
		color: #cbd5e1;
	}
	.form-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.65rem;
	}
	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
		margin-top: 0.4rem;
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		border: 0;
	}
</style>
