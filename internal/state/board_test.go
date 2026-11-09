package state

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// FuzzBoardViewPaging pages a generated mixed open/done board under every scope and limit, flipping card
// columns between pages, and checks the traversal against an independent sort-and-filter reference model.
// spec: openspec/specs/harness/spec.md#Requirement: Bounded board reads
func FuzzBoardViewPaging(f *testing.F) {
	f.Add([]byte{6, 1, 0, 1, 1, 3, 0, 3, 1, 0, 0, 2, 1}, uint8(0), uint8(2))
	f.Add([]byte{4, 2, 1, 2, 0, 2, 1, 2, 0, 1, 2, 3}, uint8(1), uint8(1))
	f.Add([]byte{0}, uint8(2), uint8(0))
	f.Add([]byte{11, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 2, 2, 1, 1, 0, 0, 1, 1, 7, 3, 5}, uint8(2), uint8(3))

	scopes := []string{BoardScopeOpen, BoardScopeDone, BoardScopeAll}
	f.Fuzz(func(t *testing.T, raw []byte, scopeSel, limitSel uint8) {
		next := func() int {
			if len(raw) == 0 {
				return 0
			}
			b := int(raw[0])
			raw = raw[1:]
			return b
		}
		n := next() % 12
		st := &State{}
		for i := 0; i < n; i++ {
			// Numbers collide on purpose so the ID tie-break is exercised.
			column := "backlog"
			if next()%2 == 1 {
				column = "done"
			}
			st.Cards = append(st.Cards, BoardCard{ID: fmt.Sprintf("C%02d", i), Number: next() % 5, Column: column})
		}
		scope := scopes[int(scopeSel)%len(scopes)]
		limit := int(limitSel) % (n + 2)

		matches := func(column string) bool {
			switch scope {
			case BoardScopeOpen:
				return column != "done"
			case BoardScopeDone:
				return column == "done"
			}
			return true
		}
		// throughout holds the IDs that match scope in every board state the traversal sees.
		throughout := map[string]bool{}
		for _, c := range st.Cards {
			throughout[c.ID] = matches(c.Column)
		}
		order := slices.Clone(st.Cards)
		slices.SortFunc(order, func(a, b BoardCard) int {
			return cmp.Or(cmp.Compare(b.Number, a.Number), strings.Compare(a.ID, b.ID))
		})
		rank := map[string]int{}
		for i, c := range order {
			rank[c.ID] = i
		}

		first, err := st.BoardView(BoardQuery{Scope: scope, Limit: limit})
		if err != nil {
			t.Fatalf("BoardView: %v", err)
		}
		wantTotal, wantDone := 0, 0
		for _, c := range st.Cards {
			if matches(c.Column) {
				wantTotal++
			}
			if c.Column == "done" {
				wantDone++
			}
		}
		if first.Total != wantTotal || first.Counts["done"] != wantDone || first.Scope != scope {
			t.Fatalf("first page scope=%q total=%d counts.done=%d, want %q %d %d", first.Scope, first.Total, first.Counts["done"], scope, wantTotal, wantDone)
		}

		seen := map[string]bool{}
		last := -1
		page := first
		for pages := 0; ; pages++ {
			if pages > n+1 {
				t.Fatalf("traversal did not finish after %d pages", pages)
			}
			if limit > 0 && len(page.Cards) > limit {
				t.Fatalf("page holds %d cards, limit %d", len(page.Cards), limit)
			}
			for _, c := range page.Cards {
				if seen[c.ID] {
					t.Fatalf("card %s returned twice", c.ID)
				}
				seen[c.ID] = true
				if rank[c.ID] <= last {
					t.Fatalf("card %s out of order", c.ID)
				}
				last = rank[c.ID]
			}
			if page.Complete != (page.NextAfter == "") {
				t.Fatalf("complete=%v with next_after=%q", page.Complete, page.NextAfter)
			}
			if page.Complete {
				break
			}
			if n > 0 {
				i := next() % n
				if st.Cards[i].Column == "done" {
					st.Cards[i].Column = "backlog"
				} else {
					st.Cards[i].Column = "done"
				}
				if !matches(st.Cards[i].Column) {
					throughout[st.Cards[i].ID] = false
				}
			}
			page, err = st.BoardView(BoardQuery{Scope: scope, Limit: limit, After: page.NextAfter})
			if err != nil {
				t.Fatalf("BoardView after %q: %v", page.NextAfter, err)
			}
		}
		for id, ok := range throughout {
			if ok && !seen[id] {
				t.Fatalf("card %s matched %s throughout but was never returned", id, scope)
			}
		}
	})
}

// TestBoardViewQuery covers history access, the unfiltered default, and argument validation.
// spec: openspec/specs/harness/spec.md#Requirement: Bounded board reads
func TestBoardViewQuery(t *testing.T) {
	st := &State{Cards: []BoardCard{
		{ID: "GH-1", Number: 1, Column: "done"},
		{ID: "GH-3", Number: 3, Column: "backlog"},
		{ID: "GH-2", Number: 2, Column: "done"},
	}}
	ids := func(s BoardSnapshot) string {
		var out []string
		for _, c := range s.Cards {
			out = append(out, c.ID)
		}
		return strings.Join(out, ",")
	}

	all, err := st.BoardView(BoardQuery{})
	if err != nil || all.Scope != BoardScopeAll || ids(all) != "GH-3,GH-2,GH-1" || !all.Complete || all.Total != 3 {
		t.Fatalf("empty query = %q scope=%q complete=%v total=%d err=%v", ids(all), all.Scope, all.Complete, all.Total, err)
	}
	open, _ := st.BoardView(BoardQuery{Scope: BoardScopeOpen})
	if ids(open) != "GH-3" || open.Counts["done"] != 2 || open.Counts["backlog"] != 1 {
		t.Fatalf("open = %q counts=%v", ids(open), open.Counts)
	}
	done, _ := st.BoardView(BoardQuery{Scope: BoardScopeDone, Limit: 1})
	if ids(done) != "GH-2" || done.NextAfter != "2:GH-2" || done.Complete || done.Total != 2 {
		t.Fatalf("done page 1 = %q next=%q complete=%v total=%d", ids(done), done.NextAfter, done.Complete, done.Total)
	}
	done, _ = st.BoardView(BoardQuery{Scope: BoardScopeDone, Limit: 1, After: done.NextAfter})
	if ids(done) != "GH-1" || !done.Complete {
		t.Fatalf("done page 2 = %q complete=%v", ids(done), done.Complete)
	}
	empty, _ := (&State{}).BoardView(BoardQuery{Scope: BoardScopeOpen})
	if empty.Cards == nil || !empty.Complete {
		t.Fatalf("empty board cards=%v complete=%v, want [] and complete", empty.Cards, empty.Complete)
	}

	for _, tc := range []struct {
		q    BoardQuery
		want string
	}{
		{BoardQuery{Scope: "closed"}, "scope:"},
		{BoardQuery{Limit: -1}, "limit:"},
		{BoardQuery{After: "GH-2"}, "after:"},
		{BoardQuery{After: "x:GH-2"}, "after:"},
		{BoardQuery{After: "2:"}, "after:"},
	} {
		snap, err := st.BoardView(tc.q)
		if err == nil || !strings.HasPrefix(err.Error(), tc.want) || snap.Cards != nil {
			t.Fatalf("BoardView(%+v) = %d cards, err %v; want error starting %q and no cards", tc.q, len(snap.Cards), err, tc.want)
		}
	}
}
