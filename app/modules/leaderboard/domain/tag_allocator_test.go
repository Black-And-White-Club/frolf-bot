package leaderboarddomain

import "testing"

func TestAllocateTagsClosedPool(t *testing.T) {
	t.Run("reassigns pooled tags by finish rank", func(t *testing.T) {
		changes := AllocateTagsClosedPool([]TagAllocationInput{
			{MemberID: "alice", FinishRank: 2, CurrentTag: 1},
			{MemberID: "bob", FinishRank: 1, CurrentTag: 2},
			{MemberID: "charlie", FinishRank: 3, CurrentTag: 0}, // no tag -> not in pool
		})

		if len(changes) != 2 {
			t.Fatalf("expected 2 changes, got %d", len(changes))
		}

		if changes[0].TagNumber != 1 || changes[0].NewMemberID != "bob" || changes[0].OldMemberID != "alice" {
			t.Fatalf("unexpected first change: %+v", changes[0])
		}
		if changes[1].TagNumber != 2 || changes[1].NewMemberID != "alice" || changes[1].OldMemberID != "bob" {
			t.Fatalf("unexpected second change: %+v", changes[1])
		}
	})

	t.Run("uses deterministic tie break on current tag", func(t *testing.T) {
		changes := AllocateTagsClosedPool([]TagAllocationInput{
			{MemberID: "alice", FinishRank: 1, CurrentTag: 2},
			{MemberID: "bob", FinishRank: 1, CurrentTag: 1},
		})

		if len(changes) != 0 {
			t.Fatalf("expected no changes for stable tie-break ordering, got %d", len(changes))
		}
	})
}

func TestAllocateTagsFromReset(t *testing.T) {
	changes := AllocateTagsFromReset([]string{"alice", "bob", "charlie"})
	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}

	for i := range changes {
		if changes[i].TagNumber != i+1 {
			t.Fatalf("expected tag %d at index %d, got %d", i+1, i, changes[i].TagNumber)
		}
	}
}

func TestComputeFinalTagState(t *testing.T) {
	state := map[string]int{
		"alice": 1,
		"bob":   2,
	}
	changes := []TagChange{
		{TagNumber: 1, NewMemberID: "bob"},
		{TagNumber: 2, NewMemberID: "alice"},
	}

	assignments := ComputeFinalTagState(state, changes)
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}

	if assignments[0].MemberID != "bob" || assignments[0].Tag != 1 {
		t.Fatalf("unexpected assignment[0]: %+v", assignments[0])
	}
	if assignments[1].MemberID != "alice" || assignments[1].Tag != 2 {
		t.Fatalf("unexpected assignment[1]: %+v", assignments[1])
	}
}
