# Change Proposal: `precompact-foreign-124`

## Why

The handoff slot is keyed `agent@branch`, so two sessions of the same agent on one branch share it. When one session compacts, the pre-compact hook treats a fresh handoff written by the other session like a missing one and overwrites it with `UNKNOWN — Auto-saved before context compaction` (#124, dogfood pass 7). The auto-save exists so a compacting session leaves something behind; it should not destroy a better handoff another session wrote on purpose.

## What Changes

- Pre-compact leaves a non-stale handoff from a different or unknown session unchanged and says in its output which session holds the slot.
- A nil or stale handoff, or another session's fallback auto-save, still gets replaced by the fallback auto-save; a same-session handoff is still preserved and marked `auto_compacted`.

## Non-goals

- Keying the slot by session (`agent@branch@session`). Two sessions writing intentional handoffs to one slot stay last-writer-wins; that is a separate issue if it shows up in practice.
