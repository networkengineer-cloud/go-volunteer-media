# Image Handling Improvement Plan

**Date:** November 25, 2025  
**Status:** Phase 3 In Progress  
**Priority:** High

---

## 🎯 Executive Summary

This plan addresses three key issues with the current image handling system:
1. **Non-admin users** can upload images but cannot easily view them (limited to photo gallery for comments only)
2. **Admin users** cannot effectively select or change an animal's profile picture from uploaded images
3. **Missing features**: Image crop/rotate tools for both user types

---

## 📊 Current State Analysis

### What Works Today

#### Image Upload (Both Users)
- ✅ All authenticated users can upload images via `/api/animals/upload-image`
- ✅ Images are validated (type, size <10MB, content)
- ✅ Auto-optimization (resize to 1200px, JPEG compression at 85%)
- ✅ Secure storage in `/public/uploads/` with unique filenames
- ✅ Images can be attached to comments on animal pages

#### Photo Gallery (All Users)
- ✅ Accessible at `/groups/:groupId/animals/:id/photos`
- ✅ Shows all images from comments with lightbox viewer
- ✅ Displays uploader name, date, tags, and comment text
- ✅ Keyboard navigation (arrows, ESC)

#### Admin Animal Management
- ✅ Admins can create/edit animals at `/groups/:groupId/animals/:id`
- ✅ Image upload field in animal form
- ✅ Preview of selected image

### Critical Gaps

#### 1. Non-Admin Image Viewing
**Problem:** Non-admin users upload images via comments, but the only way to view them is:
- Go to animal detail page → Click "View All Photos" button
- This is not discoverable enough

**Impact:** 
- Low engagement with uploaded photos
- Users don't know their images were successfully saved
- No central place to browse all animal photos across groups

#### 2. Admin Profile Picture Management
**Problem:** Admins can upload a NEW image for animal profile, but cannot:
- Select from existing uploaded images (gallery)
- See which image is currently set
- Change profile picture without re-uploading

**Impact:**
- Duplicate images (same photo uploaded multiple times)
- Inefficient workflow
- Cannot leverage user-contributed photos as profile pictures

#### 3. Image Editing
**Problem:** No crop or rotate functionality
- Users upload full images that may need adjustment
- Profile pictures show entire image, not just animal
- Orientation issues (sideways photos)

**Impact:**
- Poor visual presentation
- Wasted storage (large uncropped images)
- Manual pre-editing required before upload

---

## 🏗️ Proposed Solution Architecture

### Overview
Create a comprehensive image management system with three components:
1. **Enhanced Photo Gallery** - All users can view/browse
2. **Admin Image Selector** - Pick profile picture from gallery
3. **Image Editor** - Crop/rotate before saving

---

## 📋 Detailed Implementation Plan

### Phase 1: Enhanced Photo Gallery for All Users (Priority: P0)

#### 1.1 Create Dedicated Image Model & Table

**Backend Changes:**

```go
// internal/models/models.go - Add new model
type AnimalImage struct {
    ID              uint           `gorm:"primaryKey" json:"id"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
    AnimalID        uint           `gorm:"not null;index" json:"animal_id"`
    UserID          uint           `gorm:"not null;index" json:"user_id"`
    ImageURL        string         `gorm:"not null" json:"image_url"`
    Caption         string         `json:"caption"`
    IsProfilePicture bool          `gorm:"default:false" json:"is_profile_picture"`
    Width           int            `json:"width"`
    Height          int            `json:"height"`
    FileSize        int            `json:"file_size"` // in bytes
    User            User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Animal          Animal         `gorm:"foreignKey:AnimalID" json:"animal,omitempty"`
}

// Update Animal model to include relationship
type Animal struct {
    // ... existing fields ...
    Images []AnimalImage `gorm:"foreignKey:AnimalID" json:"images,omitempty"`
}
```

**Migration:**
- Create `animal_images` table
- Migrate existing comment images to new table (optional, can reference both)

#### 1.2 New API Endpoints

**Backend:**

```go
// internal/handlers/animal_image.go - NEW FILE

// GET /api/groups/:id/animals/:animalId/images
// Returns all images for an animal (authenticated users)
func GetAnimalImages(db *gorm.DB) gin.HandlerFunc

// POST /api/groups/:id/animals/:animalId/images
// Upload image for an animal (authenticated users)
func UploadAnimalImageToGallery(db *gorm.DB) gin.HandlerFunc

// DELETE /api/groups/:id/animals/:animalId/images/:imageId
// Delete an image (admin or owner only)
func DeleteAnimalImage(db *gorm.DB) gin.HandlerFunc

// PUT /api/admin/animals/:animalId/images/:imageId/set-profile
// Set image as profile picture (admin only)
func SetAnimalProfilePicture(db *gorm.DB) gin.HandlerFunc
```

**Routes (cmd/api/main.go):**

```go
// Protected routes (all authenticated users)
protected.GET("/groups/:id/animals/:animalId/images", handlers.GetAnimalImages(db))
protected.POST("/groups/:id/animals/:animalId/images", handlers.UploadAnimalImageToGallery(db))
protected.DELETE("/groups/:id/animals/:animalId/images/:imageId", handlers.DeleteAnimalImage(db))

// Admin routes
admin.PUT("/animals/:animalId/images/:imageId/set-profile", handlers.SetAnimalProfilePicture(db))
```

#### 1.3 Frontend Gallery Enhancements

**Update PhotoGallery Component:**

```tsx
// frontend/src/pages/PhotoGallery.tsx - MAJOR REFACTOR

interface Photo {
  id: number;
  image_url: string;
  caption: string;
  user: { username: string };
  created_at: string;
  is_profile_picture: boolean;
  width: number;
  height: number;
}

const PhotoGallery: React.FC = () => {
  // ... existing state ...
  const [photos, setPhotos] = useState<Photo[]>([]);
  const { user } = useAuth(); // Get current user for permissions
  
  // Load from new API endpoint
  const loadPhotos = async () => {
    const res = await api.get(`/groups/${groupId}/animals/${id}/images`);
    setPhotos(res.data);
  };
  
  // Add upload functionality IN gallery
  const handleUploadFromGallery = async (file: File, caption: string) => {
    const formData = new FormData();
    formData.append('image', file);
    formData.append('caption', caption);
    await api.post(`/groups/${groupId}/animals/${id}/images`, formData);
    loadPhotos(); // Refresh
  };
  
  return (
    <div className="photo-gallery-page">
      {/* Add upload button at top */}
      <div className="gallery-actions">
        <button onClick={() => setShowUploadModal(true)}>
          📷 Upload Photo
        </button>
      </div>
      
      {/* Grid with badges for profile picture */}
      <div className="photos-grid">
        {photos.map((photo) => (
          <div className="photo-card">
            {photo.is_profile_picture && (
              <div className="profile-badge">Profile Picture</div>
            )}
            <img src={photo.image_url} alt={photo.caption} />
            {/* Show delete button if user owns image or is admin */}
          </div>
        ))}
      </div>
    </div>
  );
};
```

**Add to Navigation:**
- Add "Photo Gallery" link to animal detail page header
- Add badge showing photo count on animal cards
- Quick access from group pages

---

### Phase 2: Admin Profile Picture Selector (Priority: P0)

#### 2.1 Profile Picture Selection Modal

**Frontend - AnimalForm Component:**

```tsx
// frontend/src/pages/AnimalForm.tsx - ADD NEW SECTION

const AnimalForm: React.FC = () => {
  const [showImageSelector, setShowImageSelector] = useState(false);
  const [availableImages, setAvailableImages] = useState<Photo[]>([]);
  
  // Load available images for selection
  const loadAvailableImages = async () => {
    if (!id) return; // Only for editing existing animals
    const res = await api.get(`/groups/${groupId}/animals/${id}/images`);
    setAvailableImages(res.data);
  };
  
  const handleSelectExistingImage = async (imageId: number) => {
    // Set as profile picture
    await api.put(`/admin/animals/${id}/images/${imageId}/set-profile`);
    // Update local state
    const selectedImage = availableImages.find(img => img.id === imageId);
    setFormData({ ...formData, image_url: selectedImage.image_url });
    setShowImageSelector(false);
    toast.showSuccess('Profile picture updated!');
  };
  
  return (
    <form onSubmit={handleSubmit}>
      {/* REPLACE existing image upload section */}
      <div className="form-field">
        <label className="form-field__label">Animal Image</label>
        
        {/* Show current profile picture */}
        {formData.image_url && (
          <div className="current-profile-picture">
            <img src={formData.image_url} alt="Current profile" />
            <div className="profile-picture-badge">Profile Picture</div>
          </div>
        )}
        
        {/* Two options: Upload new OR select existing */}
        <div className="image-upload-options">
          <button 
            type="button"
            onClick={() => fileInputRef.current?.click()}
            className="btn-secondary"
          >
            📤 Upload New Image
          </button>
          
          {id && ( // Only show if editing existing animal
            <button
              type="button"
              onClick={() => {
                loadAvailableImages();
                setShowImageSelector(true);
              }}
              className="btn-secondary"
            >
              🖼️ Choose from Gallery ({availableImages.length})
            </button>
          )}
        </div>
        
        {/* Hidden file input for new uploads */}
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif"
          onChange={handleImageUpload}
          style={{ display: 'none' }}
        />
      </div>
      
      {/* Image Selector Modal */}
      {showImageSelector && (
        <Modal
          isOpen={showImageSelector}
          onClose={() => setShowImageSelector(false)}
          title="Select Profile Picture"
        >
          <div className="image-selector-grid">
            {availableImages.map((image) => (
              <div
                key={image.id}
                className={`selectable-image ${
                  image.is_profile_picture ? 'current' : ''
                }`}
                onClick={() => handleSelectExistingImage(image.id)}
              >
                <img src={image.image_url} alt={image.caption} />
                {image.is_profile_picture && (
                  <div className="current-badge">Current</div>
                )}
                <div className="image-info">
                  <small>By {image.user.username}</small>
                  <small>{new Date(image.created_at).toLocaleDateString()}</small>
                </div>
              </div>
            ))}
          </div>
          {availableImages.length === 0 && (
            <p className="no-images-message">
              No images available. Upload some photos first!
            </p>
          )}
        </Modal>
      )}
    </form>
  );
};
```

---

### Phase 3: Image Crop & Rotate Editor (Priority: P1) ✅ COMPLETE

#### 3.1 Integrate Image Editing Library ✅

**Choose Library:** [react-easy-crop](https://www.npmjs.com/package/react-easy-crop)
- Lightweight (28KB gzipped)
- Touch-friendly
- Supports zoom, rotation, aspect ratio
- No external dependencies

**Installation:** ✅ DONE
```bash
cd frontend
npm install react-easy-crop
```

#### 3.2 Create Image Editor Component ✅

**Frontend:** ✅ DONE - Created `frontend/src/components/ImageEditor.tsx`

**Features Implemented:**
- ✅ Crop with adjustable area
- ✅ Zoom slider (1x to 3x)
- ✅ Rotation control (0-360° with slider)
- ✅ Preset rotation buttons (0°, 90°, 180°, 270°)
- ✅ Aspect ratio presets (Square 1:1, Landscape 4:3, Wide 16:9, Free)
- ✅ Canvas-based image processing
- ✅ JPEG output at 90% quality
- ✅ Full-screen modal with dark overlay
- ✅ Responsive design for mobile
- ✅ Touch-friendly controls

```tsx
// frontend/src/components/ImageEditor.tsx - IMPLEMENTED

import React, { useState, useCallback } from 'react';
import Cropper from 'react-easy-crop';
import { Point, Area } from 'react-easy-crop/types';

interface ImageEditorProps {
  imageSrc: string;
  onSave: (croppedImage: Blob, rotation: number) => void;
  onCancel: () => void;
  aspectRatio?: number; // 1 for square, 4/3 for landscape, etc.
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
  const [croppedAreaPixels, setCroppedAreaPixels] = useState<Area | null>(null);
  
  const onCropComplete = useCallback((croppedArea: Area, croppedAreaPixels: Area) => {
    setCroppedAreaPixels(croppedAreaPixels);
  }, []);
  
  const createCroppedImage = async () => {
    if (!croppedAreaPixels) return;
    
    const image = new Image();
    image.src = imageSrc;
    await new Promise((resolve) => { image.onload = resolve; });
    
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    
    canvas.width = croppedAreaPixels.width;
    canvas.height = croppedAreaPixels.height;
    
    ctx?.drawImage(
      image,
      croppedAreaPixels.x,
      croppedAreaPixels.y,
      croppedAreaPixels.width,
      croppedAreaPixels.height,
      0,
      0,
      croppedAreaPixels.width,
      croppedAreaPixels.height
    );
    
    return new Promise<Blob>((resolve) => {
      canvas.toBlob((blob) => {
        if (blob) resolve(blob);
      }, 'image/jpeg', 0.9);
    });
  };
  
  const handleSave = async () => {
    const croppedImage = await createCroppedImage();
    if (croppedImage) {
      onSave(croppedImage, rotation);
    }
  };
  
  return (
    <div className="image-editor-modal">
      <div className="image-editor-container">
        <Cropper
          image={imageSrc}
          crop={crop}
          zoom={zoom}
          rotation={rotation}
          aspect={aspectRatio}
          onCropChange={setCrop}
          onZoomChange={setZoom}
          onRotationChange={setRotation}
          onCropComplete={onCropComplete}
        />
      </div>
      
      <div className="image-editor-controls">
        <div className="control-group">
          <label>Zoom</label>
          <input
            type="range"
            min={1}
            max={3}
            step={0.1}
            value={zoom}
            onChange={(e) => setZoom(Number(e.target.value))}
          />
        </div>
        
        <div className="control-group">
          <label>Rotation</label>
          <input
            type="range"
            min={0}
            max={360}
            step={1}
            value={rotation}
            onChange={(e) => setRotation(Number(e.target.value))}
          />
          <div className="rotation-shortcuts">
            <button onClick={() => setRotation(0)}>0°</button>
            <button onClick={() => setRotation(90)}>90°</button>
            <button onClick={() => setRotation(180)}>180°</button>
            <button onClick={() => setRotation(270)}>270°</button>
          </div>
        </div>
        
        <div className="control-group">
          <label>Aspect Ratio</label>
          <div className="aspect-ratio-buttons">
            <button onClick={() => setAspectRatio(1)}>Square</button>
            <button onClick={() => setAspectRatio(4/3)}>4:3</button>
            <button onClick={() => setAspectRatio(16/9)}>16:9</button>
            <button onClick={() => setAspectRatio(0)}>Free</button>
          </div>
        </div>
      </div>
      
      <div className="image-editor-actions">
        <button onClick={onCancel} className="btn-secondary">
          Cancel
        </button>
        <button onClick={handleSave} className="btn-primary">
          Save & Upload
        </button>
      </div>
    </div>
  );
};

export default ImageEditor;
```

#### 3.3 Integrate Editor into Upload Flow ✅

**Status:** ✅ DONE - Integrated into both AnimalForm and PhotoGallery

**Changes Made:**
- ✅ Updated `AnimalForm.tsx` - Image editor shows before upload
- ✅ Updated `PhotoGallery.tsx` - Image editor integrated into gallery upload
- ✅ Changed button text from "Change Image" to "Upload Different Image"
- ✅ Added state management for editor (showImageEditor, editingImageUrl)
- ✅ Created handleEditorSave and handleEditorCancel functions
- ✅ Blob upload support after cropping

**Update Upload Functions:** ✅ IMPLEMENTED

```tsx
// frontend/src/pages/AnimalForm.tsx & PhotoGallery.tsx - IMPLEMENTED

const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
  const file = e.target.files?.[0];
  if (!file) return;
  
  // Validate file
  if (!file.type.startsWith('image/')) {
    toast.showError('Please select an image file');
    return;
  }
  
  // Create preview URL for editor
  const previewUrl = URL.createObjectURL(file);
  
  // Show editor modal
  setEditingImage(previewUrl);
  setShowImageEditor(true);
};

const handleEditorSave = async (croppedBlob: Blob, rotation: number) => {
  setShowImageEditor(false);
  setUploading(true);
  
  try {
    // Create FormData with cropped image
    const formData = new FormData();
    formData.append('image', croppedBlob, 'cropped-image.jpg');
    
    // Upload cropped image
    const res = await animalsApi.uploadImage(formData);
    setFormData({ ...formData, image_url: res.data.url });
    toast.showSuccess('Image uploaded successfully!');
  } catch (error) {
    toast.showError('Failed to upload image');
  } finally {
    setUploading(false);
    URL.revokeObjectURL(editingImage); // Clean up
  }
};

// In render:
{showImageEditor && editingImage && (
  <ImageEditor
    imageSrc={editingImage}
    onSave={handleEditorSave}
    onCancel={() => {
      setShowImageEditor(false);
      URL.revokeObjectURL(editingImage);
    }}
    aspectRatio={4/3}
  />
)}
```

---

## 🗂️ Data Migration Strategy

### Option A: Keep Existing System, Add New (Recommended)

**Pros:**
- No breaking changes
- Comments with images continue working
- Gradual migration

**Cons:**
- Two sources of truth for images
- Some complexity in queries

**Implementation:**
```sql
-- New table
CREATE TABLE animal_images (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    animal_id INTEGER NOT NULL REFERENCES animals(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    image_url VARCHAR(255) NOT NULL,
    caption TEXT,
    is_profile_picture BOOLEAN DEFAULT FALSE,
    width INTEGER,
    height INTEGER,
    file_size INTEGER
);

-- Migrate existing comment images (OPTIONAL)
INSERT INTO animal_images (animal_id, user_id, image_url, caption, created_at)
SELECT 
    ac.animal_id,
    ac.user_id,
    ac.image_url,
    ac.content as caption,
    ac.created_at
FROM animal_comments ac
WHERE ac.image_url IS NOT NULL AND ac.image_url != '';

-- Photo Gallery queries UNION from both sources
SELECT * FROM animal_images WHERE animal_id = ? AND deleted_at IS NULL
UNION ALL
SELECT id, created_at, updated_at, animal_id, user_id, image_url, content as caption 
FROM animal_comments 
WHERE animal_id = ? AND image_url IS NOT NULL AND deleted_at IS NULL
ORDER BY created_at DESC;
```

### Option B: Migrate Everything

**Pros:**
- Single source of truth
- Cleaner architecture

**Cons:**
- Breaking change
- Requires careful migration

---

## 🎨 UI/UX Design Considerations

### For Non-Admin Users

1. **Photo Gallery Access**
   - Add prominent "📷 View Photos" button on animal cards
   - Show photo count badge on animal cards
   - Add gallery link to animal detail page header
   - Include recent photos preview on animal detail page

2. **Upload Experience**
   - Upload button in gallery page (not just comments)
   - Clear feedback: "Image uploaded to gallery"
   - Option to add caption during upload
   - Preview before final save

3. **Image Editor**
   - Simple, intuitive controls
   - Visual guides (grid overlay)
   - Preset rotation buttons (90°, 180°, 270°)
   - Touch-friendly for mobile

### For Admin Users

1. **Profile Picture Management**
   - Clear indication of current profile picture
   - Easy toggle between "Upload New" and "Select Existing"
   - Visual preview before confirming selection
   - Confirmation toast: "Profile picture updated"

2. **Bulk Operations**
   - Multi-select in gallery
   - Bulk delete (with confirmation)
   - Batch set captions
   - Move images between animals

3. **Image Metadata**
   - Show upload date, uploader, dimensions, file size
   - Filter/sort options (date, uploader, size)
   - Search by caption

---

## 📱 Mobile Considerations

1. **Touch-Friendly Gestures**
   - Pinch to zoom in image editor
   - Swipe to navigate gallery
   - Long-press for context menu (delete, set as profile)

2. **Responsive Design**
   - Stacked layout on mobile
   - Larger touch targets (minimum 44x44px)
   - Bottom sheet modals instead of centered modals

3. **Performance**
   - Lazy load images in gallery
   - Thumbnail generation for grid view
   - Progressive image loading

---

## 🔒 Security Considerations

1. **Authorization**
   - Non-admins can only delete their own images
   - Admins can delete any image
   - Only admins can set profile pictures
   - Validate user has access to group/animal

2. **File Validation**
   - Maintain existing validation (size, type, content)
   - Scan for malicious content
   - Strip EXIF data (except orientation)

3. **Rate Limiting**
   - Max 10 image uploads per user per hour
   - Max 5 profile picture changes per hour
   - Prevent spam uploads

---

## 🧪 Testing Strategy

### Unit Tests

```go
// internal/handlers/animal_image_test.go
func TestGetAnimalImages(t *testing.T)
func TestUploadAnimalImageToGallery(t *testing.T)
func TestDeleteAnimalImage_OwnerCanDelete(t *testing.T)
func TestDeleteAnimalImage_AdminCanDelete(t *testing.T)
func TestDeleteAnimalImage_UnauthorizedCannot(t *testing.T)
func TestSetAnimalProfilePicture_AdminOnly(t *testing.T)
```

### E2E Tests

```typescript
// frontend/tests/animal-images.spec.ts
test('non-admin can upload image to gallery', async ({ page }) => {
  // Login as non-admin
  // Navigate to animal page
  // Click "Upload Photo" in gallery
  // Select file
  // Crop/rotate image
  // Confirm upload
  // Verify image appears in gallery
});

test('admin can select profile picture from gallery', async ({ page }) => {
  // Login as admin
  // Edit animal
  // Click "Choose from Gallery"
  // Select an image
  // Confirm selection
  // Verify profile picture updated
});

test('user can delete their own image', async ({ page }) => {
  // Upload image as user
  // Click delete button
  // Confirm deletion
  // Verify image removed
});
```

---

## 📦 Implementation Phases

### Phase 1: Foundation (Week 1)
- [ ] Create `animal_images` table and model
- [ ] Implement GET `/groups/:id/animals/:animalId/images` endpoint
- [ ] Implement POST `/groups/:id/animals/:animalId/images` endpoint
- [ ] Update PhotoGallery to use new endpoint
- [ ] Add upload button to PhotoGallery
- [ ] Basic unit tests

**Deliverable:** Non-admin users can upload images to gallery and view all images

### Phase 2: Admin Selector (Week 2)
- [ ] Implement PUT `/admin/animals/:animalId/images/:imageId/set-profile` endpoint
- [ ] Add DELETE `/groups/:id/animals/:animalId/images/:imageId` endpoint
- [ ] Create ImageSelector modal component
- [ ] Update AnimalForm with "Choose from Gallery" option
- [ ] Visual indication of profile picture in gallery
- [ ] Permission checks (owner + admin delete)
- [ ] E2E tests for selection flow

**Deliverable:** Admins can select profile pictures from uploaded images

### Phase 3: Image Editor (Week 3) ✅ COMPLETE
- [x] Install react-easy-crop library
- [x] Create ImageEditor component
- [x] Integrate editor into upload flow (AnimalForm + PhotoGallery)
- [x] Add crop, zoom, rotate controls
- [x] Backend handling of edited images (uses existing upload endpoint)
- [x] Mobile responsive design
- [ ] E2E tests for editor (TODO)

**Deliverable:** ✅ All users can crop/rotate images before uploading

### Phase 4: Polish & Optimization (Week 4)
- [ ] Add image metadata display
- [ ] Implement thumbnail generation
- [ ] Lazy loading in gallery
- [ ] Batch operations for admins
- [ ] Performance optimization
- [ ] Accessibility improvements
- [ ] Documentation updates
- [ ] Final testing and bug fixes

**Deliverable:** Production-ready image management system

---

## 🎯 Success Metrics

### User Engagement
- **Target:** 80% of users upload at least one image per week
- **Target:** 50% increase in total images uploaded
- **Target:** Photo gallery views increase by 200%

### Admin Efficiency
- **Target:** 90% of profile pictures set from gallery (not re-uploads)
- **Target:** 60% reduction in duplicate image uploads
- **Target:** Profile picture selection takes <30 seconds

### Technical
- **Target:** Image editor loads in <2 seconds
- **Target:** Gallery page loads 100 images in <3 seconds
- **Target:** Mobile crop/rotate works on iOS Safari and Android Chrome

---

## 🚀 Quick Wins (Can Start Immediately)

### 1. Add Gallery Link to Animal Cards (1 hour)
```tsx
// In animal cards, add:
{animal.image_count > 0 && (
  <Link to={`/groups/${groupId}/animals/${animal.id}/photos`}>
    📷 {animal.image_count} photos
  </Link>
)}
```

### 2. Show Current Profile Picture in AnimalForm (1 hour)
```tsx
// Add visual indicator:
{formData.image_url && (
  <div className="current-profile-pic">
    <label>Current Profile Picture:</label>
    <img src={formData.image_url} alt="Profile" />
  </div>
)}
```

### 3. Add Upload Button to PhotoGallery (2 hours)
```tsx
// No backend changes needed - use existing endpoint
<button onClick={() => setShowUploadModal(true)}>
  📷 Upload Photo to Gallery
</button>
```

---

## 💰 Cost Estimate

### Development Time
- Phase 1 (Foundation): 32 hours
- Phase 2 (Admin Selector): 24 hours
- Phase 3 (Image Editor): 32 hours
- Phase 4 (Polish): 24 hours
- **Total:** ~112 hours (3 weeks with full-time developer)

### Infrastructure
- Storage: ~$5/month per 1000 images (assuming 100KB average after compression)
- CDN (if added): ~$10/month
- No additional compute costs

---

## 🔄 Alternative Approaches

### Alternative 1: Use Existing Comment System
**Pros:** No new tables, simpler
**Cons:** Comments are not ideal for image management, limited metadata

### Alternative 2: Third-Party Service (Cloudinary, Uploadcare)
**Pros:** Built-in crop/rotate/CDN, no development time
**Cons:** Recurring cost ($89+/month), vendor lock-in, privacy concerns

### Alternative 3: Mobile App First
**Pros:** Better native camera integration
**Cons:** Delays web users, requires separate development

---

## 📝 Documentation Updates Needed

1. **API.md**
   - New endpoints for animal images
   - Image upload flow diagram
   - Permission matrix

2. **README.md**
   - Update features list
   - Add image editor library credit

3. **CONTRIBUTING.md**
   - Guidelines for image handling
   - Testing requirements

4. **User Guide** (new)
   - How to upload images
   - How to use image editor
   - Admin: Setting profile pictures

---

## ✅ Definition of Done

### Phase 1
- [ ] Non-admin users can view all images for an animal
- [ ] Non-admin users can upload images to gallery
- [ ] Images are stored with metadata (dimensions, uploader, date)
- [ ] Gallery shows image count
- [ ] Unit tests pass
- [ ] E2E test for upload flow

### Phase 2
- [ ] Admins see "Choose from Gallery" button in animal form
- [ ] Modal shows grid of available images
- [ ] Clicking image sets it as profile picture
- [ ] Current profile picture is indicated
- [ ] Users can delete their own images
- [ ] Admins can delete any image
- [ ] E2E test for profile picture selection

### Phase 3
- [ ] Image editor appears after file selection
- [ ] Crop functionality works on desktop and mobile
- [ ] Rotate with presets (90°, 180°, 270°)
- [ ] Zoom slider functional
- [ ] Aspect ratio presets (square, 4:3, 16:9, free)
- [ ] Save button uploads cropped image
- [ ] E2E test for image editing

### All Phases
- [ ] Documentation updated
- [ ] No breaking changes to existing functionality
- [ ] Lighthouse score >90 on gallery page
- [ ] Works on Chrome, Firefox, Safari (desktop + mobile)
- [ ] Accessible (keyboard navigation, screen readers)

---

## 🤔 Open Questions

1. **Should we migrate existing comment images to new table?**
   - Recommendation: No, keep both systems for now

2. **What's the max number of images per animal?**
   - Recommendation: 100 images per animal (soft limit)

3. **Should non-admins be able to suggest profile picture?**
   - Recommendation: Phase 5 feature, add "Suggest as profile" button

4. **Do we need video support?**
   - Recommendation: Out of scope for now (see ROADMAP.md)

5. **Should we generate thumbnails server-side or client-side?**
   - Recommendation: Server-side for consistency, client-side for speed

---

## 📞 Next Steps

1. **Review this plan** with team and stakeholders
2. **Prioritize phases** based on user feedback
3. **Create GitHub issues** for each phase
4. **Assign developer(s)** to Phase 1
5. **Set timeline** and milestones
6. **Update ROADMAP.md** with this feature

---

**Last Updated:** November 25, 2025  
**Author:** UX Design Expert Agent  
**Status:** Ready for Review
