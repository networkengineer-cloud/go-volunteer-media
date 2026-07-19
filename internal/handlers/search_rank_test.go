package handlers

import (
	"reflect"
	"testing"
)

func TestFuseRankedIDs_UnionsAndOrdersByCombinedScore(t *testing.T) {
	// ID 1: keyword rank 1 (best), absent from semantic.
	// ID 2: keyword rank 2, semantic rank 1 (best) — should outrank ID 1
	//       since it scores on both lists.
	// ID 3: absent from keyword, semantic rank 2.
	keyword := []uint{1, 2}
	semantic := []uint{2, 3}

	ordered, scores := fuseRankedIDs(keyword, semantic)

	want := []uint{2, 1, 3}
	if !reflect.DeepEqual(ordered, want) {
		t.Fatalf("expected order %v, got %v", want, ordered)
	}

	// ID 2's score: keyword rank 2 (1/(60+2)) + semantic rank 1 (1/(60+1)).
	expectedID2 := 1.0/62.0 + 1.0/61.0
	if got := scores[2]; got < expectedID2-1e-15 || got > expectedID2+1e-15 {
		t.Fatalf("expected score[2] == %v, got %v", expectedID2, got)
	}
}

func TestFuseRankedIDs_TiesBrokenByIDAscending(t *testing.T) {
	// Two IDs that appear in neither list together — same score (both rank 1
	// in their respective single list) — must be ordered deterministically.
	ordered, _ := fuseRankedIDs([]uint{5}, []uint{3})
	want := []uint{3, 5}
	if !reflect.DeepEqual(ordered, want) {
		t.Fatalf("expected tie-break order %v, got %v", want, ordered)
	}
}

func TestFuseRankedIDs_EmptyListsReturnEmpty(t *testing.T) {
	ordered, scores := fuseRankedIDs(nil, nil)
	if len(ordered) != 0 {
		t.Fatalf("expected empty result, got %v", ordered)
	}
	if len(scores) != 0 {
		t.Fatalf("expected empty scores map, got %v", scores)
	}
}

func TestFuseRankedIDs_OnlyKeyword(t *testing.T) {
	ordered, _ := fuseRankedIDs([]uint{10, 20, 30}, nil)
	want := []uint{10, 20, 30}
	if !reflect.DeepEqual(ordered, want) {
		t.Fatalf("expected keyword-only order preserved: %v, got %v", want, ordered)
	}
}

func TestCandidatePoolSize_FloorsAtMinimum(t *testing.T) {
	if got := candidatePoolSize(0, 10); got != 50 {
		t.Fatalf("expected floor of 50 for offset=0,limit=10, got %d", got)
	}
}

func TestCandidatePoolSize_GrowsToCoverDeepPage(t *testing.T) {
	if got := candidatePoolSize(200, 10); got != 210 {
		t.Fatalf("expected pool of 210 to cover offset=200,limit=10, got %d", got)
	}
}

func TestCandidatePoolSize_CapsAtMaximum(t *testing.T) {
	if got := candidatePoolSize(10000, 100); got != 500 {
		t.Fatalf("expected pool capped at 500, got %d", got)
	}
}
