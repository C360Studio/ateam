<script lang="ts">
	import { onMount } from 'svelte';
	import { createDashboardLoader } from '$lib/dashboard';
	const dashboardLoader = createDashboardLoader();
	import ModelRoutingPanel from '$lib/ModelRoutingPanel.svelte';
	import {
		BOARD_COLUMNS,
		buildOpsItems,
		filterCards,
		formatUserBadge,
		groupCardsByColumn,
		resolveCardRelations,
		resolveConfiguredForges,
		resolveRequirementCards,
		type BoardCard,
		type DashboardPayload,
		type OpenSpecRequirement,
		type SuiteReport
	} from '$lib/board';

	let payload = $state<DashboardPayload | null>(null);
	let loading = $state(true);
	let errorMsg = $state('');
	let statusNotice = $state('');
	let apiToken = $state('');

	let activeTab = $state<'board' | 'table' | 'openspec'>('board');
	let searchQuery = $state('');
	let filterAgent = $state('all');
	let filterForge = $state('all');
	let filterColumn = $state('all');

	let selectedOpsId = $state('');
	let benchReport = $state<SuiteReport | null>(null);
	let benchRunning = $state(false);

	let drawerCardId = $state<string | null>(null);
	let drawerReqId = $state<string | null>(null);

	let showNewCardModal = $state(false);
	let newCardTitle = $state('');
	let newCardColumn = $state('backlog');
	let newCardType = $state('issue');
	let newCardAgent = $state('');
	let newCardForge = $state('github');
	let newCardLabels = $state('');
	let newCardHold = $state('');

	let showSettingsModal = $state(false);

	let newRepoPath = $state('');
	let cfgForgeProvider = $state<'github' | 'gitea'>('gitea');
	let cfgForgeBaseURL = $state('');
	let cfgForgeOwner = $state('');
	let cfgForgeRepo = $state('');

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
	const openspecSpecs = $derived(payload?.state?.openspec_specs ?? []);
	const allRequirements = $derived<OpenSpecRequirement[]>(
		openspecSpecs.flatMap((sp) => sp.requirements ?? [])
	);
	const opsItems = $derived(buildOpsItems(payload?.state, payload?.active_repo));
	const selectedOps = $derived(
		opsItems.find((item) => item.id === selectedOpsId) ?? opsItems[0] ?? null
	);
	const drawerCard = $derived(
		drawerCardId ? (cards.find((c) => c.id === drawerCardId) ?? null) : null
	);
	const drawerCardRel = $derived(
		drawerCard ? resolveCardRelations(drawerCard, payload?.state, opsItems) : null
	);
	const drawerReq = $derived(
		drawerReqId ? (allRequirements.find((r) => r.id === drawerReqId) ?? null) : null
	);
	const drawerReqCards = $derived(
		drawerReq ? resolveRequirementCards(drawerReq, cards) : []
	);
	const userBadge = $derived(formatUserBadge(payload?.state?.git));
	const configuredForges = $derived(
		resolveConfiguredForges(payload?.state?.forge, payload?.state?.forges)
	);
	const repos = $derived(payload?.repos ?? []);

	function openCardDrawer(id: string) {
		drawerReqId = null;
		drawerCardId = id;
	}

	function openReqDrawer(id: string) {
		drawerCardId = null;
		drawerReqId = id;
	}

	function closeDrawer() {
		drawerCardId = null;
		drawerReqId = null;
	}

	function selectOpsFromDrawer(opsId: string) {
		selectedOpsId = opsId;
		closeDrawer();
	}

	function authHeaders(json = false): Record<string, string> {
		const h: Record<string, string> = {};
		if (json) {
			h['Content-Type'] = 'application/json';
			if (payload?.active_repo) h['X-ATeam-Repo'] = payload.active_repo;
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

	async function loadDashboard(): Promise<boolean> {
		loading = true;
		errorMsg = '';
		const result = await dashboardLoader.load(authHeaders());
		if (result.status === 'stale') return false;
		loading = false;
		if (result.status === 'failure') {
			payload = null;
			errorMsg = result.error;
			return false;
		}
		payload = result.payload;
		return true;
	}

	function openModal(node: HTMLDialogElement) {
		node.showModal();
		return { destroy() { node.close(); } };
	}

	async function changeRepository(action: Record<string, string>, notice: string) {
		if (loading) return;
		const headers = authHeaders(true);
		dashboardLoader.invalidate();
		payload = null;
		loading = true;
		errorMsg = '';
		statusNotice = '';
		closeDrawer();
		showNewCardModal = false;
		try {
			const res = await fetch('/api/repos', { method: 'POST', headers, body: JSON.stringify(action) });
			const data = await res.json();
			if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
			if (await loadDashboard()) flashNotice(notice);
		} catch (err) {
			loading = false;
			errorMsg = err instanceof Error ? err.message : 'Failed to change repository';
		}
	}

	async function runBenchmarks() {
		benchRunning = true;
		errorMsg = '';
		try {
			const res = await fetch('/api/benchmarks', { headers: authHeaders(false) });
			if (!res.ok) {
				throw new Error(`HTTP ${res.status}`);
			}
			const report: SuiteReport = await res.json();
			benchReport = report;
			flashNotice(
				`Benchmark self-test complete (${report.passed_tasks}/${report.evaluated_tasks} evaluated passed; ${report.skipped_tasks} skipped)`
			);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : 'Failed to run benchmark self-test';
		} finally {
			benchRunning = false;
		}
	}

	async function switchRepo(targetPath: string) {
		await changeRepository({ active: targetPath }, `Switched active repository to ${targetPath.split('/').pop() || targetPath}`);
	}

	async function addRepo(e: SubmitEvent) {
		e.preventDefault();
		if (!newRepoPath.trim()) return;
		await changeRepository({ add_path: newRepoPath.trim() }, 'Added and activated repository');
		if (payload) newRepoPath = '';
	}

	async function removeRepo(targetPath: string) {
		await changeRepository({ remove_path: targetPath }, 'Removed repository from switcher');
	}

	async function saveForgeConfig(e: SubmitEvent) {
		e.preventDefault();
		if (!cfgForgeOwner.trim() || !cfgForgeRepo.trim()) return;
		const selectedRepo = payload?.active_repo;
		if (!selectedRepo || loading) return;
		try {
			const res = await fetch('/api/forges', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({
					action: 'upsert',
					provider: cfgForgeProvider,
					base_url: cfgForgeBaseURL.trim(),
					owner: cfgForgeOwner.trim(),
					repo: cfgForgeRepo.trim()
				})
			});
			if (payload?.active_repo !== selectedRepo) return;
			const data = await res.json();
			if (payload?.active_repo !== selectedRepo) return;
			if (!res.ok) {
				throw new Error(data.error || `HTTP ${res.status}`);
			}
			cfgForgeBaseURL = '';
			cfgForgeOwner = '';
			cfgForgeRepo = '';
			if (!(await loadDashboard())) return;
			flashNotice(`Configured ${cfgForgeProvider} forge connection`);
		} catch (err) {
			if (payload?.active_repo !== selectedRepo) return;
			errorMsg = err instanceof Error ? err.message : 'Failed to configure forge';
		}
	}
	async function removeForgeConfig(provider: string) {
		const selectedRepo = payload?.active_repo;
		if (!selectedRepo || loading) return;
		try {
			const res = await fetch('/api/forges', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({ action: 'remove', provider })
			});
			if (payload?.active_repo !== selectedRepo) return;
			const data = await res.json();
			if (payload?.active_repo !== selectedRepo) return;
			if (!res.ok) {
				throw new Error(data.error || `HTTP ${res.status}`);
			}
			if (!(await loadDashboard())) return;
			flashNotice(`Removed ${provider} forge connection`);
		} catch (err) {
			if (payload?.active_repo !== selectedRepo) return;
			errorMsg = err instanceof Error ? err.message : 'Failed to remove forge';
		}
	}
	async function updateMode(mode: string) {
		const selectedRepo = payload?.active_repo;
		if (!selectedRepo || loading) return;
		try {
			const res = await fetch('/api/mode', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({ mode })
			});
			if (payload?.active_repo !== selectedRepo) return;
			if (!res.ok) {
				throw new Error(`HTTP ${res.status}`);
			}
			if (payload) {
				payload.state.mode = mode;
				flashNotice(`Agent discipline mode switched to ${mode.toUpperCase()}`);
			}
		} catch (err) {
			if (payload?.active_repo !== selectedRepo) return;
			errorMsg =
				err instanceof Error
					? `Failed to switch discipline mode (${err.message})`
					: 'Failed to switch discipline mode';
		}
	}
	async function moveCard(id: string, column: string, agent?: string) {
		const selectedRepo = payload?.active_repo;
		if (!selectedRepo || loading) return;
		try {
			const res = await fetch('/api/cards/move', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({ id, column, agent })
			});
			if (payload?.active_repo !== selectedRepo) return;
			if (!res.ok) {
				throw new Error(`Failed to move card (HTTP ${res.status})`);
			}
			const updated: BoardCard = await res.json();
			if (payload?.active_repo !== selectedRepo) return;
			if (payload) {
				payload.state.cards = payload.state.cards.map((c) => (c.id === updated.id ? updated : c));
			}
			flashNotice(`Moved ${id} -> ${column}`);
		} catch (err) {
			if (payload?.active_repo !== selectedRepo) return;
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
		const selectedRepo = payload?.active_repo;
		if (!selectedRepo || loading) return;
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
			if (payload?.active_repo !== selectedRepo) return;
			if (!res.ok) {
				throw new Error(`Failed to create card (HTTP ${res.status})`);
			}
			const created: BoardCard = await res.json();
			if (payload?.active_repo !== selectedRepo) return;
			if (payload) {
				payload.state.cards = [...payload.state.cards, created];
			}
			newCardTitle = '';
			newCardLabels = '';
			newCardHold = '';
			showNewCardModal = false;
			flashNotice(`Created card ${created.id}`);
		} catch (err) {
			if (payload?.active_repo !== selectedRepo) return;
			errorMsg = err instanceof Error ? err.message : 'Failed to create card';
		}
	}
	async function syncForge(provider: string) {
		const selectedRepo = payload?.active_repo;
		if (!selectedRepo || loading) return;
		try {
			const res = await fetch('/api/sync', {
				method: 'POST',
				headers: authHeaders(true),
				body: JSON.stringify({ provider })
			});
			if (payload?.active_repo !== selectedRepo) return;
			const data = await res.json();
			if (payload?.active_repo !== selectedRepo) return;
			if (!res.ok) {
				flashNotice(`Forge sync note (${provider}): ${data.error || 'offline fallback active'}`);
				return;
			}
			if (!(await loadDashboard())) return;
			flashNotice(`Synced ${data.synced ?? 0} items from ${provider}`);
		} catch {
			if (payload?.active_repo !== selectedRepo) return;
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
					Claude Code · Codex CLI · Gemini CLI · OpenSpec · GitHub & Gitea
				</p>
			</div>
		</div>

		<div class="top-controls">
			{#if payload && repos.length > 0}
				<div class="status-chip repo-switcher">
					<label for="repo-switcher-select" class="muted">Repo:</label>
					<select
						id="repo-switcher-select"
						aria-label="Active repository"
						value={payload.active_repo}
						onchange={(e) => switchRepo((e.currentTarget as HTMLSelectElement).value)}
					>
						{#each repos as r}
							<option value={r.path}>{r.name}</option>
						{/each}
					</select>
					<button
						type="button"
						class="link-btn"
						onclick={() => (showSettingsModal = true)}
						title="Add or manage repositories"
					>
						+ Repo
					</button>
				</div>
			{/if}

			<button
				type="button"
				class="user-btn"
				onclick={() => (showSettingsModal = true)}
				aria-label="User profile and settings"
				title={userBadge.email ? `${userBadge.name} (${userBadge.email}) — Settings` : `${userBadge.name} — Settings`}
			>
				<span class="user-avatar" aria-hidden="true">{userBadge.initials}</span>
				<span class="user-name">{userBadge.name}</span>
				<span class="user-gear" aria-hidden="true">⚙</span>
			</button>
		</div>
	</header>

	{#if statusNotice && !showSettingsModal && !showNewCardModal && !drawerCard && !drawerReq}
		<div class="banner info" role="status">{statusNotice}</div>
	{/if}
	{#if errorMsg && !showSettingsModal && !showNewCardModal && !drawerCard && !drawerReq}
		<div class="banner error" role="alert">{errorMsg} <button type="button" class="btn" onclick={loadDashboard} disabled={loading}>Reload state</button></div>
	{/if}

	{#if payload}
		<ModelRoutingPanel routing={payload.model_routing} />
		<section class="strip" aria-label="Current Ops: Active worktrees, claims, and session context">
			<div class="strip-card">
				<div class="strip-header">
					<div>
						<strong>Current Ops — Who's Working on What ({opsItems.length})</strong>
						<div class="muted">Select a worktree or active agent claim to inspect its session context</div>
					</div>
					<div class="status-chip" title={payload.state.git.dirty ? 'Working tree has uncommitted changes' : 'Working tree clean'}>
						<span class="dot {payload.state.git.dirty ? 'warn' : 'ok'}"></span>
						<code>{payload.state.git.branch || 'main'}@{payload.state.git.head_sha || 'HEAD'}</code>
					</div>
				</div>
				<div class="ops-list" role="listbox" aria-label="Active worktrees and agent claims">
					{#each opsItems as item (item.id)}
						{@const isSelected = selectedOps?.id === item.id}
						<button
							type="button"
							role="option"
							aria-selected={isSelected}
							class="ops-row {isSelected ? 'selected' : ''} {item.mergeStatus === 'merged' ? 'merged-stale' : ''}"
							onclick={() => (selectedOpsId = item.id)}
						>
							<div class="ops-row-main">
								<span class="agent-tag {item.agent}">{item.agent}</span>
								<span class="badge">{item.kind === 'worktree' ? 'worktree' : 'claim'}</span>
								<span class="ops-row-title">{item.title}</span>
							</div>
							<div class="ops-row-meta">
								{#if item.mergeStatus === 'merged'}
									<span
										class="badge merged-badge"
										title={item.mergeEvidence ? `${item.mergeEvidence} — worktree already merged` : 'Worktree already merged'}
										aria-label="Merged worktree ({item.mergeEvidence || 'already merged into main'})"
									>
										✓ Merged
									</span>
								{:else if item.mergeStatus === 'closed-unmerged'}
									<span class="badge warn" title={item.mergeEvidence || 'PR closed without merging'}>
										Closed (unmerged)
									</span>
								{/if}
								{#if item.prNumber}
									<span class="badge pr-badge">{item.prNumber}</span>
								{/if}
								<code>{item.branch}{item.kind === 'worktree' ? `@${item.head}` : ''}</code>
								{#if item.dirty}
									<span class="dot warn" title="Uncommitted changes"></span>
								{/if}
							</div>
						</button>
					{/each}
				</div>
			</div>

			<div class="strip-card">
				<div class="strip-header">
					<div class="strip-title-group">
						<strong>Selected Session Context</strong>
						{#if selectedOps}
							<span class="agent-tag {selectedOps.agent}">{selectedOps.agent}</span>
							{#if selectedOps.mergeStatus === 'merged'}
								<span
									class="badge merged-badge"
									title={selectedOps.mergeEvidence || 'Branch already merged into main'}
									aria-label="Merged or potentially stale worktree"
								>
									✓ Merged / potentially stale
								</span>
							{/if}
							{#if selectedOps.drifted}
								<span class="badge warn">HEAD DRIFTED</span>
							{:else if selectedOps.kind === 'worktree' && selectedOps.mergeStatus !== 'merged'}
								<span class="badge ok">Active Worktree</span>
							{:else if selectedOps.kind !== 'worktree'}
								<span class="badge">{selectedOps.statusColumn || 'Claim'}</span>
							{/if}
						{/if}
					</div>
				</div>

				{#if selectedOps}
					<dl class="context-grid">
						<div class="context-field">
							<dt>Working On / Goal</dt>
							<dd>{selectedOps.goal}</dd>
						</div>
						<div class="context-field">
							<dt>Linked PR / Addresses</dt>
							<dd>
								{#if selectedOps.prNumber && selectedOps.prUrl}
									<a href={selectedOps.prUrl} target="_blank" rel="noreferrer" class="pr-link">
										{selectedOps.prNumber} ↗
									</a>
									<span class="muted"> · {selectedOps.addresses}</span>
								{:else if selectedOps.prNumber}
									<span class="badge pr-badge">{selectedOps.prNumber}</span>
									<span> {selectedOps.addresses}</span>
								{:else}
									{selectedOps.addresses}
								{/if}
							</dd>
						</div>
						<div class="context-field">
							<dt>Branch &amp; Writer Status</dt>
							<dd><code>{selectedOps.writerStatus}</code></dd>
						</div>
						<div class="context-field">
							<dt>Holds &amp; Constraints</dt>
							<dd>{selectedOps.constraints}</dd>
						</div>
						<div class="context-field context-field-wide">
							<dt>Done When &amp; Verification</dt>
							<dd>
								{selectedOps.doneWhen}
								<span class="muted"> · Checks: </span>
								<code>{selectedOps.checksToRun}</code>
							</dd>
						</div>
					</dl>
				{:else}
					<div class="muted">Select a worktree or claim on the left to inspect its context.</div>
				{/if}
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
					OpenSpec
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
						<option value="all">All Forges</option>
						<option value="github">GitHub</option>
						<option value="gitea">Gitea</option>
						<option value="local">Local</option>
					</select>

					{#each configuredForges as f}
						<button
							type="button"
							class="btn secondary"
							onclick={() => syncForge(f.provider)}
							title={f.owner && f.repo ? `Sync ${f.provider} (${f.owner}/${f.repo})` : `Sync ${f.provider}`}
						>
							↻ Sync {f.provider === 'github' ? 'GitHub' : 'Gitea'}
						</button>
					{/each}

					<button type="button" class="btn primary" onclick={() => (showNewCardModal = true)}>
						+ New Card
					</button>
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
									{@const rel = resolveCardRelations(card, payload.state, opsItems)}
									<button
										type="button"
										class="card {drawerCardId === card.id ? 'selected' : ''}"
										onclick={() => openCardDrawer(card.id)}
										aria-label="Inspect card {card.id}: {card.title}"
									>
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
										{#if rel.requirements.length > 0 || (card.labels && card.labels.length > 0)}
											<div class="label-row">
												{#each rel.requirements as req}
													<span class="lbl spec-pill" title="OpenSpec requirement: {req.name}">
														📜 {req.id}
													</span>
												{/each}
												{#each card.labels ?? [] as lbl}
													<span class="lbl">{lbl}</span>
												{/each}
											</div>
										{/if}
									</button>
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
								<th>Linked Spec / Branch</th>
								<th>Labels / Hold</th>
							</tr>
						</thead>
						<tbody>
							{#each filtered as card (card.id)}
								{@const rel = resolveCardRelations(card, payload.state, opsItems)}
								<tr>
									<td>
										<button type="button" class="link-btn" onclick={() => openCardDrawer(card.id)}>
											<code>{card.id}</code>
										</button>
									</td>
									<td>
										<button type="button" class="table-title-btn" onclick={() => openCardDrawer(card.id)}>
											<strong>{card.title}</strong>
										</button>
									</td>
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
										<div class="label-row">
											{#each rel.requirements as req}
												<button
													type="button"
													class="lbl spec-pill clickable-pill"
													onclick={() => openReqDrawer(req.id)}
												>
													📜 {req.id}
												</button>
											{/each}
											{#if card.branch}<code>{card.branch}</code>{/if}
										</div>
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
				<div class="openspec-stack">
					<div class="panel">
						<div class="spec-panel-header">
							<div>
								<h2>Canonical OpenSpec Specifications ({allRequirements.length} Requirements)</h2>
								<p class="muted">
									Parsed live from <code>openspec/specs/*/spec.md</code>. Click any requirement or linked card to inspect scenarios, acceptance criteria, and active worktrees.
								</p>
							</div>
						</div>

						{#if openspecSpecs.length === 0}
							<div class="empty-box">No canonical specs found under <code>openspec/specs/</code>.</div>
						{:else}
							{#each openspecSpecs as sp}
								<div class="canon-spec-block">
									<div class="canon-spec-header">
										<div>
											<span class="badge ok">{sp.name}</span>
											<code>{sp.path}</code>
										</div>
										<span class="muted">{sp.requirements?.length ?? 0} requirements</span>
									</div>
									{#if sp.purpose}
										<p class="canon-spec-purpose">{sp.purpose}</p>
									{/if}
									<div class="req-grid">
										{#each sp.requirements ?? [] as req (req.id)}
											{@const linkedCards = resolveRequirementCards(req, cards)}
											<div class="req-card {drawerReqId === req.id ? 'selected' : ''}">
												<button
													type="button"
													class="req-card-main"
													onclick={() => openReqDrawer(req.id)}
												>
													<div class="req-card-top">
														<span class="lbl spec-pill">📜 {req.id}</span>
														<span class="muted">{req.scenarios?.length ?? 0} scenario(s)</span>
													</div>
													<strong class="req-card-title">{req.name}</strong>
													{#if req.statement}
														<p class="req-card-stmt">{req.statement}</p>
													{/if}
												</button>
												<div class="req-card-links">
													<span class="muted">Linked Cards:</span>
													{#if linkedCards.length === 0}
														<span class="muted">None</span>
													{:else}
														{#each linkedCards as lc}
															<button
																type="button"
																class="link-btn card-pill"
																onclick={() => openCardDrawer(lc.id)}
																title={lc.title}
															>
																{lc.id} · {lc.column}
															</button>
														{/each}
													{/if}
												</div>
											</div>
										{/each}
									</div>
								</div>
							{/each}
						{/if}
					</div>

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
							<h2>`// shortcut:` Tech Debt Inventory ({shortcuts.length})</h2>
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
				</div>
			{/if}
		</main>
	{:else if loading}
		<div class="loading">Loading ATeam harness state...</div>
	{/if}

	{#if showNewCardModal}
		<dialog class="modal-backdrop" use:openModal onclose={() => (showNewCardModal = false)} aria-label="New board card">
			<form class="modal" onsubmit={createCard}>
				{#if errorMsg}<div class="banner error" role="alert">{errorMsg} <button type="button" class="btn" onclick={loadDashboard} disabled={loading}>Reload state</button></div>{/if}
				{#if statusNotice}<div class="banner info" role="status">{statusNotice}</div>{/if}
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
		</dialog>
	{/if}

	{#if showSettingsModal}
		<dialog class="modal-backdrop" use:openModal onclose={() => (showSettingsModal = false)} aria-label="User profile and settings">
			<div class="modal modal-wide">
				{#if errorMsg}<div class="banner error" role="alert">{errorMsg} <button type="button" class="btn" onclick={loadDashboard} disabled={loading}>Reload state</button></div>{/if}
				{#if statusNotice}<div class="banner info" role="status">{statusNotice}</div>{/if}
				<h2>User Profile &amp; Settings</h2>

				<div class="user-card">
					<span class="user-avatar large" aria-hidden="true">{userBadge.initials}</span>
					<div class="user-card-info">
						<strong>{userBadge.name}</strong>
						{#if userBadge.email}
							<span class="muted">{userBadge.email}</span>
						{:else}
							<span class="muted">Auto-detected from local git / OS user</span>
						{/if}
					</div>
				</div>

				{#if payload}
					<div class="settings-section">
						<span class="settings-label">Repositories ({repos.length})</span>
						<div class="settings-list">
							{#each repos as r, idx}
								<div class="settings-item">
									<div>
										<strong>{r.name}</strong>
										<code class="muted">{r.path}</code>
									</div>
									<div class="settings-item-actions">
										{#if r.active}
											<span class="badge ok">Active</span>
										{:else}
											<button type="button" class="link-btn" onclick={() => switchRepo(r.path)}>Switch</button>
										{/if}
										{#if idx > 0}
											<button type="button" class="link-btn" onclick={() => removeRepo(r.path)}>Remove</button>
										{/if}
									</div>
								</div>
							{/each}
						</div>
						<form class="inline-form" onsubmit={addRepo}>
							<input
								type="text"
								bind:value={newRepoPath}
								placeholder="/path/to/another/repo"
								aria-label="Local repository path"
							/>
							<button type="submit" class="btn secondary compact">+ Add Repo</button>
						</form>
					</div>

					<div class="settings-section">
						<span class="settings-label">Connected Forges ({configuredForges.length})</span>
						{#if configuredForges.length === 0}
							<span class="settings-hint">No remote forge configured (local board mode).</span>
						{:else}
							<div class="settings-list">
								{#each configuredForges as f}
									<div class="settings-item">
										<div>
											<span class="badge forge-{f.provider}">{f.provider}</span>
											<code>{f.owner || '?'}/{f.repo || '?'}</code>
											{#if f.base_url && f.provider === 'gitea'}
												<span class="muted">({f.base_url})</span>
											{/if}
										</div>
										<div class="settings-item-actions">
											<button type="button" class="link-btn" onclick={() => syncForge(f.provider)}>Sync</button>
											<button type="button" class="link-btn" onclick={() => removeForgeConfig(f.provider)}>Remove</button>
										</div>
									</div>
								{/each}
							</div>
						{/if}
						<form class="forge-config-form" onsubmit={saveForgeConfig}>
							<div class="form-row">
								<select bind:value={cfgForgeProvider} aria-label="Forge provider">
									<option value="gitea">Gitea</option>
									<option value="github">GitHub</option>
								</select>
								{#if cfgForgeProvider === 'gitea'}
									<input
										type="url"
										required
										bind:value={cfgForgeBaseURL}
										placeholder="https://gitea.example.com"
										aria-label="Gitea base URL"
									/>
								{:else}
									<input
										type="url"
										bind:value={cfgForgeBaseURL}
										placeholder="https://api.github.com (default)"
										aria-label="GitHub API URL"
									/>
								{/if}
							</div>
							<div class="inline-form">
								<input
									type="text"
									required
									bind:value={cfgForgeOwner}
									placeholder="Owner / Org"
									aria-label="Forge owner"
								/>
								<input
									type="text"
									required
									bind:value={cfgForgeRepo}
									placeholder="Repo name"
									aria-label="Forge repository"
								/>
								<button type="submit" class="btn secondary compact">Save Forge</button>
							</div>
						</form>
					</div>

					<div class="settings-section">
						<span class="settings-label">Agent Discipline Mode</span>
						<div class="pill-group" role="group" aria-label="Agent discipline mode">
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
						<span class="settings-hint">
							Controls the decision-ladder and verification rigor injected by <code>ateam hook</code> (`lite`, `full`, `ultra`, or `off`).
						</span>
					</div>

					<div class="settings-section">
						<div class="settings-row-between">
							<span class="settings-label">Harness Diagnostics &amp; Benchmarks (`ateam bench --selftest`)</span>
							<button
								type="button"
								class="btn secondary compact"
								disabled={benchRunning}
								onclick={runBenchmarks}
							>
								{benchRunning ? 'Running...' : 'Run Self-Test'}
							</button>
						</div>
						<span class="settings-hint">
							Offline verification of AST safety gates, mutation catch rate, MCP token budget, and hook latency. Triggered on-demand here or during <code>task verify</code> (never runs continuously in the background).
						</span>
						{#if benchReport}
							<div class="metrics-mini">
								<div class="metric-box">
									<span class="metric-val">{benchReport.safety_pass_rate_pct}% / {benchReport.mutation_catch_pct}%</span>
									<span class="metric-lbl">Safety &amp; Mutant Catch</span>
								</div>
								<div class="metric-box">
									<span class="metric-val">{benchReport.mcp_token_tax} tok</span>
									<span class="metric-lbl">MCP Schema (&lt;300)</span>
								</div>
								<div class="metric-box">
									<span class="metric-val">{benchReport.hook_latency_us} µs</span>
									<span class="metric-lbl">Hook Latency (&lt;5ms)</span>
								</div>
								<div class="metric-box">
									<span class="metric-val">{benchReport.passed_tasks}/{benchReport.evaluated_tasks} PASS · {benchReport.skipped_tasks} skipped</span>
									<span class="metric-lbl">Reference Fixtures</span>
								</div>
							</div>
						{/if}
					</div>
				{/if}

				<label>
					API Token (ATEAM_API_TOKEN)
					<input
						id="api-token-input"
						type="password"
						placeholder="Optional token for authenticated API calls"
						value={apiToken}
						onchange={(e) => saveApiToken((e.currentTarget as HTMLInputElement).value)}
						aria-label="API token"
					/>
					<span class="settings-hint">Stored in browser localStorage and sent via <code>X-ATeam-Token</code>.</span>
				</label>

				<div class="modal-actions">
					<button type="button" class="btn primary" onclick={() => (showSettingsModal = false)}>Done</button>
				</div>
			</div>
		</dialog>
	{/if}

	{#if drawerCard || drawerReq}
		<dialog
			class="drawer-backdrop"
			use:openModal
			onclose={closeDrawer}
			aria-label={drawerCard ? `Card details for ${drawerCard.id}` : `OpenSpec requirement ${drawerReq?.name ?? ''}`}
		>
			<button
				type="button"
				class="drawer-dismiss"
				aria-label="Close detail drawer"
				onclick={closeDrawer}
			></button>

			<aside class="drawer-panel">
				{#if errorMsg}<div class="banner error" role="alert">{errorMsg}</div>{/if}
				{#if statusNotice}<div class="banner info" role="status">{statusNotice}</div>{/if}
				{#if drawerCard && drawerCardRel}
					<header class="drawer-header">
						<div class="drawer-title-meta">
							<span class="card-id">{drawerCard.id}</span>
							<span class="badge forge-{drawerCard.forge || 'local'}">{drawerCard.forge || 'local'}</span>
							<span class="badge type-{drawerCard.type}">{drawerCard.type}</span>
							{#if drawerCard.agent}
								<span class="agent-tag {drawerCard.agent}">{drawerCard.agent}</span>
							{/if}
						</div>
						<button type="button" class="btn secondary compact" onclick={closeDrawer}>
							✕ Close
						</button>
					</header>

					<h2 class="drawer-title">{drawerCard.title}</h2>

					{#if drawerCard.url}
						<a href={drawerCard.url} target="_blank" rel="noreferrer" class="pr-link">
							Open on {drawerCard.forge || 'remote forge'} ({drawerCard.id}) ↗
						</a>
					{/if}

					{#if drawerCard.hold_reason}
						<div class="hold-alert">{drawerCard.hold_reason}</div>
					{/if}

					<div class="drawer-section">
						<span class="settings-label">Workflow Status &amp; Agent Assignment</span>
						<div class="form-row">
							<label class="drawer-field-label">
								Status Column
								<select
									aria-label="Move {drawerCard.id} to column"
									value={drawerCard.column}
									onchange={(e) =>
										moveCard(drawerCard.id, (e.currentTarget as HTMLSelectElement).value, drawerCard.agent)}
								>
									{#each BOARD_COLUMNS as dest}
										<option value={dest.id}>{dest.label}</option>
									{/each}
								</select>
							</label>
							<label class="drawer-field-label">
								Assigned Agent
								<select
									aria-label="Assign agent for {drawerCard.id}"
									value={drawerCard.agent || ''}
									onchange={(e) =>
										moveCard(drawerCard.id, drawerCard.column, (e.currentTarget as HTMLSelectElement).value)}
								>
									<option value="">Unassigned</option>
									<option value="claude">claude</option>
									<option value="codex">codex</option>
									<option value="gemini">gemini</option>
								</select>
							</label>
						</div>
					</div>

					<div class="drawer-section">
						<span class="settings-label">Branch, Milestone &amp; Labels</span>
						<div class="drawer-kv-grid">
							<div class="context-field">
								<dt>Branch</dt>
								<dd><code>{drawerCard.branch || '—'}</code></dd>
							</div>
							<div class="context-field">
								<dt>Milestone</dt>
								<dd>{drawerCard.milestone || '—'}</dd>
							</div>
						</div>
						{#if drawerCard.labels && drawerCard.labels.length > 0}
							<div class="label-row">
								{#each drawerCard.labels as lbl}
									<span class="lbl">{lbl}</span>
								{/each}
							</div>
						{/if}
					</div>

					<div class="drawer-section">
						<span class="settings-label">
							Linked OpenSpec Requirements ({drawerCardRel.requirements.length})
						</span>
						{#if drawerCardRel.requirements.length === 0}
							<div class="empty-box">
								No OpenSpec requirement directly linked to {drawerCard.id}.
							</div>
						{:else}
							<div class="drawer-list">
								{#each drawerCardRel.requirements as req}
									<div class="drawer-linked-card">
										<div class="drawer-linked-head">
											<button
												type="button"
												class="lbl spec-pill clickable-pill"
												onclick={() => openReqDrawer(req.id)}
											>
												📜 {req.id} ↗
											</button>
											<code class="muted">openspec/specs/{req.spec}/spec.md</code>
										</div>
										<strong>{req.name}</strong>
										{#if req.statement}
											<p class="drawer-stmt">{req.statement}</p>
										{/if}
										{#if req.scenarios && req.scenarios.length > 0}
											<div class="scenario-list">
												{#each req.scenarios as sc}
													<div class="scenario-box">
														<strong>Scenario: {sc.name}</strong>
														{#each sc.steps ?? [] as step}
															<div class="scenario-line">{step}</div>
														{/each}
													</div>
												{/each}
											</div>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<div class="drawer-section">
						<span class="settings-label">
							Linked Worktree / Active Session ({drawerCardRel.opsItems.length})
						</span>
						{#if drawerCardRel.opsItems.length === 0}
							<div class="empty-box">
								No active git worktree or session claim currently matches {drawerCard.id}.
							</div>
						{:else}
							<div class="drawer-list">
								{#each drawerCardRel.opsItems as op}
									<div class="drawer-linked-card">
										<div class="drawer-linked-head">
											<div>
												<span class="agent-tag {op.agent}">{op.agent}</span>
												<span class="badge">{op.kind}</span>
												{#if op.prNumber}
													<span class="badge pr-badge">{op.prNumber}</span>
												{/if}
											</div>
											<button
												type="button"
												class="link-btn"
												onclick={() => selectOpsFromDrawer(op.id)}
											>
												Focus in Current Ops ↑
											</button>
										</div>
										<strong>{op.title}</strong>
										<code class="muted">{op.writerStatus}</code>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{:else if drawerReq}
					<header class="drawer-header">
						<div class="drawer-title-meta">
							<span class="lbl spec-pill">📜 {drawerReq.id}</span>
							<code class="muted">openspec/specs/{drawerReq.spec}/spec.md</code>
						</div>
						<button type="button" class="btn secondary compact" onclick={closeDrawer}>
							✕ Close
						</button>
					</header>

					<h2 class="drawer-title">Requirement: {drawerReq.name}</h2>

					{#if drawerReq.statement}
						<div class="drawer-section">
							<span class="settings-label">Specification Statement</span>
							<p class="drawer-stmt">{drawerReq.statement}</p>
						</div>
					{/if}

					<div class="drawer-section">
						<span class="settings-label">
							Acceptance Scenarios ({drawerReq.scenarios?.length ?? 0})
						</span>
						{#if !drawerReq.scenarios || drawerReq.scenarios.length === 0}
							<div class="empty-box">No scenarios defined for this requirement.</div>
						{:else}
							<div class="scenario-list">
								{#each drawerReq.scenarios as sc}
									<div class="scenario-box">
										<strong>Scenario: {sc.name}</strong>
										{#each sc.steps ?? [] as step}
											<div class="scenario-line">{step}</div>
										{/each}
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<div class="drawer-section">
						<span class="settings-label">
							Linked Board Cards ({drawerReqCards.length})
						</span>
						{#if drawerReqCards.length === 0}
							<div class="empty-box">No board cards currently linked to this requirement.</div>
						{:else}
							<div class="drawer-list">
								{#each drawerReqCards as lc}
									<button
										type="button"
										class="drawer-linked-card clickable-card"
										onclick={() => openCardDrawer(lc.id)}
									>
										<div class="drawer-linked-head">
											<div>
												<span class="card-id">{lc.id}</span>
												<span class="badge">{lc.column}</span>
												{#if lc.agent}
													<span class="agent-tag {lc.agent}">{lc.agent}</span>
												{/if}
											</div>
											<span class="muted">Inspect Card →</span>
										</div>
										<strong>{lc.title}</strong>
										{#if lc.branch}
											<code class="muted">{lc.branch}</code>
										{/if}
									</button>
								{/each}
							</div>
						{/if}
					</div>
				{/if}
			</aside>
		</dialog>
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
		padding: 0.25rem;
		border-radius: 0.45rem;
		gap: 0.25rem;
		width: fit-content;
	}
	.mode-btn {
		background: transparent;
		border: none;
		color: #cbd5e1;
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.32rem 0.65rem;
		border-radius: 0.35rem;
		cursor: pointer;
		text-transform: uppercase;
	}
	.mode-btn.active {
		background: #3b82f6;
		color: #fff;
	}
	.user-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		background: #1e293b;
		border: 1px solid #334155;
		color: #f8fafc;
		padding: 0.28rem 0.65rem 0.28rem 0.35rem;
		border-radius: 999px;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
	}
	.user-btn:hover {
		border-color: #475569;
		background: #263349;
	}
	.user-avatar {
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: linear-gradient(135deg, #3b82f6, #8b5cf6);
		color: #fff;
		font-size: 0.68rem;
		font-weight: 800;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		letter-spacing: 0.02em;
	}
	.user-avatar.large {
		width: 38px;
		height: 38px;
		font-size: 0.9rem;
	}
	.user-name {
		max-width: 140px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.user-gear {
		color: #94a3b8;
		font-size: 0.82rem;
	}
	.user-card {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.5rem;
		padding: 0.75rem;
	}
	.user-card-info {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}
	.settings-section {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		padding-top: 0.25rem;
		border-top: 1px solid #1e293b;
	}
	.settings-row-between {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.5rem;
	}
	.settings-label {
		font-size: 0.78rem;
		font-weight: 600;
		color: #cbd5e1;
	}
	.settings-hint {
		font-size: 0.72rem;
		color: #94a3b8;
	}
	.settings-list {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}
	.settings-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.5rem;
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.4rem;
		padding: 0.4rem 0.6rem;
		font-size: 0.78rem;
	}
	.settings-item-actions {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}
	.inline-form {
		display: flex;
		gap: 0.45rem;
		align-items: center;
	}
	.inline-form input {
		flex: 1;
	}
	.forge-config-form {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}
	.repo-switcher select {
		padding: 0.2rem 0.45rem;
		font-weight: 600;
	}
	.strip-title-group {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
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
	.btn.compact {
		padding: 0.3rem 0.65rem;
		font-size: 0.74rem;
		white-space: nowrap;
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
		grid-template-columns: 1fr 1.25fr;
		gap: 0.9rem;
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
		margin-bottom: 0.55rem;
		gap: 0.6rem;
	}
	.ops-list {
		display: flex;
		flex-direction: column;
		gap: 0.38rem;
		max-height: 8.5rem;
		overflow-y: auto;
	}
	.ops-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.6rem;
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.42rem;
		padding: 0.45rem 0.65rem;
		color: #e2e8f0;
		font-size: 0.78rem;
		text-align: left;
		cursor: pointer;
	}
	.ops-row:hover {
		border-color: #334155;
		background: #131c2e;
	}
	.ops-row.selected {
		border-color: #3b82f6;
		background: rgba(59, 130, 246, 0.12);
	}
	.ops-row.merged-stale {
		opacity: 0.85;
	}
	.badge.merged-badge {
		background: rgba(16, 185, 129, 0.16);
		border: 1px solid rgba(16, 185, 129, 0.45);
		color: #6ee7b7;
	}
	.ops-row-main {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		min-width: 0;
		flex: 1;
	}
	.ops-row-title {
		font-weight: 600;
		color: #f8fafc;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.ops-row-meta {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		flex-shrink: 0;
		font-size: 0.74rem;
		color: #94a3b8;
	}
	.badge.pr-badge {
		background: rgba(139, 92, 246, 0.25);
		color: #c4b5fd;
	}
	.pr-link {
		color: #93c5fd;
		font-weight: 700;
		text-decoration: none;
	}
	.pr-link:hover {
		text-decoration: underline;
	}
	.context-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.45rem 0.85rem;
		margin: 0;
	}
	.context-field {
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.4rem;
		padding: 0.4rem 0.6rem;
	}
	.context-field-wide {
		grid-column: 1 / -1;
	}
	.context-field dt {
		font-size: 0.68rem;
		font-weight: 700;
		text-transform: uppercase;
		color: #94a3b8;
		margin-bottom: 0.15rem;
	}
	.context-field dd {
		margin: 0;
		font-size: 0.78rem;
		color: #f1f5f9;
		line-height: 1.35;
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
		min-width: 0;
		overflow: hidden;
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
		min-width: 0;
	}
	.card {
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.48rem;
		padding: 0.7rem;
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		min-width: 0;
		width: 100%;
		box-sizing: border-box;
		overflow: hidden;
		text-align: left;
		color: inherit;
		font: inherit;
		cursor: pointer;
		transition: border-color 0.12s ease, background 0.12s ease;
	}
	.card:hover {
		border-color: #3b82f6;
		background: #131c2e;
	}
	.card.selected {
		border-color: #3b82f6;
		background: rgba(59, 130, 246, 0.14);
	}
	.card-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.3rem;
		min-width: 0;
		width: 100%;
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
		overflow-wrap: anywhere;
	}
	.card-branch {
		font-size: 0.72rem;
		color: #38bdf8;
		overflow-wrap: anywhere;
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
		align-items: center;
	}
	.lbl {
		background: #1e293b;
		color: #cbd5e1;
		font-size: 0.68rem;
		padding: 0.12rem 0.4rem;
		border-radius: 0.25rem;
	}
	.lbl.spec-pill {
		background: rgba(59, 130, 246, 0.2);
		border: 1px solid rgba(59, 130, 246, 0.4);
		color: #93c5fd;
		font-weight: 600;
	}
	.clickable-pill {
		cursor: pointer;
		font-family: inherit;
	}
	.clickable-pill:hover {
		background: rgba(59, 130, 246, 0.32);
	}
	.table-title-btn {
		background: transparent;
		border: none;
		color: #f8fafc;
		padding: 0;
		font: inherit;
		text-align: left;
		cursor: pointer;
	}
	.table-title-btn:hover {
		color: #60a5fa;
		text-decoration: underline;
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
	.openspec-stack {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.spec-panel-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1rem;
	}
	.canon-spec-block {
		margin-top: 0.75rem;
		padding-top: 0.75rem;
		border-top: 1px solid #1e293b;
	}
	.canon-spec-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.35rem;
	}
	.canon-spec-purpose {
		margin: 0 0 0.75rem;
		font-size: 0.8rem;
		color: #cbd5e1;
		line-height: 1.4;
	}
	.req-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
		gap: 0.7rem;
	}
	.req-card {
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.48rem;
		padding: 0.7rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: 0.55rem;
	}
	.req-card.selected {
		border-color: #3b82f6;
		background: rgba(59, 130, 246, 0.12);
	}
	.req-card-main {
		background: transparent;
		border: none;
		color: inherit;
		font: inherit;
		padding: 0;
		text-align: left;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		cursor: pointer;
	}
	.req-card-main:hover .req-card-title {
		color: #60a5fa;
	}
	.req-card-top {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.4rem;
	}
	.req-card-title {
		font-size: 0.84rem;
		color: #f8fafc;
	}
	.req-card-stmt {
		margin: 0;
		font-size: 0.76rem;
		color: #cbd5e1;
		line-height: 1.35;
	}
	.req-card-links {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.3rem;
		padding-top: 0.4rem;
		border-top: 1px solid #1e293b;
	}
	.card-pill {
		font-family: monospace;
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
	dialog.modal-backdrop, dialog.drawer-backdrop {
		margin: 0;
		padding: 0;
		border: 0;
		color: inherit;
		width: 100%;
		height: 100dvh;
		max-width: none;
		max-height: none;
		box-sizing: border-box;
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
	.modal.modal-wide {
		max-width: 540px;
		max-height: 88vh;
		overflow-y: auto;
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
	.drawer-backdrop {
		position: fixed;
		inset: 0;
		z-index: 60;
		display: flex;
		justify-content: flex-end;
		background: rgba(0, 0, 0, 0.65);
	}
	.drawer-dismiss {
		flex: 1;
		background: transparent;
		border: none;
		cursor: pointer;
	}
	.drawer-panel {
		width: 100%;
		max-width: 520px;
		height: 100vh;
		box-sizing: border-box;
		background: #111827;
		border-left: 1px solid #334155;
		padding: 1.2rem 1.35rem;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 0.85rem;
		box-shadow: -12px 0 32px rgba(0, 0, 0, 0.5);
	}
	.drawer-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.5rem;
	}
	.drawer-title-meta {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		flex-wrap: wrap;
	}
	.drawer-title {
		margin: 0;
		font-size: 1.05rem;
		line-height: 1.35;
		color: #f8fafc;
	}
	.drawer-section {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		padding-top: 0.65rem;
		border-top: 1px solid #1e293b;
	}
	.drawer-field-label {
		display: flex;
		flex-direction: column;
		gap: 0.28rem;
		font-size: 0.75rem;
		color: #cbd5e1;
	}
	.drawer-kv-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}
	.drawer-list {
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
	}
	.drawer-linked-card {
		background: #0b0f19;
		border: 1px solid #1e293b;
		border-radius: 0.45rem;
		padding: 0.65rem 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.38rem;
		text-align: left;
		color: inherit;
		font: inherit;
	}
	.drawer-linked-card.clickable-card {
		cursor: pointer;
		width: 100%;
		box-sizing: border-box;
	}
	.drawer-linked-card.clickable-card:hover {
		border-color: #3b82f6;
		background: #131c2e;
	}
	.drawer-linked-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 0.4rem;
	}
	.drawer-stmt {
		margin: 0;
		font-size: 0.78rem;
		color: #cbd5e1;
		line-height: 1.4;
	}
	.scenario-list {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		margin-top: 0.2rem;
	}
	.scenario-box {
		background: #111827;
		border: 1px solid #1e293b;
		border-radius: 0.38rem;
		padding: 0.45rem 0.6rem;
		font-size: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}
	.scenario-line {
		color: #cbd5e1;
		line-height: 1.35;
	}
</style>
