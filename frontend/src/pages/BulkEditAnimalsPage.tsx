import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { animalsApi, groupsApi } from '../api/client';
import type { Animal, Group } from '../api/client';
import { useToast } from '../hooks/useToast';
import { formatDateShort, calculateQuarantineEndDate, calculateDaysSince } from '../utils/dateUtils';
import { useDebounce } from '../hooks/useDebounce';
import EmptyState from '../components/EmptyState';
import SkeletonLoader from '../components/SkeletonLoader';
import ErrorState from '../components/ErrorState';
import Modal from '../components/Modal';
import './BulkEditAnimalsPage.css';

const BulkEditAnimalsPage: React.FC = () => {
  const toast = useToast();
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [selectedAnimalIds, setSelectedAnimalIds] = useState<Set<number>>(new Set());
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterGroup, setFilterGroup] = useState<string>('');
  const [filterName, setFilterName] = useState<string>('');
  const [bulkAction, setBulkAction] = useState<string>('');
  const [bulkGroupId, setBulkGroupId] = useState<string>('');
  const [bulkStatus, setBulkStatus] = useState<string>('');
  const [showImportModal, setShowImportModal] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importResult, setImportResult] = useState<{ message: string; warnings?: string[] } | null>(null);
  const [updatingAnimal, setUpdatingAnimal] = useState<number | null>(null);
  const [viewMode, setViewMode] = useState<'cards' | 'table'>('cards');

  // Debounce the name filter to reduce API calls
  const debouncedFilterName = useDebounce(filterName, 300);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [animalsRes, groupsRes] = await Promise.all([
        animalsApi.getAllForAdmin(
          filterStatus === 'all' ? undefined : filterStatus,
          filterGroup ? parseInt(filterGroup) : undefined,
          debouncedFilterName || undefined
        ),
        groupsApi.getAll(),
      ]);
      setAnimals(animalsRes.data);
      setGroups(groupsRes.data);
    } catch (err: unknown) {
      console.error('Failed to load data:', err);
      const errorObj = err as { response?: { data?: { error?: string } } };
      const errorMsg = errorObj.response?.data?.error || 'Failed to load animals. Please try again.';
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  }, [filterStatus, filterGroup, debouncedFilterName]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Calculate statistics
  const stats = useMemo(() => {
    const total = animals.length;
    const available = animals.filter(a => a.status === 'available').length;
    const foster = animals.filter(a => a.status === 'foster').length;
    const quarantine = animals.filter(a => a.status === 'bite_quarantine').length;
    const archived = animals.filter(a => a.status === 'archived').length;
    const avgStay = total > 0
      ? Math.round(animals.reduce((sum, a) => sum + calculateDaysSince(a.arrival_date), 0) / total)
      : 0;

    return { total, available, foster, quarantine, archived, avgStay };
  }, [animals]);

  const handleSelectAll = () => {
    if (selectedAnimalIds.size === animals.length) {
      setSelectedAnimalIds(new Set());
    } else {
      setSelectedAnimalIds(new Set(animals.map((a) => a.id)));
    }
  };

  const handleSelectAnimal = (id: number) => {
    const newSelection = new Set(selectedAnimalIds);
    if (newSelection.has(id)) {
      newSelection.delete(id);
    } else {
      newSelection.add(id);
    }
    setSelectedAnimalIds(newSelection);
  };

  const handleIndividualStatusChange = async (animalId: number, newStatus: string) => {
    setUpdatingAnimal(animalId);
    try {
      await animalsApi.updateAnimal(animalId, { status: newStatus });
      setAnimals(animals.map(a =>
        a.id === animalId ? { ...a, status: newStatus } : a
      ));
      toast.showSuccess('Status updated successfully');
    } catch (err: unknown) {
      console.error('Failed to update status:', err);
      const errorObj = err as { response?: { data?: { error?: string } } };
      const errorMsg = errorObj.response?.data?.error || 'Failed to update status. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUpdatingAnimal(null);
    }
  };

  const handleIndividualGroupChange = async (animalId: number, newGroupId: number) => {
    setUpdatingAnimal(animalId);
    try {
      await animalsApi.updateAnimal(animalId, { group_id: newGroupId });
      setAnimals(animals.map(a =>
        a.id === animalId ? { ...a, group_id: newGroupId } : a
      ));
      toast.showSuccess('Group updated successfully');
    } catch (err: unknown) {
      console.error('Failed to update group:', err);
      const errorObj = err as { response?: { data?: { error?: string } } };
      const errorMsg = errorObj.response?.data?.error || 'Failed to update group. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUpdatingAnimal(null);
    }
  };

  const handleBulkUpdate = async () => {
    if (selectedAnimalIds.size === 0) {
      toast.showWarning('Please select at least one animal');
      return;
    }

    if (bulkAction === '') {
      toast.showWarning('Please select an action');
      return;
    }

    if (bulkAction === 'move' && !bulkGroupId) {
      toast.showWarning('Please select a target group');
      return;
    }

    if (bulkAction === 'status' && !bulkStatus) {
      toast.showWarning('Please select a status');
      return;
    }

    try {
      const animalIds = Array.from(selectedAnimalIds);
      const groupId = bulkAction === 'move' ? parseInt(bulkGroupId) : undefined;
      const status = bulkAction === 'status' ? bulkStatus : undefined;

      const response = await animalsApi.bulkUpdate(animalIds, groupId, status);
      toast.showSuccess(response.data.message);
      setSelectedAnimalIds(new Set());
      setBulkAction('');
      setBulkGroupId('');
      setBulkStatus('');
      loadData();
    } catch (err: unknown) {
      console.error('Failed to bulk update:', err);
      const errorObj = err as { response?: { data?: { error?: string } } };
      const errorMsg = errorObj.response?.data?.error || 'Failed to update animals. Please try again.';
      toast.showError(errorMsg);
    }
  };

  const handleCancelBulk = () => {
    setSelectedAnimalIds(new Set());
    setBulkAction('');
    setBulkGroupId('');
    setBulkStatus('');
  };

  const handleExportCSV = async () => {
    try {
      const response = await animalsApi.exportCSV(
        filterGroup ? parseInt(filterGroup) : undefined
      );

      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', 'animals.csv');
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      toast.showSuccess('Animals exported to CSV successfully!');
    } catch (err: unknown) {
      console.error('Failed to export CSV:', err);
      const errorObj = err as { response?: { data?: { error?: string } } };
      const errorMsg = errorObj.response?.data?.error || 'Failed to export CSV. Please try again.';
      toast.showError(errorMsg);
    }
  };

  const handleExportCommentsCSV = async () => {
    try {
      const response = await animalsApi.exportCommentsCSV(
        filterGroup ? parseInt(filterGroup) : undefined
      );

      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', 'animal-comments.csv');
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      toast.showSuccess('Comments exported to CSV successfully!');
    } catch (err: unknown) {
      console.error('Failed to export comments CSV:', err);
      const errorObj = err as { response?: { data?: { error?: string } } };
      const errorMsg = errorObj.response?.data?.error || 'Failed to export comments. Please try again.';
      toast.showError(errorMsg);
    }
  };

  const handleImportCSV = async () => {
    if (!importFile) {
      toast.showWarning('Please select a CSV file');
      return;
    }

    try {
      const response = await animalsApi.importCSV(importFile);
      setImportResult(response.data);
      setImportFile(null);
      loadData();
      toast.showSuccess('CSV imported successfully!');
    } catch (err: unknown) {
      console.error('Failed to import CSV:', err);
      const errorObj = err as { response?: { data?: { error?: string } } };
      const errorMsg = errorObj.response?.data?.error || 'Failed to import CSV. Please check the file format and try again.';
      toast.showError(errorMsg);
    }
  };

  const clearFilters = () => {
    setFilterStatus('all');
    setFilterGroup('');
    setFilterName('');
  };

  if (loading) {
    return (
      <div className="bulk-edit-page">
        <div className="page-container">
          <div className="page-header">
            <SkeletonLoader variant="text" width="300px" height="2rem" />
            <div className="header-actions">
              <SkeletonLoader variant="rectangular" width="120px" height="40px" />
              <SkeletonLoader variant="rectangular" width="120px" height="40px" />
              <SkeletonLoader variant="rectangular" width="120px" height="40px" />
            </div>
          </div>
          <div className="stats-grid">
            {[1, 2, 3, 4, 5, 6].map(i => (
              <SkeletonLoader key={i} variant="rectangular" height="100px" />
            ))}
          </div>
          <SkeletonLoader variant="rectangular" height="60px" />
          <div className="animals-grid">
            {[1, 2, 3, 4, 5, 6].map(i => (
              <SkeletonLoader key={i} variant="rectangular" height="200px" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bulk-edit-page">
        <div className="page-container">
          <ErrorState
            title="Unable to Load Animals"
            message={error}
            onRetry={loadData}
            onGoBack={() => window.history.back()}
            goBackLabel="Go Back"
          />
        </div>
      </div>
    );
  }

  return (
    <div className="bulk-edit-page">
      <div className="page-container">
        {/* Header */}
        <div className="page-header">
          <div className="header-content">
            <h1>Bulk Animal Management</h1>
            <p className="page-subtitle">Manage and update multiple animals at once</p>
          </div>
          <div className="header-actions">
            <button onClick={handleExportCommentsCSV} className="btn-secondary btn-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              Export Comments
            </button>
            <button onClick={handleExportCSV} className="btn-secondary btn-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="7 10 12 15 17 10" />
                <line x1="12" y1="15" x2="12" y2="3" />
              </svg>
              Export Animals
            </button>
            <button onClick={() => setShowImportModal(true)} className="btn-primary btn-icon">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                <polyline points="17 8 12 3 7 8" />
                <line x1="12" y1="3" x2="12" y2="15" />
              </svg>
              Import CSV
            </button>
          </div>
        </div>

        {/* Statistics Dashboard */}
        <div className="stats-grid">
          <div className="stat-card">
            <div className="stat-icon total">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-label">Total Animals</div>
              <div className="stat-value">{stats.total}</div>
            </div>
          </div>

          <div className="stat-card">
            <div className="stat-icon available">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-label">Available</div>
              <div className="stat-value">{stats.available}</div>
            </div>
          </div>

          <div className="stat-card">
            <div className="stat-icon foster">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
                <polyline points="9 22 9 12 15 12 15 22" />
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-label">In Foster</div>
              <div className="stat-value">{stats.foster}</div>
            </div>
          </div>

          <div className="stat-card">
            <div className="stat-icon quarantine">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-label">Quarantine</div>
              <div className="stat-value">{stats.quarantine}</div>
            </div>
          </div>

          <div className="stat-card">
            <div className="stat-icon archived">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="21 8 21 21 3 21 3 8" />
                <rect x="1" y="3" width="22" height="5" />
                <line x1="10" y1="12" x2="14" y2="12" />
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-label">Archived</div>
              <div className="stat-value">{stats.archived}</div>
            </div>
          </div>

          <div className="stat-card">
            <div className="stat-icon avg-stay">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                <line x1="16" y1="2" x2="16" y2="6" />
                <line x1="8" y1="2" x2="8" y2="6" />
                <line x1="3" y1="10" x2="21" y2="10" />
              </svg>
            </div>
            <div className="stat-content">
              <div className="stat-label">Avg. Length of Stay</div>
              <div className="stat-value">{stats.avgStay} days</div>
            </div>
          </div>
        </div>

        {/* Filters */}
        <div className="filters-section">
          <div className="filters-header">
            <h2>Filters</h2>
            {(filterStatus !== 'all' || filterGroup || filterName) && (
              <button onClick={clearFilters} className="btn-text">
                Clear All
              </button>
            )}
          </div>
          <div className="filters-grid">
            <div className="filter-field">
              <label htmlFor="filter-status">Status</label>
              <select
                id="filter-status"
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                className="filter-select"
              >
                <option value="all">All Statuses</option>
                <option value="available">Available</option>
                <option value="foster">Foster</option>
                <option value="bite_quarantine">Bite Quarantine</option>
                <option value="archived">Archived</option>
              </select>
            </div>

            <div className="filter-field">
              <label htmlFor="filter-group">Group</label>
              <select
                id="filter-group"
                value={filterGroup}
                onChange={(e) => setFilterGroup(e.target.value)}
                className="filter-select"
              >
                <option value="">All Groups</option>
                {groups.map((group) => (
                  <option key={group.id} value={group.id}>
                    {group.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="filter-field">
              <label htmlFor="filter-name">Search Name</label>
              <div className="search-input-wrapper">
                <svg className="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="11" cy="11" r="8" />
                  <path d="m21 21-4.35-4.35" />
                </svg>
                <input
                  id="filter-name"
                  type="text"
                  value={filterName}
                  onChange={(e) => setFilterName(e.target.value)}
                  placeholder="Search by name..."
                  className="search-input"
                />
                {filterName && (
                  <button
                    onClick={() => setFilterName('')}
                    className="clear-search"
                    aria-label="Clear search"
                  >
                    ×
                  </button>
                )}
              </div>
            </div>

            <div className="filter-field">
              <label htmlFor="view-mode">View Mode</label>
              <div className="view-toggle">
                <button
                  className={viewMode === 'cards' ? 'active' : ''}
                  onClick={() => setViewMode('cards')}
                  aria-label="Card view"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <rect x="3" y="3" width="7" height="7" />
                    <rect x="14" y="3" width="7" height="7" />
                    <rect x="14" y="14" width="7" height="7" />
                    <rect x="3" y="14" width="7" height="7" />
                  </svg>
                  Cards
                </button>
                <button
                  className={viewMode === 'table' ? 'active' : ''}
                  onClick={() => setViewMode('table')}
                  aria-label="Table view"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <line x1="3" y1="12" x2="21" y2="12" />
                    <line x1="3" y1="6" x2="21" y2="6" />
                    <line x1="3" y1="18" x2="21" y2="18" />
                  </svg>
                  Table
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Bulk Actions Bar (Sticky when items selected) */}
        {selectedAnimalIds.size > 0 && (
          <div className="bulk-actions-bar">
            <div className="bulk-actions-content">
              <div className="selection-info">
                <input
                  type="checkbox"
                  checked={selectedAnimalIds.size === animals.length && animals.length > 0}
                  onChange={handleSelectAll}
                  className="select-all-checkbox"
                />
                <span className="selected-count">
                  {selectedAnimalIds.size} {selectedAnimalIds.size === 1 ? 'animal' : 'animals'} selected
                </span>
              </div>

              <div className="bulk-actions-controls">
                <select
                  value={bulkAction}
                  onChange={(e) => setBulkAction(e.target.value)}
                  className="bulk-select"
                >
                  <option value="">Select Action</option>
                  <option value="move">Move to Group</option>
                  <option value="status">Change Status</option>
                </select>

                {bulkAction === 'move' && (
                  <select
                    value={bulkGroupId}
                    onChange={(e) => setBulkGroupId(e.target.value)}
                    className="bulk-select"
                  >
                    <option value="">Select Group</option>
                    {groups.map((group) => (
                      <option key={group.id} value={group.id}>
                        {group.name}
                      </option>
                    ))}
                  </select>
                )}

                {bulkAction === 'status' && (
                  <select
                    value={bulkStatus}
                    onChange={(e) => setBulkStatus(e.target.value)}
                    className="bulk-select"
                  >
                    <option value="">Select Status</option>
                    <option value="available">Available</option>
                    <option value="foster">Foster</option>
                    <option value="bite_quarantine">Bite Quarantine</option>
                    <option value="archived">Archived</option>
                  </select>
                )}

                <div className="bulk-action-buttons">
                  <button onClick={handleCancelBulk} className="btn-secondary">
                    Cancel
                  </button>
                  <button onClick={handleBulkUpdate} className="btn-primary">
                    Apply Changes
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Animals List */}
        {animals.length === 0 ? (
          <EmptyState
            icon={
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            }
            title="No animals found"
            description={
              filterStatus !== 'all' || filterGroup || filterName
                ? "No animals match your current filters. Try adjusting your search criteria."
                : "There are no animals in the system yet."
            }
            primaryAction={
              (filterStatus !== 'all' || filterGroup || filterName)
                ? {
                    label: 'Clear Filters',
                    onClick: clearFilters
                  }
                : undefined
            }
          />
        ) : viewMode === 'cards' ? (
          <div className="animals-grid">
            {animals.map((animal) => {
              const isSelected = selectedAnimalIds.has(animal.id);
              const isUpdating = updatingAnimal === animal.id;

              return (
                <div
                  key={animal.id}
                  className={`animal-card ${isSelected ? 'selected' : ''} ${isUpdating ? 'updating' : ''}`}
                >
                  <div className="animal-card-header">
                    <input
                      type="checkbox"
                      checked={isSelected}
                      onChange={() => handleSelectAnimal(animal.id)}
                      className="animal-checkbox"
                      disabled={isUpdating}
                    />
                    <span className={`status-badge status-${animal.status}`}>
                      {animal.status.replace('_', ' ')}
                    </span>
                  </div>

                  {animal.image_url && (
                    <div className="animal-card-image">
                      <img src={animal.image_url} alt={animal.name} />
                    </div>
                  )}

                  <div className="animal-card-body">
                    <Link
                      to={`/groups/${animal.group_id}/animals/${animal.id}/view`}
                      className="animal-name"
                    >
                      {animal.name}
                    </Link>
                    <div className="animal-meta">
                      <span>{animal.species || 'Unknown species'}</span>
                      {animal.breed && <span> • {animal.breed}</span>}
                      {animal.age && <span> • {animal.age} yrs</span>}
                    </div>

                    <div className="animal-info-grid">
                      <div className="info-item">
                        <div className="info-label">Group</div>
                        <select
                          value={animal.group_id}
                          onChange={(e) => handleIndividualGroupChange(animal.id, parseInt(e.target.value))}
                          disabled={isUpdating}
                          className="info-select"
                        >
                          {groups.map((group) => (
                            <option key={group.id} value={group.id}>
                              {group.name}
                            </option>
                          ))}
                        </select>
                      </div>

                      <div className="info-item">
                        <div className="info-label">Status</div>
                        <select
                          value={animal.status}
                          onChange={(e) => handleIndividualStatusChange(animal.id, e.target.value)}
                          disabled={isUpdating}
                          className="info-select"
                        >
                          <option value="available">Available</option>
                          <option value="foster">Foster</option>
                          <option value="bite_quarantine">Quarantine</option>
                          <option value="archived">Archived</option>
                        </select>
                      </div>

                      <div className="info-item">
                        <div className="info-label">Arrived</div>
                        <div className="info-value">{formatDateShort(animal.arrival_date)}</div>
                      </div>

                      <div className="info-item">
                        <div className="info-label">Length of Stay</div>
                        <div className="info-value">{calculateDaysSince(animal.arrival_date)} days</div>
                      </div>

                      {animal.status === 'bite_quarantine' && animal.quarantine_start_date && (
                        <>
                          <div className="info-item full-width">
                            <div className="info-label">Quarantine End</div>
                            <div className="info-value quarantine-date">
                              {calculateQuarantineEndDate(animal.quarantine_start_date)}
                            </div>
                          </div>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="animals-table-container">
            <table className="animals-table">
              <thead>
                <tr>
                  <th className="checkbox-col">
                    <input
                      type="checkbox"
                      checked={selectedAnimalIds.size === animals.length && animals.length > 0}
                      onChange={handleSelectAll}
                    />
                  </th>
                  <th>ID</th>
                  <th>Name</th>
                  <th>Species</th>
                  <th>Breed</th>
                  <th>Age</th>
                  <th>Status</th>
                  <th>Group</th>
                  <th>Arrival Date</th>
                  <th>Stay (days)</th>
                  <th>Status Duration</th>
                  <th>Quarantine End</th>
                </tr>
              </thead>
              <tbody>
                {animals.map((animal) => {
                  const isUpdating = updatingAnimal === animal.id;
                  return (
                    <tr key={animal.id} className={isUpdating ? 'updating' : ''}>
                      <td className="checkbox-col">
                        <input
                          type="checkbox"
                          checked={selectedAnimalIds.has(animal.id)}
                          onChange={() => handleSelectAnimal(animal.id)}
                          disabled={isUpdating}
                        />
                      </td>
                      <td>{animal.id}</td>
                      <td>
                        <Link
                          to={`/groups/${animal.group_id}/animals/${animal.id}/view`}
                          className="table-link"
                        >
                          {animal.name}
                        </Link>
                      </td>
                      <td>{animal.species || '-'}</td>
                      <td>{animal.breed || '-'}</td>
                      <td>{animal.age || '-'}</td>
                      <td>
                        <select
                          value={animal.status}
                          onChange={(e) => handleIndividualStatusChange(animal.id, e.target.value)}
                          disabled={isUpdating}
                          className="table-select"
                        >
                          <option value="available">Available</option>
                          <option value="foster">Foster</option>
                          <option value="bite_quarantine">Bite Quarantine</option>
                          <option value="archived">Archived</option>
                        </select>
                      </td>
                      <td>
                        <select
                          value={animal.group_id}
                          onChange={(e) => handleIndividualGroupChange(animal.id, parseInt(e.target.value))}
                          disabled={isUpdating}
                          className="table-select"
                        >
                          {groups.map((group) => (
                            <option key={group.id} value={group.id}>
                              {group.name}
                            </option>
                          ))}
                        </select>
                      </td>
                      <td>{formatDateShort(animal.arrival_date)}</td>
                      <td>{calculateDaysSince(animal.arrival_date)}</td>
                      <td>{calculateDaysSince(animal.last_status_change)}</td>
                      <td>
                        {animal.status === 'bite_quarantine'
                          ? calculateQuarantineEndDate(animal.quarantine_start_date)
                          : '-'
                        }
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Import Modal */}
      {showImportModal && (
        <Modal
          isOpen={showImportModal}
          title="Import Animals from CSV"
          onClose={() => {
            setShowImportModal(false);
            setImportResult(null);
          }}
        >
          <div className="import-modal-content">
            <p className="import-description">
              Upload a CSV file with the following columns:
            </p>
            <code className="csv-format">
              group_id, name, species, breed, age, description, status, image_url
            </code>
            <div className="import-note">
              <strong>Note:</strong> Only <code>group_id</code> and <code>name</code> are required.
            </div>

            <div className="file-upload-area">
              <input
                type="file"
                accept=".csv"
                onChange={(e) => setImportFile(e.target.files?.[0] || null)}
                id="csv-file-input"
                className="file-input"
              />
              <label htmlFor="csv-file-input" className="file-label">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="17 8 12 3 7 8" />
                  <line x1="12" y1="3" x2="12" y2="15" />
                </svg>
                {importFile ? importFile.name : 'Choose CSV file'}
              </label>
            </div>

            {importResult && (
              <div className="import-result">
                <div className="result-success">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                  {importResult.message}
                </div>
                {importResult.warnings && importResult.warnings.length > 0 && (
                  <div className="result-warnings">
                    <strong>Warnings:</strong>
                    <ul>
                      {importResult.warnings.map((warning, idx) => (
                        <li key={idx}>{warning}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}

            <div className="modal-actions">
              <button
                onClick={() => {
                  setShowImportModal(false);
                  setImportResult(null);
                }}
                className="btn-secondary"
              >
                Cancel
              </button>
              <button
                onClick={handleImportCSV}
                className="btn-primary"
                disabled={!importFile}
              >
                Import
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

export default BulkEditAnimalsPage;
