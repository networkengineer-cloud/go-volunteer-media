#!/usr/bin/env python3
"""
Hit myhaws's hybrid (keyword + semantic) search API with a fixed query set
and print/save the results, for judging semantic search relevance and
comparing before/after tuning changes (e.g. SEMANTIC_SEARCH_MAX_DISTANCE) on
a live instance. Matches internal/handlers/search.go:Search.

Uses only the Python standard library - no pip install required.

Usage:
    export MYHAWS_API_TOKEN="pat_xxxxx..."   # or a JWT from POST /api/login
    python3 scripts/search_relevance_test.py --base-url https://dev.t-wallace.com --group "Test Group"

    # Save a run, then after changing SEMANTIC_SEARCH_MAX_DISTANCE on the
    # instance (and restarting it), compare against the saved baseline:
    python3 scripts/search_relevance_test.py --base-url https://dev.t-wallace.com --group "Test Group" --out baseline.json
    python3 scripts/search_relevance_test.py --base-url https://dev.t-wallace.com --group "Test Group" --out tuned.json
    python3 scripts/search_relevance_test.py --compare baseline.json tuned.json

Options:
    --base-url    Defaults to https://dev.t-wallace.com.
    --group       Group name to match (case-insensitive substring). Required
                  unless --group-id is given.
    --group-id    Group ID to search, skips the name lookup.
    --queries     Path to a newline-delimited file of queries (one per
                  line, '#' comments allowed). Defaults to the built-in set
                  below, which spans three buckets: queries with obvious
                  keyword overlap in the data, paraphrases with no keyword
                  overlap (isolates what semantic search alone contributes),
                  and unrelated queries (checks the distance cutoff isn't
                  letting noise through).
    --type        all | animals | comments | updates (default: all)
    --limit       Results per query (default: 20)
    --out         Save full JSON results to this path for later comparison.
    --compare     Diff two previously-saved --out files instead of querying.
"""

from __future__ import annotations

import argparse
import json
import os
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone

DEFAULT_BASE_URL = "https://dev.t-wallace.com"

# (bucket, query) - the bucket label is for organizing output only, not sent
# to the API.
DEFAULT_QUERIES = [
    ("keyword-overlap", "resource guarding"),
    ("keyword-overlap", "playgroup"),
    ("keyword-overlap", "leash reactive"),
    ("keyword-overlap", "counter surfing"),
    ("keyword-overlap", "separation anxiety"),
    ("semantic-only", "doesn't get along with other dogs"),
    ("semantic-only", "nervous around new people"),
    ("semantic-only", "jumps up on visitors"),
    ("semantic-only", "shouldn't go to a home with cats"),
    ("semantic-only", "guards food and toys"),
    ("semantic-only", "scared of loud noises"),
    ("semantic-only", "good with kids"),
    ("unrelated", "weather forecast this weekend"),
    ("unrelated", "car oil change schedule"),
    ("unrelated", "stock market update"),
]


def load_queries(path: str | None) -> list[tuple[str, str]]:
    if not path:
        return DEFAULT_QUERIES
    queries = []
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#"):
                queries.append(("custom", line))
    return queries


def api_request(base_url: str, path: str, token: str, params: dict | None = None) -> dict:
    url = base_url.rstrip("/") + path
    if params:
        url += "?" + urllib.parse.urlencode(params)
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {token}"})
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        raise SystemExit(f"HTTP {e.code} calling {url}: {body}")


def resolve_group_id(base_url: str, token: str, group_name: str | None, group_id: str | None) -> str:
    if group_id:
        return group_id
    groups = api_request(base_url, "/api/groups", token)
    matches = [g for g in groups if group_name.lower() in g["name"].lower()]
    if not matches:
        names = [g["name"] for g in groups]
        raise SystemExit(f"No group matching {group_name!r} found among: {names}")
    if len(matches) > 1:
        names = [g["name"] for g in matches]
        raise SystemExit(f"Ambiguous group name {group_name!r}, matches: {names}")
    return str(matches[0]["id"])


def summarize_result(row: dict, kind: str) -> str:
    rank = row.get("rank", 0)
    if kind == "animals":
        return f"  [{rank:.4f}] id={row.get('id')} {row.get('name')}"
    if kind == "comments":
        text = (row.get("content") or "")[:60]
        return f"  [{rank:.4f}] id={row.get('id')} on {row.get('animal_name')}: {text!r}"
    if kind == "updates":
        text = (row.get("title") or row.get("content") or "")[:60]
        return f"  [{rank:.4f}] id={row.get('id')} {text!r}"
    return f"  {row}"


def run_queries(base_url: str, token: str, group_id: str, queries: list[tuple[str, str]],
                 search_type: str, limit: int) -> dict:
    results = {}
    for bucket, q in queries:
        resp = api_request(
            base_url, f"/api/groups/{group_id}/search", token,
            {"q": q, "type": search_type, "limit": limit},
        )
        results[q] = {"bucket": bucket, "response": resp}

        print(f"\n=== [{bucket}] {q!r} ===")
        found_any = False
        for kind in ("animals", "comments", "updates"):
            rows = resp.get(kind)
            if rows:
                found_any = True
                print(f" {kind}:")
                for row in rows:
                    print(summarize_result(row, kind))
        if not found_any:
            print("  (no results)")
    return results


def top_ids(resp: dict) -> list[tuple[str, object]]:
    ids = []
    for kind in ("animals", "comments", "updates"):
        for row in resp.get(kind) or []:
            ids.append((kind, row.get("id")))
    return ids


def compare(path_a: str, path_b: str) -> None:
    with open(path_a, encoding="utf-8") as f:
        a = json.load(f)
    with open(path_b, encoding="utf-8") as f:
        b = json.load(f)

    queries = sorted(set(a["results"]) | set(b["results"]))
    changed = 0
    for q in queries:
        ra = a["results"].get(q, {}).get("response", {})
        rb = b["results"].get(q, {}).get("response", {})
        ids_a = top_ids(ra)
        ids_b = top_ids(rb)
        if ids_a != ids_b:
            changed += 1
            print(f"\n=== CHANGED: {q!r} ===")
            print(f"  {os.path.basename(path_a)}: {ids_a}")
            print(f"  {os.path.basename(path_b)}: {ids_b}")

    print(f"\n{changed}/{len(queries)} queries changed top results.")


def main() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL)
    parser.add_argument("--group")
    parser.add_argument("--group-id")
    parser.add_argument("--queries", help="Path to newline-delimited query file")
    parser.add_argument("--type", default="all", choices=["all", "animals", "comments", "updates"])
    parser.add_argument("--limit", type=int, default=20)
    parser.add_argument("--out", help="Save full JSON results to this path")
    parser.add_argument(
        "--compare", nargs=2, metavar=("BASELINE", "OTHER"),
        help="Diff two saved --out files instead of querying",
    )
    args = parser.parse_args()

    if args.compare:
        compare(*args.compare)
        return

    token = os.environ.get("MYHAWS_API_TOKEN")
    if not token:
        raise SystemExit(
            "Set MYHAWS_API_TOKEN to a Bearer token "
            "(a pat_... API token, or a JWT from POST /api/login)"
        )

    if not args.group and not args.group_id:
        raise SystemExit("Pass --group <name> or --group-id <id>")

    group_id = resolve_group_id(args.base_url, token, args.group, args.group_id)
    queries = load_queries(args.queries)
    results = run_queries(args.base_url, token, group_id, queries, args.type, args.limit)

    if args.out:
        with open(args.out, "w", encoding="utf-8") as f:
            json.dump(
                {
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                    "base_url": args.base_url,
                    "group_id": group_id,
                    "type": args.type,
                    "results": results,
                },
                f,
                indent=2,
            )
        print(f"\nSaved results to {args.out}")


if __name__ == "__main__":
    main()
