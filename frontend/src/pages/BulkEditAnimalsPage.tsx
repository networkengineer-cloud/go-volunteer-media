import React, { useState, useEffect } from 'react';
import { animalsApi, groupsApi } from '../api/client';
import type { Animal, Group } from '../api/client';
import './BulkEditAnimalsPage.css';

const BulkEditAnimalsPage: React.FC = () => {
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
    } catch (error) {
      console.error('Failed to load data:', error);
      alert('Failed to load animals');
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

  const handleBulkUpdate = async () => {
    if (selectedAnimalIds.size === 0) {
      alert('Please select at least one animal');
      return;
    }

    if (bulkAction === '') {
      alert('Please select an action');
      return;
    }

    if (bulkAction === 'move' && !bulkGroupId) {
      alert('Please select a target group');
      return;
    }

    if (bulkAction === 'status' && !bulkStatus) {
      alert('Please select a status');
      return;
    }

    try {
      const animalIds = Array.from(selectedAnimalIds);
      const groupId = bulkAction === 'move' ? parseInt(bulkGroupId) : undefined;
      const status = bulkAction === 'status' ? bulkStatus : undefined;

      const response = await animalsApi.bulkUpdate(animalIds, groupId, status);
      alert(response.data.message);
      setSelectedAnimalIds(new Set());
      setBulkAction('');
      setBulkGroupId('');
      setBulkStatus('');
      loadData();
    } catch (error: any) {
      console.error('Failed to bulk update:', error);
      alert(error.response?.data?.error || 'Failed to update animals');
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
    } catch (error) {
      console.error('Failed to export CSV:', error);
      alert('Failed to export CSV');
    }
  };

  const handleImportCSV = async () => {
    if (!importFile) {
      alert('Please select a CSV file');
      return;
    }

    try {
      const response = await animalsApi.importCSV(importFile);
      setImportResult(response.data);
      setImportFile(null);
      loadData();
    } catch (error: any) {
      console.error('Failed to import CSV:', error);
      alert(error.response?.data?.error || 'Failed to import CSV');
    }
  };

  const getGroupName = (groupId: number) => {
    const group = groups.find((g) => g.id === groupId);
    return group ? group.name : `Group ${groupId}`;
  };

  return (
    <div className="bulk-edit-page">
      <div className="page-header">
        <h1>Bulk Edit Animals</h1>
        <div className="header-actions">
          <button onClick={handleExportCSV} className="btn-secondary">
            Export CSV
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
            <option value="adopted">Adopted</option>
            <option value="fostered">Fostered</option>
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
                <option value="adopted">Adopted</option>
                <option value="fostered">Fostered</option>
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
              </tr>
            </thead>
            <tbody>
              {animals.length === 0 ? (
                <tr>
                  <td colSpan={8} className="no-data">
                    No animals found
                  </td>
                </tr>
              ) : (
                animals.map((animal) => (
                  <tr key={animal.id}>
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
                      <span className={`status-badge status-${animal.status}`}>
                        {animal.status}
                      </span>
                    </td>
                    <td>{getGroupName(animal.group_id)}</td>
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
