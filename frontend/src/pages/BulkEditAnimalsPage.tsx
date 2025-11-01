import React, { useState, useEffect } from 'react';
import { animalsApi, groupsApi } from '../api/client';
import type { Animal, Group } from '../api/client';
import { useToast } from '../contexts/ToastContext';
import './BulkEditAnimalsPage.css';

// Helper function to calculate days since a date
const calculateDaysSince = (dateString?: string): number => {
  if (!dateString) return 0;
  const date = new Date(dateString);
  const now = new Date();
  const diffTime = Math.abs(now.getTime() - date.getTime());
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  return diffDays;
};

// Helper function to format a date
const formatDate = (dateString?: string): string => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
};

const BulkEditAnimalsPage: React.FC = () => {
  const toast = useToast();
  const [animals, setAnimals] = useState<Animal[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [loading, setLoading] = useState(true);
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

  useEffect(() => {
    loadData();
  }, [filterStatus, filterGroup, filterName]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [animalsRes, groupsRes] = await Promise.all([
        animalsApi.getAllForAdmin(
          filterStatus === 'all' ? undefined : filterStatus,
          filterGroup ? parseInt(filterGroup) : undefined,
          filterName || undefined
        ),
        groupsApi.getAll(),
      ]);
      setAnimals(animalsRes.data);
      setGroups(groupsRes.data);
    } catch (error: any) {
      console.error('Failed to load data:', error);
      const errorMsg = error.response?.data?.error || 'Failed to load animals. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setLoading(false);
    }
  };

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
      // Update local state
      setAnimals(animals.map(a => 
        a.id === animalId ? { ...a, status: newStatus } : a
      ));
      toast.showSuccess('Status updated successfully');
    } catch (error: any) {
      console.error('Failed to update status:', error);
      const errorMsg = error.response?.data?.error || 'Failed to update status. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUpdatingAnimal(null);
    }
  };

  const handleIndividualGroupChange = async (animalId: number, newGroupId: number) => {
    setUpdatingAnimal(animalId);
    try {
      await animalsApi.updateAnimal(animalId, { group_id: newGroupId });
      // Update local state
      setAnimals(animals.map(a => 
        a.id === animalId ? { ...a, group_id: newGroupId } : a
      ));
      toast.showSuccess('Group updated successfully');
    } catch (error: any) {
      console.error('Failed to update group:', error);
      const errorMsg = error.response?.data?.error || 'Failed to update group. Please try again.';
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
    } catch (error: any) {
      console.error('Failed to bulk update:', error);
      const errorMsg = error.response?.data?.error || 'Failed to update animals. Please try again.';
      toast.showError(errorMsg);
    }
  };

  const handleExportCSV = async () => {
    try {
      const response = await animalsApi.exportCSV(
        filterGroup ? parseInt(filterGroup) : undefined
      );
      
      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', 'animals.csv');
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      toast.showSuccess('Animals exported to CSV successfully!');
    } catch (error: any) {
      console.error('Failed to export CSV:', error);
      const errorMsg = error.response?.data?.error || 'Failed to export CSV. Please try again.';
      toast.showError(errorMsg);
    }
  };

  const handleExportCommentsCSV = async () => {
    try {
      const response = await animalsApi.exportCommentsCSV(
        filterGroup ? parseInt(filterGroup) : undefined
      );
      
      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', 'animal-comments.csv');
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      toast.showSuccess('Comments exported to CSV successfully!');
    } catch (error: any) {
      console.error('Failed to export comments CSV:', error);
      const errorMsg = error.response?.data?.error || 'Failed to export comments. Please try again.';
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
    } catch (error: any) {
      console.error('Failed to import CSV:', error);
      const errorMsg = error.response?.data?.error || 'Failed to import CSV. Please check the file format and try again.';
      toast.showError(errorMsg);
    }
  };

  return (
    <div className="bulk-edit-page">
      <div className="page-header">
        <h1>Bulk Edit Animals</h1>
        <div className="header-actions">
          <button onClick={handleExportCommentsCSV} className="btn-secondary">
            Export Comments
          </button>
          <button onClick={handleExportCSV} className="btn-secondary">
            Export Animals
          </button>
          <button onClick={() => setShowImportModal(true)} className="btn-primary">
            Import CSV
          </button>
        </div>
      </div>

      <div className="filters">
        <div className="filter-group">
          <label>Filter by Status:</label>
          <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)}>
            <option value="all">All</option>
            <option value="available">Available</option>
            <option value="foster">Foster</option>
            <option value="bite_quarantine">Bite Quarantine</option>
            <option value="archived">Archived</option>
          </select>
        </div>
        <div className="filter-group">
          <label>Filter by Group:</label>
          <select value={filterGroup} onChange={(e) => setFilterGroup(e.target.value)}>
            <option value="">All Groups</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.name}
              </option>
            ))}
          </select>
        </div>
        <div className="filter-group">
          <label>Search by Name:</label>
          <input
            type="text"
            value={filterName}
            onChange={(e) => setFilterName(e.target.value)}
            placeholder="Search..."
          />
        </div>
      </div>

      {selectedAnimalIds.size > 0 && (
        <div className="bulk-actions">
          <div className="selection-info">
            {selectedAnimalIds.size} animal(s) selected
          </div>
          <div className="action-controls">
            <select value={bulkAction} onChange={(e) => setBulkAction(e.target.value)}>
              <option value="">Select Action</option>
              <option value="move">Move to Group</option>
              <option value="status">Change Status</option>
            </select>

            {bulkAction === 'move' && (
              <select value={bulkGroupId} onChange={(e) => setBulkGroupId(e.target.value)}>
                <option value="">Select Group</option>
                {groups.map((group) => (
                  <option key={group.id} value={group.id}>
                    {group.name}
                  </option>
                ))}
              </select>
            )}

            {bulkAction === 'status' && (
              <select value={bulkStatus} onChange={(e) => setBulkStatus(e.target.value)}>
                <option value="">Select Status</option>
                <option value="available">Available</option>
                <option value="foster">Foster</option>
                <option value="bite_quarantine">Bite Quarantine</option>
                <option value="archived">Archived</option>
              </select>
            )}

            <button onClick={handleBulkUpdate} className="btn-primary">
              Apply
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <p>Loading...</p>
      ) : (
        <div className="animals-table-container">
          <table className="animals-table">
            <thead>
              <tr>
                <th>
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
                <th>Length of Stay (days)</th>
                <th>Current Status (days)</th>
              </tr>
            </thead>
            <tbody>
              {animals.length === 0 ? (
                <tr>
                  <td colSpan={11} className="no-data">
                    No animals found
                  </td>
                </tr>
              ) : (
                animals.map((animal) => (
                  <tr key={animal.id} className={updatingAnimal === animal.id ? 'updating' : ''}>
                    <td>
                      <input
                        type="checkbox"
                        checked={selectedAnimalIds.has(animal.id)}
                        onChange={() => handleSelectAnimal(animal.id)}
                      />
                    </td>
                    <td>{animal.id}</td>
                    <td>{animal.name}</td>
                    <td>{animal.species || '-'}</td>
                    <td>{animal.breed || '-'}</td>
                    <td>{animal.age || '-'}</td>
                    <td>
                      <select
                        value={animal.status}
                        onChange={(e) => handleIndividualStatusChange(animal.id, e.target.value)}
                        disabled={updatingAnimal === animal.id}
                        className="inline-select"
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
                        disabled={updatingAnimal === animal.id}
                        className="inline-select"
                      >
                        {groups.map((group) => (
                          <option key={group.id} value={group.id}>
                            {group.name}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td>{formatDate(animal.arrival_date)}</td>
                    <td>{calculateDaysSince(animal.arrival_date)}</td>
                    <td>{calculateDaysSince(animal.last_status_change)}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {showImportModal && (
        <div className="modal-overlay" onClick={() => setShowImportModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Import Animals from CSV</h2>
              <button onClick={() => setShowImportModal(false)} className="close-btn">
                ×
              </button>
            </div>
            <div className="modal-body">
              <p>
                Upload a CSV file with the following columns:
                <br />
                <code>group_id, name, species, breed, age, description, status, image_url</code>
              </p>
              <p className="note">
                <strong>Note:</strong> Only <code>group_id</code> and <code>name</code> are required.
              </p>
              <input
                type="file"
                accept=".csv"
                onChange={(e) => setImportFile(e.target.files?.[0] || null)}
              />
              {importResult && (
                <div className="import-result">
                  <p className="success">{importResult.message}</p>
                  {importResult.warnings && importResult.warnings.length > 0 && (
                    <div className="warnings">
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
            </div>
            <div className="modal-footer">
              <button onClick={() => setShowImportModal(false)} className="btn-secondary">
                Cancel
              </button>
              <button onClick={handleImportCSV} className="btn-primary" disabled={!importFile}>
                Import
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default BulkEditAnimalsPage;
