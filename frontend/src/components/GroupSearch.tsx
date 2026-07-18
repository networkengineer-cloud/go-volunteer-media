import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import { searchApi } from '../api/client';
import type { SearchAnimalResult, SearchCommentResult } from '../api/client';
import { useDebounce } from '../hooks/useDebounce';
import EmptyState from './EmptyState';
import ErrorState from './ErrorState';
import SkeletonLoader from './SkeletonLoader';
import './GroupSearch.css';

const PAGE_SIZE = 10;
const DEBOUNCE_MS = 400;
const SNIPPET_LENGTH = 160;

type SearchType = 'all' | 'animals' | 'comments';

interface GroupSearchProps {
  groupId: number;
}

function snippet(text: string, length = SNIPPET_LENGTH): string {
  const trimmed = text.trim();
  if (trimmed.length <= length) return trimmed;
  return trimmed.slice(0, length).trimEnd() + '…';
}

const searchIcon = (
  <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="11" cy="11" r="7" />
    <line x1="21" y1="21" x2="16.65" y2="16.65" />
  </svg>
);

const GroupSearch: React.FC<GroupSearchProps> = ({ groupId }) => {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebounce(query.trim(), DEBOUNCE_MS);
  const [type, setType] = useState<SearchType>('all');

  const [animals, setAnimals] = useState<SearchAnimalResult[]>([]);
  const [comments, setComments] = useState<SearchCommentResult[]>([]);
  const [totalAnimals, setTotalAnimals] = useState(0);
  const [totalComments, setTotalComments] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Tracks the in-flight request so a newer call can cancel an older one —
  // without this, a slower earlier response could resolve after a faster
  // later one and overwrite fresher results with stale data.
  const abortControllerRef = useRef<AbortController | null>(null);

  const runSearch = useCallback(
    async (q: string, searchType: SearchType, requestOffset: number, append: boolean) => {
      abortControllerRef.current?.abort();
      const controller = new AbortController();
      abortControllerRef.current = controller;

      if (!q) {
        setAnimals([]);
        setComments([]);
        setTotalAnimals(0);
        setTotalComments(0);
        setError(null);
        return;
      }

      if (append) setLoadingMore(true);
      else setLoading(true);
      setError(null);

      try {
        const response = await searchApi.search(
          groupId,
          { q, type: searchType, limit: PAGE_SIZE, offset: requestOffset },
          { signal: controller.signal }
        );
        const data = response.data;
        setAnimals((prev) => (append ? [...prev, ...(data.animals ?? [])] : data.animals ?? []));
        setComments((prev) => (append ? [...prev, ...(data.comments ?? [])] : data.comments ?? []));
        setTotalAnimals(data.total_animals ?? 0);
        setTotalComments(data.total_comments ?? 0);
      } catch (err) {
        // A superseded/cancelled request rejects here too — ignore it
        // silently rather than surfacing an error for a search the user
        // has already moved on from.
        if (axios.isCancel(err)) return;
        console.error('Failed to search:', err);
        setError('Failed to search. Please try again.');
      } finally {
        // Only the request that's still current should clear the loading
        // flags — an aborted, superseded request's `finally` must not flip
        // them back off after a newer request has already set them.
        if (abortControllerRef.current === controller) {
          setLoading(false);
          setLoadingMore(false);
        }
      }
    },
    [groupId]
  );

  // Group switch: clear results immediately, before the new search below
  // even resolves. Without this, for the brief window until the new
  // request completes, the previous group's animal/comment data would
  // still render — but under Links built from the *new* groupId, pointing
  // at the *old* group's animal IDs.
  useEffect(() => {
    setAnimals([]);
    setComments([]);
    setTotalAnimals(0);
    setTotalComments(0);
    setError(null);
  }, [groupId]);

  // New query, type, or group: reset pagination and (re)search from the
  // first page.
  useEffect(() => {
    setOffset(0);
    runSearch(debouncedQuery, type, 0, false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedQuery, type, groupId]);

  // Cancel any in-flight request on unmount so it can't set state on an
  // unmounted component.
  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
    };
  }, []);

  const handleLoadMore = () => {
    const nextOffset = offset + PAGE_SIZE;
    setOffset(nextOffset);
    runSearch(debouncedQuery, type, nextOffset, true);
  };

  const showAnimals = type === 'all' || type === 'animals';
  const showComments = type === 'all' || type === 'comments';
  const hasResults = animals.length > 0 || comments.length > 0;
  const canLoadMore =
    (showAnimals && animals.length < totalAnimals) || (showComments && comments.length < totalComments);

  return (
    <div className="group-search">
      <div className="group-search__controls">
        <div className="group-search__input-wrap">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search animals and comments (e.g. &quot;resource guarding&quot;, &quot;playgroup&quot;)..."
            aria-label="Search animals and comments"
            className="group-search__input"
          />
        </div>
        <select
          value={type}
          onChange={(e) => setType(e.target.value as SearchType)}
          aria-label="Limit search to"
          className="group-search__type"
        >
          <option value="all">Animals &amp; Comments</option>
          <option value="animals">Animals Only</option>
          <option value="comments">Comments Only</option>
        </select>
      </div>

      {!debouncedQuery ? null : loading ? (
        <div className="group-search__results">
          <SkeletonLoader variant="card" count={2} />
        </div>
      ) : error ? (
        <div className="group-search__results">
          <ErrorState message={error} onRetry={() => runSearch(debouncedQuery, type, 0, false)} />
        </div>
      ) : !hasResults ? (
        <div className="group-search__results">
          <EmptyState
            icon={searchIcon}
            title="No matches found"
            description={`Nothing matched "${debouncedQuery}". Try a different word or phrase.`}
          />
        </div>
      ) : (
        <div className="group-search__results" role="status" aria-live="polite">
          {showAnimals && animals.length > 0 && (
            <div className="group-search__section">
              <h3 className="group-search__section-title">
                Animals <span className="group-search__count">({totalAnimals})</span>
              </h3>
              <ul className="group-search__animal-list">
                {animals.map((animal) => (
                  <li key={animal.id}>
                    <Link
                      to={`/groups/${groupId}/animals/${animal.id}/view`}
                      className="group-search__animal-result"
                    >
                      <span className="group-search__animal-name">{animal.name}</span>
                      <span className="group-search__animal-meta">
                        {[animal.species, animal.breed].filter(Boolean).join(' · ')}
                      </span>
                      {animal.description && (
                        <span className="group-search__snippet">{snippet(animal.description)}</span>
                      )}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {showComments && comments.length > 0 && (
            <div className="group-search__section">
              <h3 className="group-search__section-title">
                Comments <span className="group-search__count">({totalComments})</span>
              </h3>
              <ul className="group-search__comment-list">
                {comments.map((comment) => (
                  <li key={comment.id}>
                    <Link
                      to={`/groups/${groupId}/animals/${comment.animal_id}/view`}
                      className="group-search__comment-result"
                    >
                      <span className="group-search__comment-animal">on {comment.animal_name}</span>
                      <span className="group-search__snippet">{snippet(comment.content)}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {canLoadMore && (
            <button
              type="button"
              className="btn-secondary group-search__load-more"
              onClick={handleLoadMore}
              disabled={loadingMore}
            >
              {loadingMore ? 'Loading…' : 'Load more results'}
            </button>
          )}
        </div>
      )}
    </div>
  );
};

export default GroupSearch;
