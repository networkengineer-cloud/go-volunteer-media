# Comment History Implementation Plan

## Status: Phase 1 Complete (Edit Functionality)

This document outlines the implementation plan for comment history tracking, similar to how deleted comments are preserved.

## Phase 1: Edit Functionality ✅ (Current Branch)

### Frontend Implementation
- [x] Edit UI with pencil icon (✏️) for comment owners
- [x] Inline editing using SessionReportForm component
- [x] Pre-populate form with existing comment data (quick & structured)
- [x] Edit indicator badge showing "(edited)" with last edit timestamp
- [x] Cancel button to exit edit mode
- [x] Visual feedback with brand-colored border during editing
- [x] Support for editing both quick comments and structured session reports
- [x] Update API client with `update` method

### Backend Implementation
- [x] `UpdateAnimalComment` handler in `internal/handlers/animal_comment.go`
- [x] Permission check: only comment owners can edit
- [x] GORM automatic `UpdatedAt` timestamp tracking
- [x] Support for updating tags alongside content
- [x] Route registered: `PUT /api/groups/:id/animals/:animalId/comments/:commentId`

### Current Limitations
- ❌ No history tracking - original content is overwritten
- ❌ No audit trail for edits
- ❌ Admins cannot view edit history

## Phase 2: Comment History Tracking (Next)

### Design Pattern
Follow the same pattern as deleted comments:
- Deleted comments: Use `DeletedAt` field (soft delete) + `GetDeletedComments` endpoint for admins
- **Edited comments: Use `CommentHistory` table + `GetCommentHistory` endpoint for admins**

### Backend Tasks

#### 1. Database Model
```go
// internal/models/models.go
type CommentHistory struct {
    ID        uint             `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time        `gorm:"index:idx_comment_history_comment" json:"created_at"`
    CommentID uint             `gorm:"not null;index:idx_comment_history_comment" json:"comment_id"`
    Content   string           `gorm:"not null" json:"content"`
    ImageURL  string           `json:"image_url"`
    Metadata  *SessionMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`
    EditedBy  uint             `gorm:"not null" json:"edited_by"` // User ID who made the edit
    User      User             `gorm:"foreignKey:EditedBy" json:"user,omitempty"`
}
```

#### 2. Migration
- Add to `RunMigrations` in `internal/database/database.go`
- GORM will auto-create the table on next startup

#### 3. Update Handler Logic
Modify `UpdateAnimalComment` in `internal/handlers/animal_comment.go`:
```go
// Before updating (line ~291), save current version to history
history := models.CommentHistory{
    CommentID: comment.ID,
    Content:   comment.Content,
    ImageURL:  comment.ImageURL,
    Metadata:  comment.Metadata,
    EditedBy:  userID.(uint),
}
if err := db.Create(&history).Error; err != nil {
    // Log error but don't fail the update
    log.Printf("Failed to save comment history: %v", err)
}

// Then proceed with update
comment.Content = req.Content
// ... rest of update logic
```

#### 4. New API Endpoint
```go
// internal/handlers/animal_comment.go
func GetCommentHistory(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        groupID := c.Param("id")
        animalID := c.Param("animalId")
        commentID := c.Param("commentId")
        userID, _ := c.Get("user_id")
        isAdmin, _ := c.Get("is_admin")

        // Check for group admin or site admin access
        if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
            c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
            return
        }

        // Verify comment belongs to group
        var comment models.AnimalComment
        err := db.Joins("JOIN animals ON animals.id = animal_comments.animal_id").
            Where("animal_comments.id = ? AND animals.group_id = ?", commentID, groupID).
            First(&comment).Error
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
            return
        }

        // Get history entries
        var history []models.CommentHistory
        err = db.Where("comment_id = ?", commentID).
            Preload("User").
            Order("created_at DESC").
            Find(&history).Error
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
            return
        }

        c.JSON(http.StatusOK, history)
    }
}
```

#### 5. Route Registration
Add to `cmd/api/main.go` in the animal comments section:
```go
animals.GET("/:animalId/comments/:commentId/history", handlers.GetCommentHistory(db))
```

### Frontend Tasks

#### 1. API Client Update
```typescript
// frontend/src/api/client.ts
export interface CommentHistory {
  id: number;
  created_at: string;
  comment_id: number;
  content: string;
  image_url: string;
  metadata?: SessionMetadata;
  edited_by: number;
  user?: User;
}

export const animalCommentsApi = {
  // ... existing methods
  getHistory: (groupId: number, animalId: number, commentId: number) =>
    api.get<CommentHistory[]>(`/groups/${groupId}/animals/${animalId}/comments/${commentId}/history`),
};
```

#### 2. Comment History Modal
Create `frontend/src/components/CommentHistoryModal.tsx`:
- Display list of previous versions
- Show edit timestamp, editor name, and full content
- Side-by-side diff view (optional enhancement)
- Only visible to group admins and site admins

#### 3. UI Integration
Update `frontend/src/pages/AnimalDetailPage.tsx`:
- Add "View History" button for admins (next to edit/delete)
- Only show if comment has been edited (created_at !== updated_at)
- Modal opens on click, loads history via API

#### 4. Styling
Add to `frontend/src/pages/AnimalDetailPage.css`:
- History button styles
- Modal styles for history viewer
- Diff highlighting (if implementing comparison view)

### Testing Tasks

#### Backend Tests
- [ ] Test comment history is saved on edit
- [ ] Test permission: only group admins can view history
- [ ] Test permission: only site admins can view all history
- [ ] Test history includes user info and timestamps
- [ ] Test history is returned in chronological order

#### Frontend Tests
- [ ] Test history button only visible to admins
- [ ] Test history button only appears for edited comments
- [ ] Test modal opens and displays history
- [ ] Test modal displays formatted content correctly

### Documentation Updates
- [ ] Update API.md with new endpoint
- [ ] Update PERMISSIONS.md (comment history viewing requires admin access)
- [ ] Add comment to code explaining the history pattern

## Implementation Order

1. **Backend Model & Migration** (5 min)
   - Add CommentHistory model
   - Add to migrations
   - Test database migration

2. **Backend History Saving** (10 min)
   - Update UpdateAnimalComment handler
   - Add history creation before update
   - Test history is saved correctly

3. **Backend History Endpoint** (15 min)
   - Implement GetCommentHistory handler
   - Add route registration
   - Test with curl/Postman

4. **Frontend API Integration** (5 min)
   - Add types and API method
   - Test API call

5. **Frontend UI** (20 min)
   - Create CommentHistoryModal component
   - Add "View History" button to comments
   - Test modal display

6. **Testing & Documentation** (20 min)
   - Write backend tests
   - Update documentation
   - Manual end-to-end testing

**Total Estimated Time: ~75 minutes**

## Security Considerations

- ✅ History viewing requires group admin or site admin access
- ✅ History includes editor identity for accountability
- ✅ Original content is never lost
- ✅ Audit trail for compliance and moderation

## Future Enhancements (Optional)

- [ ] Diff view showing changes between versions
- [ ] Ability to restore previous version
- [ ] Comment edit notifications to group admins
- [ ] Retention policy for old history entries
- [ ] Export comment history in CSV format

---

**Created:** February 14, 2026  
**Phase 1 Completed:** February 14, 2026  
**Phase 2 Status:** Not started
