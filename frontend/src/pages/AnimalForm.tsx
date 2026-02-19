import React, { useEffect, useState, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { animalsApi, updatesApi, animalTagsApi, commentTagsApi, animalCommentsApi } from '../api/client';
import type { AnimalTag, Animal, DuplicateNameInfo, AnimalImage } from '../api/client';
import { useToast } from '../hooks/useToast';
import { calculateQuarantineEndDate } from '../utils/dateUtils';
import FormField from '../components/FormField';
import Button from '../components/Button';
import Modal from '../components/Modal';
import ImageEditor from '../components/ImageEditor';
import './Form.css';

const AnimalForm: React.FC = () => {
  const { groupId, id } = useParams<{ groupId: string; id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showQuarantineModal, setShowQuarantineModal] = useState(false);
  const [showUnarchiveModal, setShowUnarchiveModal] = useState(false);
  const [showImageSelector, setShowImageSelector] = useState(false);
  const [availableImages, setAvailableImages] = useState<AnimalImage[]>([]);
  const [loadingImages, setLoadingImages] = useState(false);
  const [showImageEditor, setShowImageEditor] = useState(false);
  const [editingImageUrl, setEditingImageUrl] = useState<string>('');
  const [pendingStatusChange, setPendingStatusChange] = useState<string>('');
  const [unarchiveIsReturned, setUnarchiveIsReturned] = useState(false);
  const [quarantineContext, setQuarantineContext] = useState('');
  const [quarantineDate, setQuarantineDate] = useState('');
  const [originalStatus, setOriginalStatus] = useState('');
  const [availableTags, setAvailableTags] = useState<AnimalTag[]>([]);
  const [selectedTagIds, setSelectedTagIds] = useState<number[]>([]);
  const [duplicateInfo, setDuplicateInfo] = useState<DuplicateNameInfo | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    species: '',
    breed: '',
    age: 0,
    description: '',
    image_url: '',
    status: 'available',
    arrival_date: '',
    quarantine_start_date: '',
    is_returned: false,
    protocol_document_url: '',
    protocol_document_name: '',
  });
  const [protocolDocumentFile, setProtocolDocumentFile] = useState<File | null>(null);
  const [uploadingDocument, setUploadingDocument] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  const loadAvailableTags = useCallback(async () => {
    if (!groupId) return;
    try {
      const response = await animalTagsApi.getAll(parseInt(groupId));
      setAvailableTags(response.data);
    } catch (error) {
      console.error('Failed to load tags:', error);
    }
  }, [groupId]);

  // Memoize modal close handlers to prevent unnecessary re-renders
  const handleQuarantineModalClose = useCallback(() => {
    setShowQuarantineModal(false);
    setQuarantineContext('');
  }, []);

  const handleUnarchiveModalClose = useCallback(() => {
    setShowUnarchiveModal(false);
    setPendingStatusChange('');
  }, []);

  const checkForDuplicateNames = useCallback(async (name: string, currentAnimalId?: number) => {
    if (!groupId || !name.trim()) {
      setDuplicateInfo(null);
      return;
    }

    try {
      const response = await animalsApi.checkDuplicates(parseInt(groupId), name.trim());
      // Filter out the current animal if editing
      const filteredAnimals = currentAnimalId 
        ? response.data.animals.filter(a => a.id !== currentAnimalId)
        : response.data.animals;
      
      setDuplicateInfo({
        ...response.data,
        animals: filteredAnimals,
        has_duplicates: filteredAnimals.length > 0,
        count: filteredAnimals.length,
      });
    } catch (error) {
      console.error('Failed to check for duplicates:', error);
      setDuplicateInfo(null);
    }
  }, [groupId]);

  const loadAnimal = useCallback(async (gId: number, animalId: number) => {
    try {
      const response = await animalsApi.getById(gId, animalId);
      const animal = response.data;
      setFormData({
        ...animal,
        arrival_date: animal.arrival_date ? animal.arrival_date.split('T')[0] : '',
        quarantine_start_date: animal.quarantine_start_date ? animal.quarantine_start_date.split('T')[0] : '',
        protocol_document_url: animal.protocol_document_url || '',
        protocol_document_name: animal.protocol_document_name || '',
      });
      setOriginalStatus(animal.status);
      // Set selected tags
      if (animal.tags) {
        setSelectedTagIds(animal.tags.map(tag => tag.id));
      }
      // Check for duplicates with current animal excluded
      await checkForDuplicateNames(animal.name, animalId);
    } catch (error) {
      console.error('Failed to load animal:', error);
      toast.showError('Failed to load animal details');
    }
  }, [toast, checkForDuplicateNames]);

  useEffect(() => {
    loadAvailableTags();
    if (id && groupId) {
      loadAnimal(parseInt(groupId), parseInt(id));
    }
  }, [id, groupId, loadAvailableTags, loadAnimal]);

  const validateField = (name: string, value: string | number) => {
    switch (name) {
      case 'name':
        if (!value || (typeof value === 'string' && value.trim().length === 0)) {
          return 'Animal name is required';
        }
        if (typeof value === 'string' && value.length < 2) {
          return 'Name must be at least 2 characters';
        }
        return '';
      case 'age':
        if (typeof value === 'number' && value < 0) {
          return 'Age cannot be negative';
        }
        if (typeof value === 'number' && value > 30) {
          return 'Please enter a realistic age';
        }
        return '';
      default:
        return '';
    }
  };

  const handleBlur = (field: string) => {
    setTouched({ ...touched, [field]: true });
    const error = validateField(field, formData[field as keyof typeof formData] as string | number);
    setErrors({ ...errors, [field]: error });
  };

  const handleTagToggle = (tagId: number) => {
    if (selectedTagIds.includes(tagId)) {
      setSelectedTagIds(selectedTagIds.filter(id => id !== tagId));
    } else {
      setSelectedTagIds([...selectedTagIds, tagId]);
    }
  };

  const handleFieldChange = (field: string, value: string | number) => {
    setFormData({ ...formData, [field]: value });
    if (touched[field]) {
      const error = validateField(field, value);
      setErrors({ ...errors, [field]: error });
    }
    
    // Check for duplicate names when name field changes
    if (field === 'name' && typeof value === 'string' && value.trim().length >= 2) {
      const currentId = id ? parseInt(id) : undefined;
      checkForDuplicateNames(value, currentId);
    }
  };

  const isFormValid = () => {
    return formData.name.trim().length >= 2 && 
           !errors.name && 
           !errors.age;
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

    // Create preview URL for editor
    const previewUrl = URL.createObjectURL(file);
    setEditingImageUrl(previewUrl);
    setShowImageEditor(true);
    
    // Clear the file input
    e.target.value = '';
  };

  const handleEditorSave = async (croppedBlob: Blob) => {
    setShowImageEditor(false);
    setUploading(true);
    
    try {
      // Create a File object from the Blob with proper filename
      const fileName = `cropped-image-${Date.now()}.jpg`;
      const file = new File([croppedBlob], fileName, { type: 'image/jpeg' });
      
      // Upload cropped image
      const res = await animalsApi.uploadImage(file);
      
      // Update formData with the new image URL
      setFormData(prev => ({ ...prev, image_url: res.data.url }));
      
      toast.showSuccess('Image uploaded successfully!');
      
      // Clean up preview URL after a short delay to ensure new image loads
      setTimeout(() => {
        if (editingImageUrl) {
          URL.revokeObjectURL(editingImageUrl);
          setEditingImageUrl('');
        }
      }, 100);
    } catch (err: unknown) {
      console.error('Upload error:', err);
      const errorMsg = (err as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to upload image. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setUploading(false);
    }
  };

  const handleEditorCancel = () => {
    setShowImageEditor(false);
    // Clean up preview URL
    if (editingImageUrl) {
      URL.revokeObjectURL(editingImageUrl);
      setEditingImageUrl('');
    }
  };

  const handleProtocolDocumentUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    const fileExtension = file.name.toLowerCase().slice(file.name.lastIndexOf('.'));
    if (fileExtension !== '.pdf' && fileExtension !== '.docx') {
      toast.showError('Please select a PDF or DOCX file');
      e.target.value = '';
      return;
    }

    // Validate file size (max 20MB)
    if (file.size > 20 * 1024 * 1024) {
      toast.showError(`File size must be under 20MB. Your file is ${(file.size / 1024 / 1024).toFixed(1)}MB`);
      e.target.value = '';
      return;
    }

    // If editing existing animal, upload immediately
    if (id && groupId) {
      setUploadingDocument(true);
      try {
        const response = await animalsApi.uploadProtocolDocument(parseInt(groupId), parseInt(id), file);
        setFormData(prev => ({
          ...prev,
          protocol_document_url: response.data.url,
          protocol_document_name: response.data.name,
        }));
        toast.showSuccess('Protocol document uploaded successfully!');
      } catch (error: unknown) {
        console.error('Upload error:', error);
        const errorMsg = (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to upload document. Please try again.';
        toast.showError(errorMsg);
      } finally {
        setUploadingDocument(false);
      }
    } else {
      // For new animals, store file to upload after animal is created
      setProtocolDocumentFile(file);
      setFormData(prev => ({
        ...prev,
        protocol_document_name: file.name,
      }));
      toast.showSuccess('Protocol document ready to upload (will be saved with animal)');
    }

    // Clear the file input
    e.target.value = '';
  };

  const handleRemoveProtocolDocument = async () => {
    if (id && groupId && formData.protocol_document_url) {
      // Remove from existing animal
      try {
        await animalsApi.deleteProtocolDocument(parseInt(groupId), parseInt(id));
        setFormData(prev => ({
          ...prev,
          protocol_document_url: '',
          protocol_document_name: '',
        }));
        toast.showSuccess('Protocol document removed');
      } catch (error: unknown) {
        console.error('Failed to remove document:', error);
        toast.showError('Failed to remove protocol document');
      }
    } else {
      // Just clear the pending file
      setProtocolDocumentFile(null);
      setFormData(prev => ({
        ...prev,
        protocol_document_name: '',
      }));
    }
  };

  const loadAvailableImages = async () => {
    if (!groupId || !id) return;
    
    setLoadingImages(true);
    try {
      const response = await animalsApi.getImages(parseInt(groupId), parseInt(id));
      setAvailableImages(response.data);
    } catch (error) {
      console.error('Failed to load images:', error);
      toast.showError('Failed to load images');
    } finally {
      setLoadingImages(false);
    }
  };

  const handleSelectExistingImage = async (imageId: number) => {
    if (!id || !groupId) return;

    try {
      await animalsApi.setProfilePicture(parseInt(groupId), parseInt(id), imageId);
      const selectedImage = availableImages.find(img => img.id === imageId);
      if (selectedImage) {
        setFormData({ ...formData, image_url: selectedImage.image_url });
      }
      
      // Reload images to update profile picture badges
      await loadAvailableImages();
      
      setShowImageSelector(false);
      toast.showSuccess('Profile picture updated!');
    } catch (error) {
      console.error('Failed to set profile picture:', error);
      toast.showError('Failed to set profile picture');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate all fields
    const nameError = validateField('name', formData.name);
    const ageError = validateField('age', formData.age);
    
    if (nameError || ageError) {
      setErrors({ name: nameError, age: ageError });
      setTouched({ name: true, age: true });
      toast.showError('Please fix the errors in the form');
      return;
    }

    // Check if status changed to bite_quarantine
    if (formData.status === 'bite_quarantine' && originalStatus !== 'bite_quarantine') {
      // Show modal to get context and date
      setQuarantineDate(formData.quarantine_start_date || new Date().toISOString().split('T')[0]);
      setShowQuarantineModal(true);
      return;
    }

    // If not changing to quarantine, proceed with normal save
    await saveAnimal();
  };

  const saveAnimal = async () => {
    setLoading(true);
    try {
      let animalId = id ? parseInt(id) : null;
      
      // Clean up formData: convert empty quarantine_start_date to null
      const cleanedFormData = {
        ...formData,
        quarantine_start_date: formData.quarantine_start_date || undefined,
      };
      
      if (id && groupId) {
        await animalsApi.update(parseInt(groupId), parseInt(id), cleanedFormData);
        toast.showSuccess('Animal updated successfully!');
      } else if (groupId) {
        const response = await animalsApi.create(parseInt(groupId), cleanedFormData);
        animalId = response.data.id;
        toast.showSuccess('Animal added successfully!');
        
        // Upload protocol document if one was selected for new animal
        if (animalId && protocolDocumentFile) {
          try {
            await animalsApi.uploadProtocolDocument(parseInt(groupId), animalId, protocolDocumentFile);
            toast.showSuccess('Protocol document uploaded!');
          } catch (error) {
            console.error('Failed to upload protocol document:', error);
            toast.showWarning('Animal saved but protocol document upload failed');
          }
        }
      }
      
      // Assign tags to the animal
      if (animalId && selectedTagIds.length >= 0 && groupId) {
        try {
          await animalTagsApi.assignToAnimal(parseInt(groupId), animalId, selectedTagIds);
        } catch (error) {
          console.error('Failed to assign tags:', error);
          // Don't fail the whole operation if tag assignment fails
          toast.showWarning('Animal saved but failed to update tags');
        }
      }
      
      navigate(`/groups/${groupId}`);
    } catch (error: unknown) {
      const errorMsg = (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to save animal. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  const handleQuarantineSubmit = async () => {
    if (!groupId) return;
    if (!quarantineContext.trim()) {
      toast.showError('Please provide context about the bite incident');
      return;
    }

    // Update formData with the quarantine date
    const updatedFormData = {
      ...formData,
      quarantine_start_date: quarantineDate,
    };

    setLoading(true);
    try {
      // Save the animal first (and get the animal ID for comments)
      let animalId: number | null = null;
      if (id && groupId) {
        const response = await animalsApi.update(parseInt(groupId), parseInt(id), updatedFormData);
        console.log('Update response:', response);
        console.log('Response data:', response.data);
        console.log('Animal ID from response:', response.data.id);
        animalId = response.data.id;
      } else if (groupId) {
        const response = await animalsApi.create(parseInt(groupId), updatedFormData);
        console.log('Create response:', response);
        console.log('Response data:', response.data);
        console.log('Animal ID from response:', response.data.id);
        animalId = response.data.id;
      }

      const endDate = calculateQuarantineEndDate(quarantineDate, true);
      
      // Create a comment on the animal with behavior tag
      if (animalId && groupId) {
        try {
          // Fetch comment tags to find the behavior tag
          const commentTagsResponse = await commentTagsApi.getAll(parseInt(groupId));
          const commentTags = commentTagsResponse.data;
          const behaviorTag = commentTags.find(tag => tag.name.toLowerCase() === 'behavior');
          
          // Create comment with bite quarantine details
          const commentContent = `🚨 BITE QUARANTINE\n\n` +
            `Quarantine Start: ${new Date(quarantineDate).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })}\n` +
            `Quarantine End: ${endDate}\n\n` +
            `Incident Details:\n${quarantineContext}`;
          
          console.log('Creating comment for animal ID:', animalId, 'in group:', groupId);
          console.log('Comment tags:', commentTags);
          console.log('Behavior tag:', behaviorTag);
          
          await animalCommentsApi.create(
            parseInt(groupId),
            animalId,
            commentContent,
            undefined, // no image
            behaviorTag ? [behaviorTag.id] : [] // attach behavior tag if found
          );
          
          console.log('Comment created successfully');
        } catch (commentError) {
          console.error('Failed to create comment:', commentError);
          // Don't fail the whole operation if comment creation fails
          toast.showWarning('Animal updated but comment creation failed');
        }
      } else {
        console.warn('Missing animalId or groupId, skipping comment creation:', { animalId, groupId });
      }

      // Create group update (post) with behavior tag context for activity feed
      const updateTitle = `🚨 Bite Quarantine: ${formData.name}`;
      const updateContent = `${formData.name} has been placed in bite quarantine.\n\n` +
        `Quarantine Start: ${new Date(quarantineDate).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })}\n` +
        `Quarantine End: ${endDate}\n\n` +
        `Details:\n${quarantineContext}\n\n` +
        `#behavior`;

      // Create group update (shows in activity feed) with GroupMe notification
      // Parameters: (groupId, title, content, send_groupme, image_url?)
      await updatesApi.create(parseInt(groupId), updateTitle, updateContent, true);

      toast.showSuccess('Animal updated, comment added, and announcement posted successfully!');
      setShowQuarantineModal(false);
      navigate(`/groups/${groupId}`);
    } catch (error: unknown) {
      const errorMsg = (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to save animal. Please try again.';
      toast.showError(errorMsg);
    } finally {
      setLoading(false);
    }
  };


  const handleDelete = async () => {
    try {
      if (id && groupId) {
        await animalsApi.delete(parseInt(groupId), parseInt(id));
        toast.showSuccess('Animal deleted successfully');
        navigate(`/groups/${groupId}`);
      }
    } catch (error: unknown) {
      const errorMsg = (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to delete animal. Please try again.';
      toast.showError(errorMsg);
    }
    setShowDeleteModal(false);
  };

  return (
    <div className="form-page">
      <div className="form-container">
        <Link to={`/groups/${groupId}`} className="back-link">
          ← Back to Group
        </Link>
        <h1>{id ? 'Edit Animal' : 'Add New Animal'}</h1>
        <form onSubmit={handleSubmit}>
          <FormField
            label="Name"
            id="name"
            type="text"
            value={formData.name}
            onChange={(value) => handleFieldChange('name', value)}
            onBlur={() => handleBlur('name')}
            error={touched.name ? errors.name : ''}
            success={touched.name && !errors.name && formData.name.length >= 2}
            required
            helperText="The animal's name as it will appear to volunteers"
            autoComplete="off"
            minLength={2}
          />

          {/* Duplicate Name Warning */}
          {duplicateInfo && duplicateInfo.has_duplicates && (
            <div className="duplicate-warning" role="alert">
              <div className="duplicate-warning-header">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                  <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <line x1="12" y1="9" x2="12" y2="13" />
                  <line x1="12" y1="17" x2="12.01" y2="17" />
                </svg>
                <strong>Name Collision Detected</strong>
              </div>
              <p>
                There {duplicateInfo.count === 1 ? 'is' : 'are'} already <strong>{duplicateInfo.count}</strong> other animal{duplicateInfo.count > 1 ? 's' : ''} named "{formData.name}" in this group.
              </p>
              <div className="duplicate-animals-list">
                {duplicateInfo.animals.slice(0, 3).map((animal: Animal) => (
                  <div key={animal.id} className="duplicate-animal-item">
                    <span className="duplicate-animal-name">
                      {animal.name} (ID: {animal.id})
                    </span>
                    <span className="duplicate-animal-details">
                      {animal.breed || 'Unknown breed'} • {animal.age} years • {animal.status}
                    </span>
                  </div>
                ))}
                {duplicateInfo.count > 3 && (
                  <p className="duplicate-more">+ {duplicateInfo.count - 3} more</p>
                )}
              </div>
              <p className="duplicate-warning-footer">
                💡 <strong>Tip:</strong> Consider adding breed, color, or other identifying details to help volunteers distinguish between animals.
              </p>
            </div>
          )}

          <div className="form-row">
            <FormField
              label="Species"
              id="species"
              type="text"
              value={formData.species}
              onChange={(value) => handleFieldChange('species', value)}
              helperText="e.g., Dog, Cat, Rabbit"
              autoComplete="off"
            />

            <FormField
              label="Breed"
              id="breed"
              type="text"
              value={formData.breed}
              onChange={(value) => handleFieldChange('breed', value)}
              helperText="e.g., Labrador, Tabby, Mixed"
              autoComplete="off"
            />
          </div>

          <div className="form-row">
            <FormField
              label="Age (years)"
              id="age"
              type="number"
              value={formData.age}
              onChange={(value) => handleFieldChange('age', parseInt(value) || 0)}
              onBlur={() => handleBlur('age')}
              error={touched.age ? errors.age : ''}
              success={touched.age && !errors.age && formData.age >= 0}
              helperText="Approximate age in years"
            />

            <div className="form-field">
              <label htmlFor="status" className="form-field__label">
                Status
              </label>
              <select
                id="status"
                value={formData.status}
                onChange={(e) => {
                  const newStatus = e.target.value;
                  // If changing away from archived, show confirmation modal
                  if (originalStatus === 'archived' && newStatus !== 'archived' && id) {
                    setPendingStatusChange(newStatus);
                    setUnarchiveIsReturned(formData.is_returned);
                    setShowUnarchiveModal(true);
                  } else {
                    setFormData({ ...formData, status: newStatus });
                  }
                }}
                className="form-field__input"
              >
                <option value="available">Available</option>
                <option value="foster">Foster</option>
                <option value="bite_quarantine">Bite Quarantine</option>
                <option value="archived">Archived</option>
              </select>
              <p className="form-field__helper">Current status of the animal</p>
            </div>
          </div>

          <div className="form-field">
            <label htmlFor="arrival_date" className="form-field__label">
              Date in Shelter
            </label>
            <input
              id="arrival_date"
              type="date"
              value={formData.arrival_date}
              onChange={(e) => setFormData({ ...formData, arrival_date: e.target.value })}
              className="form-field__input"
              max={new Date().toISOString().split('T')[0]}
            />
            <p className="form-field__helper">
              Date the animal entered the shelter. Used to calculate length of stay. Leave empty to use today's date.
            </p>
          </div>

          <div className="form-field">
            <label htmlFor="image_upload" className="form-field__label">
              Animal Image {formData.image_url && <span className="label-badge">Profile Picture</span>}
            </label>
            
            {formData.image_url && (
              <div className="image-preview">
                <img src={formData.image_url} alt="Current profile picture" />
              </div>
            )}
            
            <div className="image-upload-options">
              <input
                id="image_upload"
                type="file"
                accept="image/jpeg,image/png,image/gif"
                onChange={handleImageUpload}
                className="form-field__input"
                disabled={uploading}
                aria-label="Upload animal image"
                style={{ display: 'none' }}
              />
              <Button
                type="button"
                variant="secondary"
                onClick={() => document.getElementById('image_upload')?.click()}
                disabled={uploading}
              >
                📤 {formData.image_url ? 'Upload Different Image' : 'Upload New Image'}
              </Button>
              
              {id && (
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => {
                    loadAvailableImages();
                    setShowImageSelector(true);
                  }}
                  disabled={uploading}
                >
                  🖼️ Choose from Gallery
                </Button>
              )}
            </div>
            
            <p className="form-field__helper">
              Upload a photo (JPG, PNG, or GIF, max 10MB) {id && 'or choose from uploaded images'}
            </p>
            {uploading && (
              <p className="form-field__helper" style={{ color: 'var(--color-primary)' }}>
                Uploading image...
              </p>
            )}
          </div>

          <FormField
            label="Description"
            id="description"
            type="textarea"
            value={formData.description}
            onChange={(value) => setFormData({ ...formData, description: value })}
            rows={4}
            helperText="Optional details about the animal's personality, care needs, or history"
            maxLength={1000}
          />

          {/* Animal Tags Selection */}
          <div className="form-field">
            <label className="form-field__label">Animal Tags</label>
            <p className="form-field__helper">
              Select tags for behavior traits and walker status
            </p>
            
            {availableTags.length > 0 && (
              <div className="tag-categories">
                {/* Behavior Tags */}
                <div className="tag-category">
                  <h4>Behavior Traits</h4>
                  <div className="tag-selection-grid">
                    {availableTags.filter(tag => tag.category === 'behavior').map(tag => (
                      <label key={tag.id} className="tag-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedTagIds.includes(tag.id)}
                          onChange={() => handleTagToggle(tag.id)}
                        />
                        <span
                          className="tag-label"
                          style={{ 
                            backgroundColor: selectedTagIds.includes(tag.id) ? tag.color : 'transparent',
                            borderColor: tag.color,
                            color: selectedTagIds.includes(tag.id) ? 'white' : tag.color
                          }}
                        >
                          {tag.name}
                        </span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* Walker Status Tags */}
                <div className="tag-category">
                  <h4>Walker Status</h4>
                  <div className="tag-selection-grid">
                    {availableTags.filter(tag => tag.category === 'walker_status').map(tag => (
                      <label key={tag.id} className="tag-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedTagIds.includes(tag.id)}
                          onChange={() => handleTagToggle(tag.id)}
                        />
                        <span
                          className="tag-label"
                          style={{ 
                            backgroundColor: selectedTagIds.includes(tag.id) ? tag.color : 'transparent',
                            borderColor: tag.color,
                            color: selectedTagIds.includes(tag.id) ? 'white' : tag.color
                          }}
                        >
                          {tag.name}
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Protocol Document Upload Section */}
          <div className="form-field">
            <label htmlFor="protocol_document_upload" className="form-field__label">
              Protocol Document (Optional)
              {formData.protocol_document_name && <span className="label-badge">Uploaded</span>}
            </label>
            <p className="form-field__helper">
              Upload care protocols, medical instructions, or behavioral guidelines for this animal (PDF or DOCX, max 20MB)
            </p>
            
            {formData.protocol_document_url || formData.protocol_document_name ? (
              <div className="document-preview">
                <div className="document-info">
                  <span className="document-icon">
                    {formData.protocol_document_name?.endsWith('.pdf') ? '📄' : '📝'}
                  </span>
                  <div className="document-details">
                    <span className="document-name">{formData.protocol_document_name}</span>
                    {formData.protocol_document_url && (
                      <a 
                        href={formData.protocol_document_url} 
                        target="_blank" 
                        rel="noopener noreferrer"
                        className="document-view-link"
                      >
                        View Document →
                      </a>
                    )}
                    {!formData.protocol_document_url && protocolDocumentFile && (
                      <span className="document-pending">Will be uploaded when animal is saved</span>
                    )}
                  </div>
                  <Button
                    type="button"
                    variant="danger"
                    onClick={handleRemoveProtocolDocument}
                    disabled={uploadingDocument}
                    size="small"
                  >
                    Remove
                  </Button>
                </div>
              </div>
            ) : (
              <div className="document-upload-area">
                <input
                  id="protocol_document_upload"
                  type="file"
                  accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
                  onChange={handleProtocolDocumentUpload}
                  className="form-field__input"
                  disabled={uploadingDocument || uploading}
                  aria-label="Upload protocol document"
                  style={{ display: 'none' }}
                />
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => document.getElementById('protocol_document_upload')?.click()}
                  disabled={uploadingDocument || uploading}
                >
                  📎 {uploadingDocument ? 'Uploading...' : 'Upload Protocol Document'}
                </Button>
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
              {id ? 'Update Animal' : 'Add Animal'}
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => navigate(`/groups/${groupId}`)}
              disabled={loading}
            >
              Cancel
            </Button>
            {id && (
              <Button
                type="button"
                onClick={() => setShowDeleteModal(true)}
                variant="danger"
                disabled={loading}
              >
                Delete
              </Button>
            )}
          </div>
        </form>
      </div>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        title="Delete Animal"
        size="small"
      >
        <p>Are you sure you want to delete <strong>{formData.name}</strong>?</p>
        <p>This action cannot be undone. All comments and photos associated with this animal will also be deleted.</p>
        <div className="modal__actions" style={{ marginTop: '1.5rem', display: 'flex', gap: '0.75rem', justifyContent: 'flex-end' }}>
          <Button
            variant="secondary"
            onClick={() => setShowDeleteModal(false)}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={handleDelete}
          >
            Delete Animal
          </Button>
        </div>
      </Modal>

      {/* Bite Quarantine Context Modal */}
      <Modal
        isOpen={showQuarantineModal}
        onClose={handleQuarantineModalClose}
        title="🚨 Bite Quarantine Information"
        size="medium"
      >
        <p style={{ marginBottom: '1rem' }}>
          Please provide details about the bite incident. This information will be posted as an announcement with the #behavior tag.
        </p>
        
        <div style={{ marginBottom: '1rem' }}>
          <label htmlFor="quarantine-date" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: '500' }}>
            Bite Date:
          </label>
          <input
            id="quarantine-date"
            type="date"
            value={quarantineDate}
            onChange={(e) => setQuarantineDate(e.target.value)}
            style={{
              width: '100%',
              padding: '0.5rem',
              border: '1px solid var(--neutral-300)',
              borderRadius: '4px',
              fontSize: '1rem'
            }}
          />
          <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)', marginTop: '0.25rem' }}>
            Quarantine will end: {calculateQuarantineEndDate(quarantineDate, true)}
          </p>
        </div>

        <div style={{ marginBottom: '1.5rem' }}>
          <label htmlFor="quarantine-context" style={{ display: 'block', marginBottom: '0.5rem', fontWeight: '500' }}>
            Incident Details: *
          </label>
          <textarea
            id="quarantine-context"
            value={quarantineContext}
            onChange={(e) => setQuarantineContext(e.target.value)}
            placeholder="Describe what happened, who was involved, any injuries, and any other relevant details..."
            rows={6}
            required
            style={{
              width: '100%',
              padding: '0.75rem',
              border: '1px solid var(--neutral-300)',
              borderRadius: '4px',
              fontSize: '1rem',
              fontFamily: 'inherit',
              resize: 'vertical'
            }}
          />
        </div>

        <div className="modal__actions" style={{ marginTop: '1.5rem', display: 'flex', gap: '0.75rem', justifyContent: 'flex-end' }}>
          <Button
            variant="secondary"
            onClick={handleQuarantineModalClose}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleQuarantineSubmit}
            loading={loading}
            disabled={loading || !quarantineContext.trim()}
          >
            Save & Post Announcement
          </Button>
        </div>
      </Modal>

      {/* Unarchive Confirmation Modal */}
      <Modal
        isOpen={showUnarchiveModal}
        onClose={handleUnarchiveModalClose}
        title="Confirm Status Change"
        size="medium"
      >
        <p style={{ marginBottom: '1rem' }}>
          You are changing <strong>{formData.name}</strong> from <strong>Archived</strong> to{' '}
          <strong>{pendingStatusChange === 'available' ? 'Available' : pendingStatusChange === 'foster' ? 'Foster' : 'Bite Quarantine'}</strong>.
        </p>
        
        {pendingStatusChange === 'available' && (
          <>
            <div style={{ marginBottom: '1.5rem', padding: '1rem', backgroundColor: 'var(--neutral-100)', borderRadius: '8px' }}>
              <p style={{ marginBottom: '0.75rem', fontWeight: '500' }}>
                Was this animal returning to the shelter?
              </p>
              <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)', marginBottom: '1rem' }}>
                Select "Yes" if this animal previously left the shelter (adopted, fostered, transferred) and has now come back. 
                Select "No" if this is a different situation (e.g., moving from quarantine or correcting a mistake).
              </p>
              <div style={{ display: 'flex', gap: '1rem' }}>
                <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
                  <input
                    type="radio"
                    name="is_returned"
                    checked={unarchiveIsReturned === true}
                    onChange={() => setUnarchiveIsReturned(true)}
                    style={{ marginRight: '0.5rem' }}
                  />
                  <span>Yes, this is a returning animal</span>
                </label>
                <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
                  <input
                    type="radio"
                    name="is_returned"
                    checked={unarchiveIsReturned === false}
                    onChange={() => setUnarchiveIsReturned(false)}
                    style={{ marginRight: '0.5rem' }}
                  />
                  <span>No, not a return</span>
                </label>
              </div>
            </div>
          </>
        )}

        <div className="modal__actions" style={{ marginTop: '1.5rem', display: 'flex', gap: '0.75rem', justifyContent: 'flex-end' }}>
          <Button
            variant="secondary"
            onClick={() => {
              setShowUnarchiveModal(false);
              setPendingStatusChange('');
            }}
          >
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={() => {
              setFormData({ 
                ...formData, 
                status: pendingStatusChange,
                is_returned: pendingStatusChange === 'available' ? unarchiveIsReturned : false
              });
              setShowUnarchiveModal(false);
              setPendingStatusChange('');
            }}
          >
            Confirm Change
          </Button>
        </div>
      </Modal>

      {/* Image Selector Modal */}
      <Modal
        isOpen={showImageSelector}
        onClose={() => setShowImageSelector(false)}
        title="Select Profile Picture"
      >
        {loadingImages ? (
          <div style={{ padding: '2rem', textAlign: 'center' }}>
            <p>Loading images...</p>
          </div>
        ) : availableImages.length === 0 ? (
          <div style={{ padding: '2rem', textAlign: 'center' }}>
            <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
              No images available yet.
            </p>
            <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)' }}>
              Upload images to the photo gallery first, then you can select them as profile pictures.
            </p>
            {groupId && id && (
              <Link
                to={`/groups/${groupId}/animals/${id}/photos`}
                style={{ 
                  display: 'inline-block', 
                  marginTop: '1rem',
                  color: 'var(--brand)',
                  textDecoration: 'underline'
                }}
              >
                Go to Photo Gallery →
              </Link>
            )}
          </div>
        ) : (
          <div className="image-selector-grid">
            {availableImages.map((image) => (
              <div
                key={image.id}
                className={`selectable-image ${image.is_profile_picture ? 'current' : ''}`}
                onClick={() => handleSelectExistingImage(image.id)}
                style={{
                  cursor: 'pointer',
                  border: image.is_profile_picture ? '3px solid var(--brand)' : '2px solid var(--border)',
                  borderRadius: '8px',
                  overflow: 'hidden',
                  position: 'relative',
                  transition: 'all 0.2s',
                }}
                onMouseEnter={(e) => {
                  if (!image.is_profile_picture) {
                    e.currentTarget.style.borderColor = 'var(--brand)';
                  }
                }}
                onMouseLeave={(e) => {
                  if (!image.is_profile_picture) {
                    e.currentTarget.style.borderColor = 'var(--border)';
                  }
                }}
              >
                <img
                  src={image.image_url}
                  alt={image.caption || 'Gallery image'}
                  style={{
                    width: '100%',
                    height: '200px',
                    objectFit: 'cover',
                    display: 'block',
                  }}
                />
                {image.is_profile_picture && (
                  <div
                    style={{
                      position: 'absolute',
                      top: '0.5rem',
                      right: '0.5rem',
                      background: 'var(--brand)',
                      color: 'white',
                      padding: '0.25rem 0.75rem',
                      borderRadius: '12px',
                      fontSize: '0.75rem',
                      fontWeight: 600,
                    }}
                  >
                    Current
                  </div>
                )}
                <div style={{ padding: '0.75rem', backgroundColor: 'var(--bg-secondary)' }}>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                    <div>By {image.user?.username || 'Unknown'}</div>
                    <div>{new Date(image.created_at).toLocaleDateString()}</div>
                  </div>
                  {image.caption && (
                    <div style={{ 
                      fontSize: '0.875rem', 
                      marginTop: '0.5rem',
                      color: 'var(--text-primary)',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}>
                      {image.caption}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
        
        <div style={{ marginTop: '1.5rem', display: 'flex', justifyContent: 'flex-end' }}>
          <Button
            variant="secondary"
            onClick={() => setShowImageSelector(false)}
          >
            Close
          </Button>
        </div>
      </Modal>

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
  );
};

export default AnimalForm;
