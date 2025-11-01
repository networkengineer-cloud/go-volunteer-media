import React, { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { updatesApi, animalsApi } from '../api/client';
import { useToast } from '../contexts/ToastContext';
import FormField from '../components/FormField';
import Button from '../components/Button';
import './Form.css';

const UpdateForm: React.FC = () => {
  const { groupId } = useParams<{ groupId: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [formData, setFormData] = useState({
    title: '',
    content: '',
    image_url: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  const validateField = (name: string, value: string) => {
    switch (name) {
      case 'title':
        if (!value || value.trim().length === 0) {
          return 'Title is required';
        }
        if (value.length < 3) {
          return 'Title must be at least 3 characters';
        }
        if (value.length > 200) {
          return 'Title must be less than 200 characters';
        }
        return '';
      case 'content':
        if (!value || value.trim().length === 0) {
          return 'Content is required';
        }
        if (value.length < 10) {
          return 'Content must be at least 10 characters';
        }
        return '';
      default:
        return '';
    }
  };

  const handleBlur = (field: string) => {
    setTouched({ ...touched, [field]: true });
    const error = validateField(field, formData[field as keyof typeof formData] as string);
    setErrors({ ...errors, [field]: error });
  };

  const handleFieldChange = (field: string, value: string) => {
    setFormData({ ...formData, [field]: value });
    if (touched[field]) {
      const error = validateField(field, value);
      setErrors({ ...errors, [field]: error });
    }
  };

  const isFormValid = () => {
    return formData.title.trim().length >= 3 && 
           formData.content.trim().length >= 10 &&
           !errors.title && 
           !errors.content;
  };

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    if (!file.type.startsWith('image/')) {
      toast.showError('Please select an image file (JPG, PNG, or GIF)');
      return;
    }

    // Validate file size (max 10MB)
    if (file.size > 10 * 1024 * 1024) {
      toast.showError(`Image size must be under 10MB. Your file is ${(file.size / 1024 / 1024).toFixed(1)}MB`);
      return;
    }

    setUploading(true);
    try {
      const res = await animalsApi.uploadImage(file);
      setFormData({ ...formData, image_url: res.data.url });
      toast.showSuccess('Image uploaded successfully!');
    } catch (err: any) {
      console.error('Upload error:', err);
      const errorMsg = err.response?.data?.error || 'Failed to upload image. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUploading(false);
      // Clear the file input
      if (e.target) e.target.value = '';
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate all fields
    const titleError = validateField('title', formData.title);
    const contentError = validateField('content', formData.content);
    
    if (titleError || contentError) {
      setErrors({ title: titleError, content: contentError });
      setTouched({ title: true, content: true });
      toast.showError('Please fix the errors in the form');
      return;
    }

    setLoading(true);
    try {
      if (groupId) {
        await updatesApi.create(
          parseInt(groupId),
          formData.title,
          formData.content,
          formData.image_url
        );
        toast.showSuccess('Announcement created successfully!');
        navigate(`/groups/${groupId}`);
      }
    } catch (error: any) {
      const errorMsg = error.response?.data?.error || 'Failed to create announcement. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="form-page">
      <div className="form-container">
        <Link to={`/groups/${groupId}`} className="back-link">
          ← Back to Group
        </Link>
        <h1>New Announcement</h1>
        <form onSubmit={handleSubmit}>
          <FormField
            label="Title"
            id="title"
            type="text"
            value={formData.title}
            onChange={(value) => handleFieldChange('title', value)}
            onBlur={() => handleBlur('title')}
            error={touched.title ? errors.title : ''}
            success={touched.title && !errors.title && formData.title.length >= 3}
            required
            helperText="A brief, descriptive title for your announcement"
            autoComplete="off"
            minLength={3}
            maxLength={200}
          />

          <FormField
            label="Content"
            id="content"
            type="textarea"
            value={formData.content}
            onChange={(value) => handleFieldChange('content', value)}
            onBlur={() => handleBlur('content')}
            error={touched.content ? errors.content : ''}
            success={touched.content && !errors.content && formData.content.length >= 10}
            required
            rows={8}
            helperText="Share important information, updates, or news with your group"
            maxLength={5000}
          />

          <div className="form-field">
            <label htmlFor="image_upload" className="form-field__label">
              Announcement Image (Optional)
            </label>
            <input
              id="image_upload"
              type="file"
              accept="image/jpeg,image/png,image/gif"
              onChange={handleImageUpload}
              className="form-field__input"
              disabled={uploading}
              aria-label="Upload announcement image"
            />
            <p className="form-field__helper">
              Add a photo to your announcement (JPG, PNG, or GIF, max 10MB)
            </p>
            {uploading && (
              <p className="form-field__helper" style={{ color: 'var(--color-primary)' }}>
                Uploading image...
              </p>
            )}
            {formData.image_url && (
              <div className="image-preview">
                <label>Preview:</label>
                <img src={formData.image_url} alt="Announcement Preview" />
              </div>
            )}
          </div>

          <div className="form-actions">
            <Button
              type="submit"
              variant="primary"
              loading={loading}
              disabled={!isFormValid() || uploading}
              fullWidth={false}
            >
              Create Announcement
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => navigate(`/groups/${groupId}`)}
              disabled={loading}
            >
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default UpdateForm;
