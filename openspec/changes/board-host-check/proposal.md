# Proposal: board-host-check
Approved-by:

## Track

`quick`: it modifies two existing harness Requirements and adds none, touches one capability, adds no exported surface, and is no boot path, durability or gate file. The code is `internal/web/server.go` and one line in `internal/state/forgefiles.go`.

## Decisions you are making by approving

- Without `ATEAM_API_TOKEN`, the board API answers only when the browser addressed it as `localhost`, `127.0.0.1` or `[::1]`, with or without a port, and any `Origin` it sends is one of those too. Reaching the board through another name, such as a tunnel, a LAN name or a hosts-file alias, now needs `ATEAM_API_TOKEN` set and sent. A token keeps working from any host.
- Approve refuses a branch name that is not a valid git branch name with a 400, before it calls the forge. A real branch such as `claude/topic` is unaffected.

## Why

With no token, the board trusts any request whose peer address is loopback. A web page on another site whose DNS name is re-pointed at `127.0.0.1` makes the visitor's browser send such a request, and the page controls both `Host` and `Origin`. The existing same-origin comparison then passes, so the page can read board state and call Approve, which writes an approval commit. The server never looked at the `Host` header. Approve also put the branch name into forge URLs after only trimming it.

## What Changes

1. `authorizeAPIRequest` in its tokenless branch, after the loopback peer check, answers 403 unless `Host` and any `Origin` name a loopback host. This covers every board read and write, including model routing. The token branch, the webhooks and the static UI are unchanged.
2. `handleOpenSpecApprove` answers 400 for a branch that git would refuse as a ref name, one that climbs a path (`..`), starts with `-`, or holds a character that ends a URL path.
3. The Gitea branch lookup escapes the branch per path segment, as the GitHub one already does.
4. The board API and Approve Requirements and one sentence each in `README.md` and `docs/adoption.md`.

## Scope & Non-Goals

- **In-bounds:** `internal/web/server.go` and its tests, `internal/state/forgefiles.go` and its test, `openspec/specs/harness/spec.md`, the two doc sentences.
- **Non-goals:** request body limits, server timeouts, webhook replay and the approve lock are tracked in #474; binding the approval digest is #464. No allow-list setting for extra hosts: a token is the supported way to reach the board remotely.

## Definition of Done

- Tests: a loopback peer with Host `attacker.example:8420` gets 403 on `GET /api/state` and on Approve, and Approve makes no forge call; loopback Hosts still succeed; a loopback Host with a foreign `Origin` is refused; a token still admits any Host; invalid branch names get 400 with no forge call; a Gitea branch with `/`, `%`, `?` and `#` reaches the forge as the expected escaped path.
- `task mutate:check` reports `detection` for removing the Host check, the Origin check, the branch validation and the Gitea escape.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` pass, and `task spec:check` reports only this change's approval hold until the owner approves.
