import React, { useState, useEffect, useRef } from 'react';
import { animalsApi } from '../api/client';
import Button from './Button';
import './VideoUpload.css';

const MAX_VIDEO_SIZE = 200 * 1024 * 1024;
const ALLOWED_TYPES = ['video/mp4', 'video/quicktime'];

interface VideoUploadProps {
  groupId: number;
  animalId: number;
  onSuccess: () => void;
  onCancel: () => void;
  preselectedFile?: File;
}

const VideoUpload: React.FC<VideoUploadProps> = ({ groupId, animalId, onSuccess, onCancel, preselectedFile }) => {
  const [videoFile, setVideoFile] = useState<File | null>(preselectedFile ?? null);
  const [caption, setCaption] = useState('');
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [thumbnailUrl, setThumbnailUrl] = useState<string | null>(null);
  const thumbnailPromiseRef = useRef<Promise<{ blob: Blob; duration: number }> | null>(null);
  const thumbnailObjectUrlRef = useRef<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    return () => {
      if (thumbnailObjectUrlRef.current) {
        URL.revokeObjectURL(thumbnailObjectUrlRef.current);
      }
    };
  }, []);

  const applyThumbnailPromise = (promise: Promise<{ blob: Blob; duration: number }>) => {
    promise
      .then(({ blob }) => {
        const url = URL.createObjectURL(blob);
        thumbnailObjectUrlRef.current = url;
        setThumbnailUrl(url);
      })
      .catch(() => {});
  };

  useEffect(() => {
    if (!preselectedFile) return;
    if (!ALLOWED_TYPES.includes(preselectedFile.type)) {
      setError('Only MP4 and MOV videos are supported.');
      setVideoFile(null);
    } else if (preselectedFile.size > MAX_VIDEO_SIZE) {
      setError('This video is too large. Please use a clip under 200MB.');
      setVideoFile(null);
    } else {
      const promise = extractThumbnail(preselectedFile);
      thumbnailPromiseRef.current = promise;
      applyThumbnailPromise(promise);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [preselectedFile]);

  const extractThumbnail = (file: File): Promise<{ blob: Blob; duration: number }> =>
    new Promise((resolve, reject) => {
      const video = document.createElement('video');
      video.preload = 'auto';
      video.muted = true;
      video.playsInline = true;
      const objectUrl = URL.createObjectURL(file);

      const capture = () => {
        if (video.videoWidth === 0 || video.videoHeight === 0) {
          URL.revokeObjectURL(objectUrl);
          reject(new Error('Video dimensions unavailable'));
          return;
        }
        const MAX_THUMB_WIDTH = 640;
        const scale = Math.min(1, MAX_THUMB_WIDTH / video.videoWidth);
        const canvas = document.createElement('canvas');
        canvas.width = Math.round(video.videoWidth * scale);
        canvas.height = Math.round(video.videoHeight * scale);
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          URL.revokeObjectURL(objectUrl);
          reject(new Error('Canvas context unavailable'));
          return;
        }
        ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
        canvas.toBlob(
          (blob) => {
            URL.revokeObjectURL(objectUrl);
            if (!blob) {
              reject(new Error('Failed to generate thumbnail'));
              return;
            }
            resolve({ blob, duration: Math.round(video.duration) });
          },
          'image/jpeg',
          0.85,
        );
      };

      video.onloadedmetadata = () => {
        let settled = false;
        const done = () => {
          if (!settled) {
            settled = true;
            capture();
          }
        };

        video.onseeked = () => {
          video.onseeked = null;
          done();
        };

        video.currentTime = isFinite(video.duration) ? Math.min(0.5, video.duration) : 0;

        setTimeout(() => {
          video.onseeked = null;
          if (!settled) {
            settled = true;
            URL.revokeObjectURL(objectUrl);
            reject(new Error('Video seek timed out'));
          }
        }, 5000);
      };

      video.onerror = () => {
        video.onseeked = null;
        URL.revokeObjectURL(objectUrl);
        reject(new Error('Failed to load video'));
      };

      video.src = objectUrl;
      video.load();
    });

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!ALLOWED_TYPES.includes(file.type)) {
      setError('Only MP4 and MOV videos are supported.');
      return;
    }
    if (file.size > MAX_VIDEO_SIZE) {
      setError('This video is too large. Please use a clip under 200MB.');
      return;
    }
    setError(null);
    setVideoFile(file);
    if (thumbnailObjectUrlRef.current) {
      URL.revokeObjectURL(thumbnailObjectUrlRef.current);
      thumbnailObjectUrlRef.current = null;
    }
    setThumbnailUrl(null);
    const promise = extractThumbnail(file);
    thumbnailPromiseRef.current = promise;
    applyThumbnailPromise(promise);
  };

  const clearSelection = () => {
    setVideoFile(null);
    setError(null);
    setThumbnailUrl(null);
    thumbnailPromiseRef.current = null;
    if (thumbnailObjectUrlRef.current) {
      URL.revokeObjectURL(thumbnailObjectUrlRef.current);
      thumbnailObjectUrlRef.current = null;
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleUpload = async () => {
    if (!videoFile) return;
    setUploading(true);
    setError(null);

    let thumbnailBlob: Blob;
    let duration: number;
    try {
      ({ blob: thumbnailBlob, duration } = await (thumbnailPromiseRef.current ?? extractThumbnail(videoFile)));
    } catch {
      setError("Couldn't generate a preview for this video. Please try a different file.");
      setUploading(false);
      return;
    }

    try {
      await animalsApi.uploadVideo(groupId, animalId, videoFile, thumbnailBlob, caption, duration);
      onSuccess();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || 'Upload failed. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  const fileSizeMB = videoFile ? (videoFile.size / 1024 / 1024).toFixed(1) : '';

  return (
    <div className="video-upload">
      <input
        ref={fileInputRef}
        type="file"
        accept="video/mp4,video/quicktime,.mp4,.mov"
        onChange={handleFileSelect}
        disabled={uploading}
        style={{ display: 'none' }}
        aria-hidden="true"
      />

      {!videoFile ? (
        <div
          className="video-upload__dropzone"
          role="button"
          tabIndex={0}
          aria-label="Choose a video file"
          onClick={() => fileInputRef.current?.click()}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') fileInputRef.current?.click();
          }}
        >
          <div className="video-upload__dropzone-icon" aria-hidden="true">🎬</div>
          <div className="video-upload__dropzone-label">Tap to choose a video</div>
          <div className="video-upload__dropzone-hint">MP4 or MOV · up to 200 MB</div>
        </div>
      ) : (
        <div
          className="video-upload__thumbnail"
          role="button"
          tabIndex={0}
          aria-label="Change video"
          onClick={() => fileInputRef.current?.click()}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') fileInputRef.current?.click();
          }}
        >
          <div
            className="video-upload__thumbnail-bg"
            style={thumbnailUrl ? { backgroundImage: `url(${thumbnailUrl})` } : undefined}
          />
          <div className="video-upload__thumbnail-overlay" />
          <div className="video-upload__thumbnail-play">
            <div className="video-upload__thumbnail-play-circle">▶</div>
          </div>
          <div className="video-upload__thumbnail-meta">
            {videoFile.name} · {fileSizeMB} MB
          </div>
          <button
            type="button"
            className="video-upload__thumbnail-repick"
            aria-label="Remove selected video"
            onClick={(e) => {
              e.stopPropagation();
              clearSelection();
            }}
          >
            ✕
          </button>
        </div>
      )}

      <input
        type="text"
        className="video-upload__caption"
        placeholder="Caption (optional)"
        value={caption}
        onChange={(e) => setCaption(e.target.value)}
        disabled={!videoFile || uploading}
      />

      {error && (
        <p className="video-upload__error" role="alert">
          {error}
        </p>
      )}

      <div className="modal__actions">
        <Button variant="secondary" onClick={onCancel} disabled={uploading}>
          Cancel
        </Button>
        <Button onClick={handleUpload} disabled={!videoFile} loading={uploading}>
          {uploading ? 'Uploading…' : 'Upload Video'}
        </Button>
      </div>
    </div>
  );
};

export default VideoUpload;
