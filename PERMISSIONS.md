# Go Volunteer Media - User Permissions & Access Control

**Last Updated:** December 2, 2025

This document defines the permission levels and capabilities for each user role in the Go Volunteer Media platform.

---

## Permission Levels Overview

The platform has three distinct permission levels:

1. **Regular User** - Basic member access
2. **Group Admin** - Administrative access for a specific group
3. **Site Admin** - Full platform-wide administrative access

---

## 1. Regular User

**Who:** Standard volunteers who are members of one or more groups.

### ✅ Permissions

#### Animal Management
- **View** animals in groups they're a member of
- **View** detailed animal profiles and photos
- **View** animal status (available, adopted, etc.)
- **View** animal tags and attributes

#### Comments & Updates
- **Create** comments on animals
- **Edit** their own comments
- **Delete** their own comments
- **Tag** comments with relevant categories
- **Create** updates about animals they've interacted with
- **Upload** photos to comments

#### Group Interaction
- **View** group information and details
- **View** activity feed (comments, announcements)
- **View** protocols (if enabled for the group)
- **View** group announcements
- **Switch** between groups they're a member of

#### Profile Management
- **View** their own profile
- **View** other users' profiles (username, email, phone number only if shared)
- **Update** their own settings (optional to share email/phone)
- **Set** default group preference
- **Update** notification preferences
- **Change** password

### ❌ Restrictions

- Cannot add, edit, or delete animals
- Cannot create, edit, or delete animal tags
- Cannot create, edit, or delete protocols
- Cannot post announcements
- Cannot manage group members
- Cannot access admin features
- Cannot view deleted content
- Cannot restore deleted items
- Cannot change user permissions

---

## 2. Group Admin

**Who:** Users with administrative privileges for a specific group. They manage day-to-day operations for their group but cannot affect other groups or site-wide settings.

### ✅ Permissions

#### Everything Regular Users Can Do, PLUS:

#### Animal Management
- **Create** new animals in their group
- **Edit** any animal in their group (name, breed, age, description, etc.)
- **Delete** animals in their group
- **Update** animal status (available, adopted, quarantine, etc.)
- **Upload** and edit animal photos
- **Bulk edit** animals in their group (via CSV import/export)
- **Mark** animals as returned
- **Manage** animal quarantine dates
- **Assign/Remove** tags to/from animals

#### Tag Management (Group-Specific)
- **View** all available animal tags
- **Assign** tags to animals in their group
- **Remove** tags from animals in their group
- **Create:** Create group specific tags
- **Edit:** Edit group specific tags
- **Delete:** Delete group specific tags

#### Protocol Management
- **Create** new protocols for their group
- **Edit** protocols in their group
- **Delete** protocols in their group
- **Upload** protocol images
- **Reorder** protocols (via order_index)

#### Announcements
- **Create** announcements for their group
- **Send** email notifications to group members
- **Integrate** with GroupMe (if configured)

#### Content Moderation
- **View** all comments in their group
- **Delete** any comment in their group
- **View** deleted comments (admin view)
- **Restore** deleted comments (if needed)
- **Moderate** inappropriate content

#### Member Management
- **View** group members list
- **View** member activity and statistics
- **Create:** new members added to the group
- **Remove:** members from the group
- **Promote:** members to group admin
- **Demote:** group admins to regular members

#### Group Settings
- **View** group settings
- **Edit** group details (name, description, images)
- **Toggle** protocol feature for the group
- **Configure** GroupMe integration settings

### ❌ Restrictions

- Cannot create or delete animal tags (site-wide)
- Cannot access other groups' data (unless also a member)
- Cannot access site-wide admin dashboard
- Cannot manage users across the platform
- Cannot view or manage deleted animals
- Cannot access site settings
- Cannot create or delete groups

### 🎯 Use Cases for Group Admin

**Typical workflows:**
1. **Daily Animal Care:** Add new animals, update statuses, add photos
2. **Protocol Management:** Maintain care instructions for volunteers
3. **Communication:** Post announcements about urgent needs or updates
4. **Content Moderation:** Remove inappropriate comments
5. **Data Management:** Export animal data for reports

---

## 3. Site Admin

**Who:** Platform administrators with full access to all groups and site-wide settings.

### ✅ Permissions

#### Everything Group Admins Can Do (for ALL groups), PLUS:

#### User Management
- **View** all users across the platform
- **Create** new user accounts
- **Edit** user information (email, username)
- **Delete** users (soft delete)
- **Restore** deleted users
- **Reset** user passwords (admin-initiated)
- **View** user activity logs
- **View** user email addresses

#### Group Management
- **Create** new groups
- **Edit** any group’s settings (name, description, images)
- **Delete** groups
- **Toggle** protocol feature for any group
- **Configure** GroupMe integration for any group
- **Add/Remove** users to/from any group
- **Promote/Demote** group admins across all groups

#### Tag Management

- **Manage site-wide tags**: create, edit, delete (affects all groups)
- **Manage group-specific tags across all groups**
- **View** tag usage and adoption statistics

#### Site Settings

- **Access** admin dashboard with analytics
- **View** platform-wide statistics
- **Configure** site-wide settings
- **Manage** default data (tags, groups, etc.)

#### Advanced Features

- **Bulk edit** animals across all groups
- **View** all deleted content (comments, images, animals)
- **Restore** deleted content
- **Access** audit logs
- **View** system health metrics

#### Security & Compliance

- **Manage** account lockouts
- **View** rate limiting logs
- **Monitor** failed login attempts
- **Access** security settings

### ⚠️ Important Notes for Site Admins

**With great power comes great responsibility:**
- Site admins can access ALL data across ALL groups
- Site-wide tag changes propagate to every group
- Deleting groups or users has platform-wide impact
- Always verify group context before bulk operations
- Prefer least-privilege workflows; defer to group admins where possible

---

## Permission Matrix

| Feature | Regular User | Group Admin | Site Admin |
|---------|-------------|-------------|------------|
| **Animals** |
| View animals | ✅ (own groups) | ✅ (own groups) | ✅ (all groups) |
| Create animals | ❌ | ✅ (own groups) | ✅ (all groups) |
| Edit animals | ❌ | ✅ (own groups) | ✅ (all groups) |
| Delete animals | ❌ | ✅ (own groups) | ✅ (all groups) |
| Bulk edit animals | ❌ | ✅ (own groups) | ✅ (all groups) |
| **Tags** |
| View tags | ✅ | ✅ | ✅ |
| Assign tags to animals | ❌ | ✅ (own groups) | ✅ (all groups) |
| Create/edit/delete group-specific tags | ❌ | ✅ (own groups) | ✅ (all groups) |
| Create/edit/delete site-wide tags | ❌ | ❌ | ✅ |
| **Protocols** |
| View protocols | ✅ | ✅ | ✅ |
| Create protocols | ❌ | ✅ (own groups) | ✅ (all groups) |
| Edit protocols | ❌ | ✅ (own groups) | ✅ (all groups) |
| Delete protocols | ❌ | ✅ (own groups) | ✅ (all groups) |
| **Comments** |
| Create comments | ✅ | ✅ | ✅ |
| Edit own comments | ✅ | ✅ | ✅ |
| Delete own comments | ✅ | ✅ | ✅ |
| Delete any comment | ❌ | ✅ (own groups) | ✅ (all groups) |
| View deleted comments | ❌ | ✅ (own groups) | ✅ (all groups) |
| **Announcements** |
| View announcements | ✅ | ✅ | ✅ |
| Create announcements | ❌ | ✅ (own groups) | ✅ (all groups) |
| **Groups** |
| View group info | ✅ (member) | ✅ (member) | ✅ (all) |
| Edit group settings | ❌ | ✅ (own groups) | ✅ (all) |
| Create groups | ❌ | ❌ | ✅ |
| Delete groups | ❌ | ❌ | ✅ |
| **Users** |
| View user profiles | ✅ (limited, if shared) | ✅ (group members) | ✅ (full) |
| View member list | ✅ | ✅ (own groups) | ✅ (all) |
| Add/remove members | ❌ | ✅ (own groups) | ✅ (all) |
| Promote/demote admins | ❌ | ✅ (own groups) | ✅ (all) |
| Reset passwords | ❌ | ❌ | ✅ |
| Delete users | ❌ | ❌ | ✅ |
| **Admin Dashboard** |
| Access admin dashboard | ❌ | ❌ | ✅ |
| View analytics | ❌ | ❌ | ✅ |
| Site settings | ❌ | ❌ | ✅ |

---

## API Permission Enforcement

### Backend Validation

All permissions are enforced at the API level using middleware and handler-level checks:

#### Regular Users
- Routes: `/api/groups/:id/*` - Requires group membership
- Check: `checkGroupAccess()` - Verifies user is a member

#### Group Admins

- Routes: `/api/groups/:id/animals/*`, `/api/groups/:id/protocols/*` (and group management endpoints for members/settings)
- Check: `checkGroupAdminAccess()` - Verifies user is group admin OR site admin
- Returns `403 Forbidden` if unauthorized

#### Site Admins
- Routes: `/api/admin/*`
- Check: `middleware.RequireAdmin()` - Verifies `is_admin` flag
- Returns `403 Forbidden` if unauthorized

### Frontend Validation

The frontend also enforces permissions for better UX:

#### Route Guards
- `PrivateRoute` - Requires authentication
- `GroupAdminRoute` - Requires group admin OR site admin for specific group
- `AdminRoute` - Requires site admin

#### Component-Level Checks
- `isAdmin` - Site admin check (from `useAuth()`)
- `membership.is_group_admin` - Group admin check for specific group
- `canEdit` - Combined check (site admin OR group admin)

---

## Security Considerations

### 1. Defense in Depth
- **Frontend** - Hide UI elements users can't access (UX)
- **Routes** - Redirect unauthorized access attempts
- **API** - Enforce permissions on every request (SECURITY)

### 2. Group Context Validation
Group admins can only act within their assigned groups:
```go
// Backend always validates groupID matches user's permissions
if !checkGroupAdminAccess(db, userID, isAdmin, groupID) {
    return 403 Forbidden
}
```

### 3. Site Admin Privileges
Site admins bypass group-specific checks but should:
- Act responsibly with platform-wide access
- Verify group context before bulk operations
- Use admin dashboard for cross-group management

### 4. Privacy Protection

- Regular users can only see other users’ contact info if shared by that user
- Group admins can view shared contact info for members of their group
- Site admins can access all user contact information when needed for support/compliance
- Deleted content is visible to group admins (their groups) and site admins

---

## Permission Changes & Migrations

When adding new permissions:

1. **Update backend handlers** - Add permission checks
2. **Update API routes** - Apply appropriate middleware
3. **Update frontend components** - Add UI permission checks
4. **Update route guards** - Restrict access appropriately
5. **Update this document** - Keep permissions documented
6. **Write tests** - Verify permission enforcement
7. **Update API.md** - Document API permission requirements

---

## Testing Permissions

### Test Accounts

Use the pre-created accounts from the database seed data:

- **Site Admin**: `admin`
- **Group Admin**: `merry` (ModSquad group admin)
- **Regular User**: `testuser`

Passwords are defined in the seed scripts. Do not hardcode or publish passwords here; refer to `internal/database/seed.go` or run the seed process to provision accounts.

### Test Scenarios

**Group Admin Testing:**
1. Login as `merry`
2. Verify quick links show "Add Animal" and "View Protocols"
3. Create a new animal in ModSquad
4. Create a protocol in ModSquad
5. Verify cannot access site admin routes (`/admin/*`)
6. Verify cannot access other groups' animals

**Regular User Testing:**
1. Login as `testuser`
2. Verify can view animals
3. Verify can add comments
4. Verify cannot see "Add Animal" button
5. Verify cannot access admin routes

---

## Troubleshooting Permission Issues

### Issue: Group Admin Can't Access Feature

**Check:**
1. Is `membership.is_group_admin` true for this group?
2. Is the route using `GroupAdminRoute` or `AdminRoute`?
3. Does the API endpoint use `checkGroupAdminAccess()`?
4. Is the groupId being passed correctly?

### Issue: Site Admin Can't Access Feature

**Check:**
1. Is `user.is_admin` true in database?
2. Is the route using `AdminRoute` wrapper?
3. Is the API endpoint protected with `middleware.RequireAdmin()`?

### Issue: Regular User Can Access Admin Feature

**Check:**
1. Is there a route guard missing?
2. Is the API endpoint properly protected?
3. Is the UI conditionally rendering based on permissions?

---

## Related Documentation

- **API.md** - API endpoints and their permission requirements
- **ARCHITECTURE.md** - System design and permission flow
- **SETUP.md** - How to create test accounts with different permissions
- **development-workflow.instructions.md** - Permission checks during development

---

*For questions about permissions or to request new permission levels, please create a GitHub issue.*
