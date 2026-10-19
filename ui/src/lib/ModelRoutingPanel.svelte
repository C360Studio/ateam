<script lang="ts">
 import type { ModelRouting } from './board';

 let { routing }: { routing?: ModelRouting } = $props();
 const status = $derived(routing?.status ?? 'unavailable');
 const labels = {
  eligible: 'Eligible route',
  blocked: 'Routing blocked',
  unconfigured: 'Not configured',
  invalid: 'Invalid policy',
  unavailable: 'Routing unavailable'
 };
</script>

<section class="routing-panel" aria-labelledby="model-routing-title">
 <div class="heading">
  <div>
   <h2 id="model-routing-title">Model routing</h2>
   <p>Configured roles for this checkout</p>
  </div>
  <strong class="status {status}" role="status">{labels[status]}</strong>
 </div>
 {#if routing?.reason}
  <p class="reason">{routing.reason}</p>
 {:else if status === 'unavailable'}
  <p class="reason">Model routing data is unavailable. Refresh after connecting to a server that supports routing.</p>
 {/if}
 {#if status === 'eligible' || status === 'blocked'}
  <div class="roles">
   {#each [{ label: 'Developer', route: routing?.developer }, { label: 'Reviewer', route: routing?.reviewer }] as role}
    <div class="role">
     <h3>{role.label}</h3>
     <dl>
      <div><dt>Model</dt><dd><code>{role.route?.model ?? 'Unavailable'}</code></dd></div>
      <div><dt>Provider</dt><dd>{role.route?.provider ?? 'Unavailable'}</dd></div>
      <div><dt>Reasoning effort</dt><dd>{role.route?.effort ?? 'Unavailable'}</dd></div>
     </dl>
    </div>
   {/each}
  </div>
 {/if}
 {#if status === 'unconfigured'}
  <p>Configure an owner-approved policy with <code>ateam review setup --policy FILE</code>, then refresh the dashboard.</p>
 {/if}
 <p class="note">Configured routes do not identify running models or approve a PR. Reasoning effort is separate from model strength and Ponytail mode.</p>
</section>

<style>
 .routing-panel { margin: 0 1.4rem 1rem; padding: 1rem; background: #111827; border: 1px solid #334155; border-radius: 12px; color: #e2e8f0; }
 .heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: .75rem; }
 h2 { font-size: 1rem; margin: 0; }
 h3 { font-size: .9rem; margin: 0 0 .7rem; }
 p { font-size: .85rem; line-height: 1.5; margin: .4rem 0; }
 .heading p, .note, dt { color: #cbd5e1; }
 .status { font-size: .8rem; border: 1px solid #64748b; border-radius: 999px; padding: .3rem .7rem; }
 .eligible { color: #a7f3d0; border-color: #059669; }
 .blocked, .invalid { color: #fde68a; border-color: #d97706; }
 .reason { margin-top: .75rem; overflow-wrap: anywhere; }
 .roles { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .75rem; margin: .8rem 0; }
 .role { padding: .8rem; background: #0f172a; border: 1px solid #334155; border-radius: 8px; min-width: 0; }
 dl { margin: 0; font-size: .85rem; }
 dl div { display: grid; grid-template-columns: 8rem minmax(0, 1fr); gap: .5rem; margin-top: .35rem; }
 dd { margin: 0; overflow-wrap: anywhere; }
 code { font-size: .85rem; overflow-wrap: anywhere; }
 .note { margin-top: .75rem; }
 @media (max-width: 640px) { .roles { grid-template-columns: 1fr; } dl div { grid-template-columns: 7rem minmax(0, 1fr); } }
</style>
