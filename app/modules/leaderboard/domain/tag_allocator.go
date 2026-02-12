package leaderboarddomain

import (
	"cmp"
	"maps"
	"slices"
)

// TagChange represents a single tag reassignment produced by the allocator.
type TagChange struct {
	TagNumber   int
	OldMemberID string // empty if tag was unclaimed
	NewMemberID string
}

// TagAllocationInput represents a participant's finish position and current tag.
type TagAllocationInput struct {
	MemberID   string
	FinishRank int // 1-based finish position (lower = better)
	CurrentTag int // 0 = no tag (not in pool)
}

// AllocateTagsClosedPool performs closed-pool tag reassignment.
//
// Only participants who already hold a tag (CurrentTag > 0) are eligible.
// The pool consists of the tags held by eligible participants.
// Tags are reassigned by finish order: best finisher gets best (lowest) tag.
//
// Deterministic tie-breaking: when two participants share a FinishRank,
// the one with the lower current tag keeps priority.
//
// Returns the list of tag changes that occurred (only actual changes, not no-ops).
func AllocateTagsClosedPool(inputs []TagAllocationInput) []TagChange {
	// Separate eligible (have a tag) from ineligible
	var eligible []TagAllocationInput
	for _, inp := range inputs {
		if inp.CurrentTag > 0 {
			eligible = append(eligible, inp)
		}
	}

	if len(eligible) == 0 {
		return nil
	}

	// Collect the tag pool from eligible participants
	tagPool := make([]int, len(eligible))
	for i, e := range eligible {
		tagPool[i] = e.CurrentTag
	}

	// Sort tag pool ascending (best tags first)
	slices.Sort(tagPool)

	// Sort eligible participants by finish rank, then by current tag for deterministic tie-break
	slices.SortFunc(eligible, func(a, b TagAllocationInput) int {
		if c := cmp.Compare(a.FinishRank, b.FinishRank); c != 0 {
			return c
		}
		return cmp.Compare(a.CurrentTag, b.CurrentTag)
	})

	// Build old tag -> member mapping for change detection
	tagToOldMember := make(map[int]string, len(eligible))
	for _, e := range eligible {
		tagToOldMember[e.CurrentTag] = e.MemberID
	}

	// Assign pooled tags by rank order
	var changes []TagChange
	for i, participant := range eligible {
		newTag := tagPool[i]
		if newTag == participant.CurrentTag {
			continue // no change
		}

		oldHolder := tagToOldMember[newTag]
		changes = append(changes, TagChange{
			TagNumber:   newTag,
			OldMemberID: oldHolder,
			NewMemberID: participant.MemberID,
		})
	}

	return changes
}

// AllocateTagsFromReset assigns tags 1..N based on qualifying round finish order.
// All existing tags for the guild should be cleared before calling this.
func AllocateTagsFromReset(finishOrder []string) []TagChange {
	changes := make([]TagChange, len(finishOrder))
	for i, memberID := range finishOrder {
		changes[i] = TagChange{
			TagNumber:   i + 1,
			NewMemberID: memberID,
		}
	}
	return changes
}

// MemberTagAssignment represents the resulting tag state for a member after allocation.
type MemberTagAssignment struct {
	MemberID string
	Tag      int
}

// ComputeFinalTagState takes current tag state and changes, returns the full resulting state.
func ComputeFinalTagState(
	currentState map[string]int, // memberID -> currentTag
	changes []TagChange,
) []MemberTagAssignment {
	// Copy current state
	result := make(map[string]int, len(currentState))
	maps.Copy(result, currentState)

	// Apply changes
	for _, ch := range changes {
		result[ch.NewMemberID] = ch.TagNumber
	}

	assignments := make([]MemberTagAssignment, 0, len(result))
	for memberID, tag := range result {
		assignments = append(assignments, MemberTagAssignment{
			MemberID: memberID,
			Tag:      tag,
		})
	}

	slices.SortFunc(assignments, func(a, b MemberTagAssignment) int {
		return cmp.Compare(a.Tag, b.Tag)
	})

	return assignments
}
