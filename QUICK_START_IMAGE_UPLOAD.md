# Quick Start: Image Upload Feature

## What Changed

### Backend (`internal/handlers/animal.go`)
- ✅ Added `UploadAnimalImage()` handler with logging
- ✅ Validates file types (.jpg, .jpeg, .png, .gif only)
- ✅ Generates unique filenames to prevent overwrites
- ✅ Saves files to `public/uploads/` directory
- ✅ Returns public URL for uploaded image

### Routes (`cmd/api/main.go`)
- ✅ Added protected route: `POST /api/animals/upload-image`
- ✅ Added static serving: `GET /uploads/<filename>`
- ✅ Requires authentication (JWT token)

### Frontend (`frontend/src/api/client.ts`)
- ✅ Added `animalsApi.uploadImage(file)` method
- ✅ Properly sends FormData with file
- ✅ Axios automatically sets correct Content-Type header

### Frontend (`frontend/src/pages/AnimalForm.tsx`)
- ✅ Added file input for image uploads
- ✅ Shows preview of uploaded image
- ✅ Better error handling and user feedback
- ✅ Clears input after upload attempt

## How to Use (User Perspective)

1. **Log in** to the application
2. Navigate to **Add Animal** or **Edit Animal**
3. Scroll to the image section
4. You now have **two options**:
   - Enter an image URL directly (old method)
   - Click "Choose File" to upload an image (new method)
5. When you upload a file:
   - Select a PNG, JPG, or GIF
   - Wait for "Image uploaded successfully!" message
   - Preview appears below the input
   - The URL is automatically set in the form
6. Click **Save Animal** to save with the image

## How to Test

### Via Frontend (Recommended)
1. Start both servers: `make run` (backend) and `cd frontend && npm run dev` (frontend)
2. Log in to the app
3. Try uploading a PNG image in the Animal form
4. Check that the image preview appears
5. Save the animal and verify the image displays on the group page

### Via curl (Advanced)
```bash
# Get a JWT token by logging in
TOKEN="your-jwt-token-here"

# Upload an image
curl -X POST http://localhost:8080/api/animals/upload-image \
  -H "Authorization: Bearer $TOKEN" \
  -F "image=@/path/to/your/image.png"

# Should return: {"url":"/uploads/1234567890_uuid.png"}
```

## Troubleshooting

### "Failed to upload image"
1. Check browser console for detailed error
2. Check backend logs for the actual error message
3. Verify you're logged in (check network tab for 401 errors)

### Image doesn't display
1. Check the URL in formData.image_url
2. Verify the file exists in `public/uploads/`
3. Check browser network tab for 404 errors on the image request

### Permission errors
```bash
# Make sure the directory is writable
chmod 755 public/uploads/
```

## Files Modified
- ✅ `internal/handlers/animal.go` - Added upload handler with logging
- ✅ `cmd/api/main.go` - Added routes and static serving
- ✅ `frontend/src/api/client.ts` - Added upload API method
- ✅ `frontend/src/pages/AnimalForm.tsx` - Added file input UI
- ✅ `.gitignore` - Ignore uploaded files
- ✅ `public/uploads/.gitkeep` - Track directory structure

## Next Steps
- Test uploading different file types (PNG, JPG, GIF)
- Test uploading larger files
- Test the preview functionality
- Verify images display correctly on the group page
