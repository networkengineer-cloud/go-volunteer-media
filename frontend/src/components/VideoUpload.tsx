import React, { useState, useEffect } from 'react';
import { animalsApi } from '../api/client';

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

  useEffect(() => {
    if (!preselectedFile) return;
    if (!ALLOWED_TYPES.includes(preselectedFile.type)) {
      setError('Only MP4 and MOV videos are supported.');
      setVideoFile(null);
    } else if (preselectedFile.size > MAX_VIDEO_SIZE) {
      setError('This video is too large. Please use a clip under 200MB.');
      setVideoFile(null);
    }
  }, [preselectedFile]);

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
  };

  const extractThumbnail = (file: File): Promise<{ blob: Blob; duration: number }> =>
    new Promise((resolve, reject) => {
      const video = document.createElement('video');
      // 'auto' tells iOS Safari to buffer frame data, not just metadata.
      // Without this, videoWidth/videoHeight are 0 at capture time on iOS.
      video.preload = 'auto';
      video.muted = true;
      video.playsInline = true;
      const objectUrl = URL.createObjectURL(file);

      const capture = () => {
        const canvas = document.createElement('canvas');
        canvas.width = video.videoWidth;
        canvas.height = video.videoHeight;
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          URL.revokeObjectURL(objectUrl);
          reject(new Error('Canvas context unavailable'));
          return;
        }
        ctx.drawImage(video, 0, 0);
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

      // Use loadedmetadata + explicit seek so iOS Safari decodes an actual frame
      // before canvas capture. loadeddata with currentTime=0 is unreliable on iOS
      // because frame 0 may not be decoded until a seek completes.
      video.onloadedmetadata = () => {
        video.onseeked = () => {
          video.onseeked = null;
          capture();
        };
        // Seek to 0.5 s rather than 0: iOS HEVC streams may not have a decodable
        // keyframe at exactly t=0, causing canvas.toBlob() to return null.
        video.currentTime = Math.min(0.5, video.duration);
      };

      video.onerror = () => {
        video.onseeked = null;
        URL.revokeObjectURL(objectUrl);
        reject(new Error('Failed to load video'));
      };

      video.src = objectUrl;
      video.load();
    });

  const handleUpload = async () => {
    if (!videoFile) return;
    setUploading(true);
    setError(null);

    let thumbnailBlob: Blob;
    let duration: number;
    try {
      ({ blob: thumbnailBlob, duration } = await extractThumbnail(videoFile));
    } catch {
      setError("Couldn't generate a preview for this video. Please try a different file.");
      setUploading(false);
      return;
    }

    try {
      await animalsApi.uploadVideo(groupId, animalId, videoFile, thumbnailBlob, caption, duration);
      onSuccess();
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || 'Upload failed. Please try again.');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="video-upload">
      <h3>Upload Video</h3>
      <input
        type="file"
        accept="video/mp4,video/quicktime,.mp4,.mov"
        onChange={handleFileSelect}
        disabled={uploading}
      />
      {videoFile && (
        <p className="file-selected">{videoFile.name} ({(videoFile.size / 1024 / 1024).toFixed(1)} MB)</p>
      )}
      <input
        type="text"
        placeholder="Caption (optional)"
        value={caption}
        onChange={(e) => setCaption(e.target.value)}
        disabled={uploading}
      />
      {error && <p className="upload-error">{error}</p>}
      <div className="upload-actions">
        <button onClick={onCancel} disabled={uploading} className="btn-secondary">
          Cancel
        </button>
        <button onClick={handleUpload} disabled={!videoFile || uploading} className="btn-primary">
          {uploading ? 'Uploading…' : 'Upload Video'}
        </button>
      </div>
    </div>
  );
};

export default VideoUpload;
