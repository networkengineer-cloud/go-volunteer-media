import React, { useState, useCallback } from 'react';
import Cropper from 'react-easy-crop';
import type { Point, Area } from 'react-easy-crop';
import './ImageEditor.css';

interface ImageEditorProps {
  imageSrc: string;
  onSave: (croppedImage: Blob) => void;
  onCancel: () => void;
  aspectRatio?: number; // 1 for square, 4/3 for landscape, 0 for free
}

const ImageEditor: React.FC<ImageEditorProps> = ({
  imageSrc,
  onSave,
  onCancel,
  aspectRatio = 4 / 3,
}) => {
  const [crop, setCrop] = useState<Point>({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [rotation, setRotation] = useState(0);
  const [currentAspectRatio, setCurrentAspectRatio] = useState(aspectRatio);
  const [croppedAreaPixels, setCroppedAreaPixels] = useState<Area | null>(null);
  const [saving, setSaving] = useState(false);

  const onCropComplete = useCallback((_croppedArea: Area, croppedAreaPixels: Area) => {
    setCroppedAreaPixels(croppedAreaPixels);
  }, []);

  const createCroppedImage = async (): Promise<Blob | null> => {
    if (!croppedAreaPixels) return null;

    return new Promise((resolve, reject) => {
      const image = new Image();
      image.crossOrigin = 'anonymous';
      image.src = imageSrc;

      image.onload = () => {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');

        if (!ctx) {
          reject(new Error('Failed to get canvas context'));
          return;
        }

        const { width, height, x, y } = croppedAreaPixels;
        const radians = (rotation * Math.PI) / 180;

        // Calculate canvas size based on rotation
        let canvasWidth = width;
        let canvasHeight = height;
        
        if (rotation === 90 || rotation === 270) {
          canvasWidth = height;
          canvasHeight = width;
        }

        canvas.width = canvasWidth;
        canvas.height = canvasHeight;

        // Apply rotation
        ctx.save();
        ctx.translate(canvasWidth / 2, canvasHeight / 2);
        ctx.rotate(radians);
        ctx.translate(-width / 2, -height / 2);

        // Draw the cropped image
        ctx.drawImage(
          image,
          x,
          y,
          width,
          height,
          0,
          0,
          width,
          height
        );

        ctx.restore();

        // Convert to blob
        canvas.toBlob(
          (blob) => {
            if (blob) {
              resolve(blob);
            } else {
              reject(new Error('Failed to create blob'));
            }
          },
          'image/jpeg',
          0.9
        );
      };

      image.onerror = () => {
        reject(new Error('Failed to load image'));
      };
    });
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const croppedImage = await createCroppedImage();
      if (croppedImage) {
        onSave(croppedImage);
      }
    } catch (error) {
      console.error('Error cropping image:', error);
      alert('Failed to crop image. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="image-editor-overlay">
      <div className="image-editor-modal">
        <div className="image-editor-header">
          <h2>Edit Image</h2>
          <button 
            className="close-button" 
            onClick={onCancel}
            aria-label="Close editor"
          >
            ×
          </button>
        </div>

        <div className="image-editor-container">
          <Cropper
            image={imageSrc}
            crop={crop}
            zoom={zoom}
            rotation={rotation}
            aspect={currentAspectRatio || undefined}
            onCropChange={setCrop}
            onZoomChange={setZoom}
            onRotationChange={setRotation}
            onCropComplete={onCropComplete}
            objectFit="contain"
          />
        </div>

        <div className="image-editor-controls">
          <div className="control-group">
            <label>Zoom</label>
            <div className="control-row">
              <span className="control-icon">🔍</span>
              <input
                type="range"
                min={1}
                max={3}
                step={0.1}
                value={zoom}
                onChange={(e) => setZoom(Number(e.target.value))}
                className="control-slider"
              />
              <span className="control-value">{zoom.toFixed(1)}x</span>
            </div>
          </div>

          <div className="control-group">
            <label>Rotation</label>
            <div className="control-row">
              <span className="control-icon">🔄</span>
              <input
                type="range"
                min={0}
                max={360}
                step={1}
                value={rotation}
                onChange={(e) => setRotation(Number(e.target.value))}
                className="control-slider"
              />
              <span className="control-value">{rotation}°</span>
            </div>
            <div className="rotation-shortcuts">
              <button 
                type="button"
                onClick={() => setRotation(0)} 
                className={rotation === 0 ? 'active' : ''}
              >
                0°
              </button>
              <button 
                type="button"
                onClick={() => setRotation(90)}
                className={rotation === 90 ? 'active' : ''}
              >
                90°
              </button>
              <button 
                type="button"
                onClick={() => setRotation(180)}
                className={rotation === 180 ? 'active' : ''}
              >
                180°
              </button>
              <button 
                type="button"
                onClick={() => setRotation(270)}
                className={rotation === 270 ? 'active' : ''}
              >
                270°
              </button>
            </div>
          </div>

          <div className="control-group">
            <label>Aspect Ratio</label>
            <div className="aspect-ratio-buttons">
              <button
                type="button"
                onClick={() => setCurrentAspectRatio(1)}
                className={currentAspectRatio === 1 ? 'active' : ''}
              >
                Square (1:1)
              </button>
              <button
                type="button"
                onClick={() => setCurrentAspectRatio(4 / 3)}
                className={currentAspectRatio === 4 / 3 ? 'active' : ''}
              >
                Landscape (4:3)
              </button>
              <button
                type="button"
                onClick={() => setCurrentAspectRatio(16 / 9)}
                className={currentAspectRatio === 16 / 9 ? 'active' : ''}
              >
                Wide (16:9)
              </button>
              <button
                type="button"
                onClick={() => setCurrentAspectRatio(0)}
                className={currentAspectRatio === 0 ? 'active' : ''}
              >
                Free
              </button>
            </div>
          </div>
        </div>

        <div className="image-editor-actions">
          <button 
            type="button"
            onClick={onCancel} 
            className="btn-secondary"
            disabled={saving}
          >
            Cancel
          </button>
          <button 
            type="button"
            onClick={handleSave} 
            className="btn-primary"
            disabled={saving}
          >
            {saving ? 'Saving...' : 'Save & Upload'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ImageEditor;
