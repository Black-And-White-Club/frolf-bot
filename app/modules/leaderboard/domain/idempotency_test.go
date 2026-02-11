package leaderboarddomain

import "testing"

func TestComputeProcessingHashDeterministic(t *testing.T) {
	inputA := []RoundInput{
		{MemberID: "alice", FinishRank: 2},
		{MemberID: "bob", FinishRank: 1},
	}
	inputB := []RoundInput{
		{MemberID: "bob", FinishRank: 1},
		{MemberID: "alice", FinishRank: 2},
	}

	hashA := ComputeProcessingHash(inputA)
	hashB := ComputeProcessingHash(inputB)
	if hashA != hashB {
		t.Fatalf("expected equal hashes for equivalent input sets, got %s and %s", hashA, hashB)
	}
}

func TestComputeProcessingHashChangesWhenInputChanges(t *testing.T) {
	base := []RoundInput{
		{MemberID: "alice", FinishRank: 1},
		{MemberID: "bob", FinishRank: 2},
	}
	changed := []RoundInput{
		{MemberID: "alice", FinishRank: 2},
		{MemberID: "bob", FinishRank: 1},
	}

	baseHash := ComputeProcessingHash(base)
	changedHash := ComputeProcessingHash(changed)
	if baseHash == changedHash {
		t.Fatalf("expected different hashes when finish ranks change")
	}
}
