import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { animalsApi, animalCommentsApi } from '../api/client';
import type { Animal, AnimalComment } from '../api/client';
import './PhotoGallery.css';

const PhotoGallery: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const [animal, setAnimal] = useState<Animal | null>(null);
  const [photos, setPhotos] = useState<AnimalComment[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPhoto, setSelectedPhoto] = useState<AnimalComment | null>(null);
  const [lightboxIndex, setLightboxIndex] = useState<number>(-1);

  useEffect(() => {
    if (groupId && id) {
      loadData(Number(groupId), Number(id));
    }
  }, [groupId, id]);

  const loadData = async (gId: number, animalId: number) => {
    try {
      const [animalRes, commentsRes] = await Promise.all([
        animalsApi.getById(gId, animalId),
        animalCommentsApi.getAll(gId, animalId),
      ]);
      
      setAnimal(animalRes.data);
      
      // Filter comments that have images
      const commentsWithImages = commentsRes.data.filter(
        (comment) => comment.image_url && comment.image_url.trim() !== ''
      );
      setPhotos(commentsWithImages);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const openLightbox = (index: number) => {
    setLightboxIndex(index);
    setSelectedPhoto(photos[index]);
  };

  const closeLightbox = () => {
    setLightboxIndex(-1);
    setSelectedPhoto(null);
  };

  const nextPhoto = () => {
    const nextIndex = (lightboxIndex + 1) % photos.length;
    setLightboxIndex(nextIndex);
    setSelectedPhoto(photos[nextIndex]);
  };

  const prevPhoto = () => {
    const prevIndex = (lightboxIndex - 1 + photos.length) % photos.length;
    setLightboxIndex(prevIndex);
    setSelectedPhoto(photos[prevIndex]);
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
  }, [lightboxIndex, photos]);

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
          <p className="photo-count">
            {photos.length} {photos.length === 1 ? 'photo' : 'photos'}
          </p>
        </div>

        {photos.length === 0 ? (
          <div className="no-photos">
            <p>No photos have been shared for {animal.name} yet.</p>
            <Link to={`/groups/${groupId}/animals/${id}/view`} className="btn-primary">
              Add a Photo
            </Link>
          </div>
        ) : (
          <div className="photos-grid">
            {photos.map((photo, index) => (
              <div
                key={photo.id}
                className="photo-card"
                onClick={() => openLightbox(index)}
              >
                <img src={photo.image_url} alt={`Photo ${index + 1}`} />
                <div className="photo-info">
                  <span className="photo-author">{photo.user?.username}</span>
                  <span className="photo-date">
                    {new Date(photo.created_at).toLocaleDateString()}
                  </span>
                </div>
                {photo.tags && photo.tags.length > 0 && (
                  <div className="photo-tags">
                    {photo.tags.map((tag) => (
                      <span
                        key={tag.id}
                        className="tag-badge"
                        style={{ backgroundColor: tag.color }}
                      >
                        {tag.name}
                      </span>
                    ))}
                  </div>
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
                disabled={photos.length <= 1}
              >
                ‹
              </button>
              <div className="lightbox-image-container">
                <img src={selectedPhoto.image_url} alt="Full size" />
                <div className="lightbox-info">
                  {selectedPhoto.content && (
                    <p className="lightbox-caption">{selectedPhoto.content}</p>
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
                  </div>
                  {selectedPhoto.tags && selectedPhoto.tags.length > 0 && (
                    <div className="lightbox-tags">
                      {selectedPhoto.tags.map((tag) => (
                        <span
                          key={tag.id}
                          className="tag-badge"
                          style={{ backgroundColor: tag.color }}
                        >
                          {tag.name}
                        </span>
                      ))}
                    </div>
                  )}
                  <p className="lightbox-counter">
                    {lightboxIndex + 1} / {photos.length}
                  </p>
                </div>
              </div>
              <button
                className="lightbox-nav lightbox-next"
                onClick={nextPhoto}
                disabled={photos.length <= 1}
              >
                ›
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default PhotoGallery;
