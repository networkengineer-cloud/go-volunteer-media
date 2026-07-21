package handlers

import "sort"

// rrfK is Reciprocal Rank Fusion's standard smoothing constant. RRF combines
// two differently-scaled ranked lists (ts_rank and cosine distance aren't on
// comparable scales) by rank position instead of raw score, so no tuning is
// needed to start.
const rrfK = 60

// fuseRankedIDs merges two rank-ordered ID lists (best match first) via
// Reciprocal Rank Fusion: score(id) = sum over lists containing id of
// 1/(rrfK + rank), where rank is the 1-based position in that list. An ID
// absent from a list contributes 0 for that list's term. Returns IDs ordered
// by combined score highest-first, ties broken by ID ascending so the result
// is deterministic, plus the score behind each ID (exposed so callers can
// surface it, e.g. as the API's per-row "rank" field).
func fuseRankedIDs(keywordIDs, semanticIDs []uint) (ordered []uint, scores map[uint]float64) {
	scores = make(map[uint]float64, len(keywordIDs)+len(semanticIDs))
	for i, id := range keywordIDs {
		scores[id] += 1.0 / float64(rrfK+i+1)
	}
	for i, id := range semanticIDs {
		scores[id] += 1.0 / float64(rrfK+i+1)
	}

	ordered = make([]uint, 0, len(scores))
	for id := range scores {
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if scores[ordered[i]] != scores[ordered[j]] {
			return scores[ordered[i]] > scores[ordered[j]]
		}
		return ordered[i] < ordered[j]
	})
	return ordered, scores
}

// Candidate pool bounds: floor covers the common case (a first page of
// normal-sized results) with room for the two ranked lists to actually
// overlap; cap bounds the cost of a very large offset.
const (
	minCandidatePool = 50
	maxCandidatePool = 500
)

// candidatePoolSize returns how many top matches to fetch from each of the
// keyword and semantic queries before fusing. Ideally it covers the
// requested page (offset+limit), so a "load more" page doesn't run out of
// fused candidates even though more matches exist in the database — but
// it's capped at maxCandidatePool regardless, trading unbounded pagination
// depth in semantic mode (once offset+limit exceeds the cap, results
// beyond position maxCandidatePool become unreachable, unlike keyword-only
// mode's real, uncapped Limit/Offset pagination) for a bounded per-request
// cost against the two source queries.
func candidatePoolSize(offset, limit int) int {
	needed := offset + limit
	switch {
	case needed < minCandidatePool:
		return minCandidatePool
	case needed > maxCandidatePool:
		return maxCandidatePool
	default:
		return needed
	}
}
