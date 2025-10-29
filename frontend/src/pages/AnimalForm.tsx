import React, { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { animalsApi } from '../api/client';
import './Form.css';

const AnimalForm: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    species: '',
    breed: '',
    age: 0,
    description: '',
    image_url: '',
    status: 'available',
  });

  useEffect(() => {
    if (id && groupId) {
      loadAnimal(parseInt(groupId), parseInt(id));
    }
  }, [id, groupId]);

  const loadAnimal = async (gId: number, animalId: number) => {
    try {
      const response = await animalsApi.getById(gId, animalId);
      setFormData(response.data);
    } catch (error) {
      console.error('Failed to load animal:', error);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      if (id && groupId) {
        await animalsApi.update(parseInt(groupId), parseInt(id), formData);
      } else if (groupId) {
        await animalsApi.create(parseInt(groupId), formData);
      }
      navigate(`/groups/${groupId}`);
    } catch (error: any) {
      alert(error.response?.data?.error || 'Failed to save animal');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm('Are you sure you want to delete this animal?')) {
      return;
    }
    try {
      if (id && groupId) {
        await animalsApi.delete(parseInt(groupId), parseInt(id));
        navigate(`/groups/${groupId}`);
      }
    } catch (error: any) {
      alert(error.response?.data?.error || 'Failed to delete animal');
    }
  };

  return (
    <div className="form-page">
      <div className="form-container">
        <Link to={`/groups/${groupId}`} className="back-link">
          ← Back to Group
        </Link>
        <h1>{id ? 'Edit Animal' : 'Add New Animal'}</h1>
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="name">Name *</label>
            <input
              id="name"
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>

          <div className="form-row">
            <div className="form-group">
              <label htmlFor="species">Species</label>
              <input
                id="species"
                type="text"
                value={formData.species}
                onChange={(e) => setFormData({ ...formData, species: e.target.value })}
              />
            </div>

            <div className="form-group">
              <label htmlFor="breed">Breed</label>
              <input
                id="breed"
                type="text"
                value={formData.breed}
                onChange={(e) => setFormData({ ...formData, breed: e.target.value })}
              />
            </div>
          </div>

          <div className="form-row">
            <div className="form-group">
              <label htmlFor="age">Age (years)</label>
              <input
                id="age"
                type="number"
                value={formData.age}
                onChange={(e) => setFormData({ ...formData, age: parseInt(e.target.value) })}
              />
            </div>

            <div className="form-group">
              <label htmlFor="status">Status</label>
              <select
                id="status"
                value={formData.status}
                onChange={(e) => setFormData({ ...formData, status: e.target.value })}
              >
                <option value="available">Available</option>
                <option value="adopted">Adopted</option>
                <option value="fostered">Fostered</option>
              </select>
            </div>
          </div>

          <div className="form-group">
            <label htmlFor="image_url">Image URL</label>
            <input
              id="image_url"
              type="text"
              value={formData.image_url}
              onChange={(e) => setFormData({ ...formData, image_url: e.target.value })}
              placeholder="Paste an image URL or upload a file"
            />
            <label htmlFor="image_upload">Or upload image</label>
            <input
              id="image_upload"
              type="file"
              accept=".jpg,.jpeg,.png,.gif"
              onChange={async (e) => {
                const file = e.target.files?.[0];
                if (!file) return;
                setLoading(true);
                try {
                  const res = await animalsApi.uploadImage(file);
                  setFormData({ ...formData, image_url: res.data.url });
                  alert('Image uploaded successfully!');
                } catch (err: any) {
                  console.error('Upload error:', err);
                  const errorMsg = err.response?.data?.error || err.message || 'Failed to upload image';
                  alert('Failed to upload image: ' + errorMsg);
                } finally {
                  setLoading(false);
                  // Clear the file input
                  e.target.value = '';
                }
              }}
            />
            {formData.image_url && (
              <div style={{ marginTop: '8px' }}>
                  <img src={formData.image_url} alt="Animal" style={{ width: 120, height: 120, objectFit: "cover", borderRadius: 8, border: "1px solid #ccc", marginTop: 8 }} />
              </div>
            )}
          </div>

          <div className="form-group">
            <label htmlFor="description">Description</label>
            <textarea
              id="description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              rows={4}
            />
          </div>

          <div className="form-actions">
            <button type="submit" className="btn-primary" disabled={loading}>
              {loading ? 'Saving...' : 'Save Animal'}
            </button>
            {id && (
              <button type="button" onClick={handleDelete} className="btn-danger">
                Delete
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
};

export default AnimalForm;
