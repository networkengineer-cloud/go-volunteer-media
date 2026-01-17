import React, { useEffect, useState, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import { animalsApi } from '../api/client';
import type { Animal, AnimalImage } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import ImageEditor from '../components/ImageEditor';
import './PhotoGallery.css';

const PhotoGallery: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const { isAdmin, user } = useAuth();
  const { showToast } = useToast();
  const [animal, setAnimal] = useState<Animal | null>(null);
  const [images, setImages] = useState<AnimalImage[]>([]);
  const [deletedImages, setDeletedImages] = useState<AnimalImage[]>([]);
  const [showDeleted, setShowDeleted] = useState(false);
  const [loading, setLoading] = useState(true);
  const [selectedPhoto, setSelectedPhoto] = useState<AnimalImage | null>(null);
  const [lightboxIndex, setLightboxIndex] = useState<number>(-1);
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploadCaption, setUploadCaption] = useState('');
  const [uploading, setUploading] = useState(false);
  const [showImageEditor, setShowImageEditor] = useState(false);
  const [editingImageUrl, setEditingImageUrl] = useState<string>('');
  const [uploadPreviewUrl, setUploadPreviewUrl] = useState<string>('');
  const [previewWidth, setPreviewWidth] = useState<number | null>(null);
  const [previewHeight, setPreviewHeight] = useState<number | null>(null);
  const [settingProfile, setSettingProfile] = useState<number | null>(null);

  const loadData = useCallback(async (gId: number, animalId: number) => {
    try {
      const promises = [
        animalsApi.getById(gId, animalId),
        animalsApi.getImages(gId, animalId),
      ];

      if (isAdmin) {
        promises.push(animalsApi.getDeletedImages(gId));
      }

      const results = await Promise.all(promises);
      const [animalRes, imagesRes, deletedRes] = results;

      setAnimal(animalRes.data);
      setImages(imagesRes.data);

      if (isAdmin && deletedRes) {
        setDeletedImages(deletedRes.data.data || deletedRes.data);
      }
    } catch (error) {
      console.error('Failed to load data:', error);
      showToast('Failed to load photos', 'error');
    } finally {
      setLoading(false);
    }
  }, [isAdmin, showToast]);

  useEffect(() => {
    if (groupId && id) {
      loadData(Number(groupId), Number(id));
    }
  }, [groupId, id, loadData]);

  const handleDeleteImage = async (imageId: number) => {
    if (!groupId || !id) return;
    
    if (!window.confirm('Are you sure you want to delete this photo?')) {
      return;
    }

    try {
      await animalsApi.deleteImage(Number(groupId), Number(id), imageId);
      showToast('Photo deleted successfully', 'success');
      await loadData(Number(groupId), Number(id));
      closeLightbox();
    } catch (error) {
      console.error('Failed to delete image:', error);
      showToast('Failed to delete photo', 'error');
    }
  };

  const handleUpload = async () => {
    if (!uploadFile || !groupId || !id) return;

    setUploading(true);
    try {
      await animalsApi.uploadImageToGallery(Number(groupId), Number(id), uploadFile, uploadCaption);
      showToast('Photo uploaded successfully', 'success');
      setShowUploadModal(false);
      setUploadFile(null);
      setUploadCaption('');
      if (uploadPreviewUrl) {
        URL.revokeObjectURL(uploadPreviewUrl);
        setUploadPreviewUrl('');
        setPreviewWidth(null);
        setPreviewHeight(null);
      }
      loadData(Number(groupId), Number(id)); // Reload images
    } catch (error) {
      console.error('Upload failed:', error);
      showToast('Failed to upload photo', 'error');
    } finally {
      setUploading(false);
    }
  };

  const handleEditorSave = async (croppedBlob: Blob) => {
    setShowImageEditor(false);
    // Create a File object from the Blob with proper filename
    const fileName = `cropped-image-${Date.now()}.jpg`;
    const file = new File([croppedBlob], fileName, { type: 'image/jpeg' });
    setUploadFile(file);
    setShowUploadModal(true);
    const previewUrl = URL.createObjectURL(file);
    setUploadPreviewUrl(previewUrl);
    const img = new Image();
    img.onload = () => {
      setPreviewWidth(img.naturalWidth);
      setPreviewHeight(img.naturalHeight);
    };
    img.src = previewUrl;
    // Clean up preview URL
    if (editingImageUrl) {
      URL.revokeObjectURL(editingImageUrl);
      setEditingImageUrl('');
    }
  };

  const handleEditorCancel = () => {
    setShowImageEditor(false);
    setShowUploadModal(true);
    // Clean up preview URL
    if (editingImageUrl) {
      URL.revokeObjectURL(editingImageUrl);
      setEditingImageUrl('');
    }
  };

  const handleSetProfilePicture = async (imageId: number) => {
    if (!id || !groupId) return;

    // Store original state for rollback
    const originalImages = [...images];
    const originalSelectedPhoto = selectedPhoto;

    setSettingProfile(imageId);
    
    // Optimistic update - immediately update UI
    setImages(prev => prev.map(img => ({
      ...img,
      is_profile_picture: img.id === imageId
    })));
    
    if (selectedPhoto) {
      setSelectedPhoto(prev => prev ? { ...prev, is_profile_picture: prev.id === imageId } : null);
    }

    try {
      await animalsApi.setProfilePicture(Number(groupId), Number(id), imageId);
      showToast('✅ Profile picture updated!', 'success');
      
      // Reload to sync with server
      loadData(Number(groupId), Number(id));
    } catch (error) {
      console.error('Set profile picture failed:', error);
      // Rollback optimistic update on error
      setImages(originalImages);
      setSelectedPhoto(originalSelectedPhoto);
      showToast('Failed to set profile picture', 'error');
    } finally {
      setSettingProfile(null);
    }
  };

  const openLightbox = (index: number) => {
    setLightboxIndex(index);
    setSelectedPhoto(images[index]);
  };

  const closeLightbox = () => {
    setLightboxIndex(-1);
    setSelectedPhoto(null);
  };

  const nextPhoto = () => {
    const nextIndex = (lightboxIndex + 1) % images.length;
    setLightboxIndex(nextIndex);
    setSelectedPhoto(images[nextIndex]);
  };

  const prevPhoto = () => {
    const prevIndex = (lightboxIndex - 1 + images.length) % images.length;
    setLightboxIndex(prevIndex);
    setSelectedPhoto(images[prevIndex]);
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (lightboxIndex === -1) return;
    if (e.key === 'Escape') closeLightbox();
    if (e.key === 'ArrowRight') nextPhoto();
    if (e.key === 'ArrowLeft') prevPhoto();
  };

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lightboxIndex, images]);

  if (loading) {
    return <div className="loading">Loading photos...</div>;
  }

  if (!animal) {
    return <div className="error">Animal not found</div>;
  }

  return (
    <div className="photo-gallery-page">
      <div className="photo-gallery-container">
        <div className="gallery-header">
          <Link to={`/groups/${groupId}/animals/${id}/view`} className="back-link">
            ← Back to {animal.name}
          </Link>
          <h1>{animal.name}'s Photos</h1>
          <div className="gallery-header-actions">
            <p className="photo-count">
              {images.length} {images.length === 1 ? 'photo' : 'photos'}
            </p>
            <button className="btn-primary" onClick={() => setShowUploadModal(true)}>
              + Upload Photo
            </button>
          </div>
        </div>

        {isAdmin && deletedImages.length > 0 && (
          <div className="admin-section">
            <button
              className="btn-toggle-deleted"
              onClick={() => setShowDeleted(!showDeleted)}
            >
              {showDeleted ? '🔽 Hide' : '▶️ Show'} Deleted Photos ({deletedImages.length})
            </button>
          </div>
        )}

        {showDeleted && deletedImages.length > 0 && (
          <div className="deleted-photos-section">
            <h3>Deleted Photos (Admin View)</h3>
            <div className="photos-grid">
              {deletedImages.filter(img => img.animal_id === Number(id)).map((image) => (
                <div key={image.id} className="photo-card deleted-photo">
                  <div className="deleted-badge">🗑️ DELETED</div>
                  <img src={image.image_url} alt={image.caption || 'Deleted photo'} />
                  <div className="photo-info">
                    <span className="photo-author">{image.user?.username}</span>
                    <span className="photo-date">
                      {new Date(image.created_at).toLocaleDateString()}
                    </span>
                  </div>
                  {image.caption && (
                    <div className="photo-caption">{image.caption}</div>
                  )}
                  <div className="photo-meta">
                    Deleted: {new Date(image.deleted_at!).toLocaleDateString()}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {images.length === 0 ? (
          <div className="no-photos">
            <p>No photos have been shared for {animal.name} yet.</p>
            <button className="btn-primary" onClick={() => setShowUploadModal(true)}>
              Upload First Photo
            </button>
          </div>
        ) : (
          <div className="photos-grid">
            {images.map((image, index) => (
              <div
                key={image.id}
                className="photo-card"
                onClick={() => openLightbox(index)}
              >
                {image.is_profile_picture && (
                  <div className="profile-badge">Profile Picture</div>
                )}
                <img src={image.image_url} alt={image.caption || `Photo ${index + 1}`} />
                <div className="photo-info">
                  <span className="photo-author">{image.user?.username}</span>
                  <span className="photo-date">
                    {new Date(image.created_at).toLocaleDateString()}
                  </span>
                </div>
                {image.caption && (
                  <div className="photo-caption">{image.caption}</div>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Lightbox */}
        {selectedPhoto && lightboxIndex >= 0 && (
          <div className="lightbox" onClick={closeLightbox}>
            <div className="lightbox-content" onClick={(e) => e.stopPropagation()}>
              <button className="lightbox-close" onClick={closeLightbox}>
                ✕
              </button>
              <button
                className="lightbox-nav lightbox-prev"
                onClick={prevPhoto}
                disabled={images.length <= 1}
              >
                ‹
              </button>
              <div className="lightbox-image-container">
                <img src={selectedPhoto.image_url} alt={selectedPhoto.caption || 'Full size'} />
                <div className="lightbox-info">
                  {selectedPhoto.caption && (
                    <p className="lightbox-caption">{selectedPhoto.caption}</p>
                  )}
                  {selectedPhoto.is_profile_picture && (
                    <div className="profile-badge-large">Current Profile Picture</div>
                  )}
                  <div className="lightbox-meta">
                    <span>By {selectedPhoto.user?.username}</span>
                    <span>
                      {new Date(selectedPhoto.created_at).toLocaleDateString()} at{' '}
                      {new Date(selectedPhoto.created_at).toLocaleTimeString([], {
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </span>
                    <span>
                      {selectedPhoto.width} × {selectedPhoto.height} px
                    </span>
                  </div>
                  <div className="lightbox-actions">
                    {!selectedPhoto.is_profile_picture && (
                      <div className="profile-picture-action">
                        <button
                          className="btn-primary"
                          onClick={() => handleSetProfilePicture(selectedPhoto.id)}
                          disabled={settingProfile === selectedPhoto.id}
                          aria-label="Set this image as the animal's main profile picture"
                        >
                          {settingProfile === selectedPhoto.id ? (
                            <>⏳ Setting...</>
                          ) : (
                            <>⭐ Set as Profile Picture</>
                          )}
                        </button>
                        <p className="action-help-text">
                          This will become the main photo shown on {animal.name}'s card
                        </p>
                      </div>
                    )}
                    {(isAdmin || selectedPhoto.user?.id === user?.id) && (
                      <button
                        className="btn-danger"
                        onClick={() => handleDeleteImage(selectedPhoto.id)}
                      >
                        Delete Photo
                      </button>
                    )}
                  </div>
                  <p className="lightbox-counter">
                    {lightboxIndex + 1} / {images.length}
                  </p>
                </div>
              </div>
              <button
                className="lightbox-nav lightbox-next"
                onClick={nextPhoto}
                disabled={images.length <= 1}
              >
                ›
              </button>
            </div>
          </div>
        )}

        {/* Upload Modal */}
        {showUploadModal && (
          <div className="modal-overlay" onClick={() => setShowUploadModal(false)}>
            <div className="modal-content" onClick={(e) => e.stopPropagation()}>
              <div className="modal-header">
                <h2>Upload Photo</h2>
              </div>
              <div className="modal-body">
                <div className="form-field">
                  <label>Photo</label>
                  {uploadFile && uploadPreviewUrl ? (
                    <div className="image-preview" aria-live="polite">
                      <div className="image-preview-thumb">
                        <img src={uploadPreviewUrl} alt={uploadFile.name} />
                        <span className="image-preview-badge">Edited</span>
                      </div>
                      <div className="image-preview-meta">
                        <p className="image-name">{uploadFile.name}</p>
                        <p className="image-details">
                          {(uploadFile.size / 1024).toFixed(1)} KB
                          {previewWidth && previewHeight && (
                            <> · {previewWidth}×{previewHeight}px</>
                          )}
                        </p>
                        <div className="image-preview-actions">
                          <button
                            type="button"
                            className="btn-secondary small"
                            onClick={() => {
                              setUploadFile(null);
                              if (uploadPreviewUrl) {
                                URL.revokeObjectURL(uploadPreviewUrl);
                                setUploadPreviewUrl('');
                              }
                              setPreviewWidth(null);
                              setPreviewHeight(null);
                            }}
                          >
                            Replace Image
                          </button>
                          <button
                            type="button"
                            className="btn-secondary small"
                            onClick={() => {
                              if (uploadPreviewUrl) {
                                setEditingImageUrl(uploadPreviewUrl);
                                setShowImageEditor(true);
                                setShowUploadModal(false);
                              }
                            }}
                          >
                            Re-Edit
                          </button>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <input
                      type="file"
                      accept="image/*"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) {
                          if (!file.type.startsWith('image/')) {
                            showToast('Please select an image file', 'error');
                            return;
                          }
                          const previewUrl = URL.createObjectURL(file);
                          setEditingImageUrl(previewUrl);
                          setShowImageEditor(true);
                          setShowUploadModal(false);
                        }
                      }}
                    />
                  )}
                </div>
                <div className="form-field">
                  <label>Caption (optional)</label>
                  <textarea
                    value={uploadCaption}
                    onChange={(e) => setUploadCaption(e.target.value)}
                    placeholder="Add a caption for this photo..."
                    rows={3}
                  />
                </div>
              </div>
              <div className="modal-actions">
                <button
                  className="btn-secondary"
                  onClick={() => {
                    setShowUploadModal(false);
                    setUploadFile(null);
                    setUploadCaption('');
                    if (uploadPreviewUrl) {
                      URL.revokeObjectURL(uploadPreviewUrl);
                      setUploadPreviewUrl('');
                      setPreviewWidth(null);
                      setPreviewHeight(null);
                    }
                  }}
                  disabled={uploading}
                >
                  Cancel
                </button>
                <button
                  className="btn-primary"
                  onClick={handleUpload}
                  disabled={!uploadFile || uploading}
                >
                  {uploading ? 'Uploading...' : 'Upload'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Image Editor Modal */}
        {showImageEditor && editingImageUrl && (
          <ImageEditor
            imageSrc={editingImageUrl}
            onSave={handleEditorSave}
            onCancel={handleEditorCancel}
            aspectRatio={4 / 3}
          />
        )}
      </div>
    </div>
  );
};

export default PhotoGallery;
