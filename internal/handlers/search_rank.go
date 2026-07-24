package handlers

import (
	"os"
	"sort"
	"strconv"
)

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

// defaultMaxSemanticDistance is maxSemanticDistance's value when
// SEMANTIC_SEARCH_MAX_DISTANCE isn't set. Cosine distance ranges 0
// (identical) to 2 (opposite); 0.75 is a conservative starting cutoff meant
// to exclude clearly-unrelated content while still admitting genuinely
// related matches — it hasn't been validated against real Voyage embedding
// output (no VOYAGE_API_KEY in dev/test) and should be tuned from production
// search logs once real query/document embedding distances are observable.
const defaultMaxSemanticDistance = 0.75

// maxSemanticDistance bounds the semantic candidate queries (see search.go)
// to rows genuinely similar to the query embedding. Without it, "embedding IS
// NOT NULL ORDER BY embedding <=> ? LIMIT pool" always returns `pool` rows
// regardless of whether any of them are actually related to the query — with
// a small group (fewer embedded rows than the pool floor), that means every
// row in the group comes back, merely sorted by how (un)related it is,
// letting fully irrelevant rows surface as "matches" whenever the keyword
// query finds nothing to fuse against.
//
// Overridable via SEMANTIC_SEARCH_MAX_DISTANCE (e.g. "0.8") so
// defaultMaxSemanticDistance can be corrected from production search logs
// (see finishSemanticSearch's candidate-count log line) without a code
// change and redeploy. Read via os.Getenv per call, not cached, matching
// embedding.SemanticSearchEnabled's pattern — cheap enough per-request, and
// keeps it trivially overridable in tests via t.Setenv.
func maxSemanticDistance() float64 {
	if v := os.Getenv("SEMANTIC_SEARCH_MAX_DISTANCE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultMaxSemanticDistance
}

// semanticDistanceClause returns the "<column> <=> ? < ?" fragment each
// resource's semantic candidate query in search.go uses to bound results to
// maxSemanticDistance. Centralized so the threshold/operator can't drift out
// of sync across the animals/comments/updates queries, which are otherwise
// identical in shape (see fuseResults' and finishSemanticSearch's doc
// comments for why those are shared too). Callers still supply the query
// vector and maxSemanticDistance as bind args, in that order, after this
// fragment's own args.
func semanticDistanceClause(embeddingColumn string) string {
	return embeddingColumn + " <=> ? < ?"
}

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
