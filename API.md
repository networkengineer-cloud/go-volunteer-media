# API Documentation

This document describes all available API endpoints in the Haws Volunteers application.

## Base URL

Development: `http://localhost:8080/api`
Production: `https://your-domain.com/api`

## Authentication

Most endpoints require authentication using JWT tokens. Include the token in the Authorization header:

```
Authorization: Bearer <your-token>
```

## Endpoints

### Authentication

#### Register

Create a new user account.

**Endpoint:** `POST /register`

**Request Body:**
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "secure_password123"
}
```

**Response:** `201 Created`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "created_at": "2025-10-10T23:37:45.305782393Z",
    "updated_at": "2025-10-10T23:37:45.305782393Z",
    "username": "john_doe",
    "email": "john@example.com",
    "is_admin": false
  }
}
```

**Errors:**
- `400 Bad Request`: Invalid request body
- `409 Conflict`: Username or email already exists

#### Login

Authenticate and receive a JWT token.

**Endpoint:** `POST /login`

**Request Body:**
```json
{
  "username": "john_doe",
  "password": "secure_password123"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "created_at": "2025-10-10T23:37:45.305782393Z",
    "updated_at": "2025-10-10T23:37:45.305782393Z",
    "username": "john_doe",
    "email": "john@example.com",
    "is_admin": false,
    "groups": [
      {
        "id": 1,
        "name": "dogs",
        "description": "Dog volunteers group"
      }
    ]
  }
}
```

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid credentials

#### Get Current User

Get information about the currently authenticated user.

**Endpoint:** `GET /me`

**Authentication:** Required

**Response:** `200 OK`
```json
{
  "id": 1,
  "created_at": "2025-10-10T23:37:45.305782393Z",
  "updated_at": "2025-10-10T23:37:45.305782393Z",
  "username": "john_doe",
  "email": "john@example.com",
  "is_admin": false,
  "groups": [
    {
      "id": 1,
      "name": "dogs",
      "description": "Dog volunteers group"
    }
  ]
}
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `404 Not Found`: User not found

---

### Groups

#### List Groups

Get all groups the user has access to. Admins see all groups, regular users see only their groups.

**Endpoint:** `GET /groups`

**Authentication:** Required

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "created_at": "2025-10-10T23:35:47.124134Z",
    "updated_at": "2025-10-10T23:35:47.124134Z",
    "name": "dogs",
    "description": "Dog volunteers group"
  },
  {
    "id": 2,
    "created_at": "2025-10-10T23:35:47.128216Z",
    "updated_at": "2025-10-10T23:35:47.128216Z",
    "name": "cats",
    "description": "Cat volunteers group"
  }
]
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `500 Internal Server Error`: Database error

#### Get Group

Get details of a specific group.

**Endpoint:** `GET /groups/:id`

**Authentication:** Required

**URL Parameters:**
- `id` (integer): Group ID

**Response:** `200 OK`
```json
{
  "id": 1,
  "created_at": "2025-10-10T23:35:47.124134Z",
  "updated_at": "2025-10-10T23:35:47.124134Z",
  "name": "dogs",
  "description": "Dog volunteers group"
}
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User not in group
- `404 Not Found`: Group not found

#### Create Group (Admin Only)

Create a new group.

**Endpoint:** `POST /admin/groups`

**Authentication:** Required (Admin)

**Request Body:**
```json
{
  "name": "rabbits",
  "description": "Rabbit volunteers group"
}
```

**Response:** `201 Created`
```json
{
  "id": 4,
  "created_at": "2025-10-10T23:40:00.000000Z",
  "updated_at": "2025-10-10T23:40:00.000000Z",
  "name": "rabbits",
  "description": "Rabbit volunteers group"
}
```

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not admin
- `500 Internal Server Error`: Database error

#### Update Group (Admin Only)

Update an existing group.

**Endpoint:** `PUT /admin/groups/:id`

**Authentication:** Required (Admin)

**URL Parameters:**
- `id` (integer): Group ID

**Request Body:**
```json
{
  "name": "rabbits",
  "description": "Updated description"
}
```

**Response:** `200 OK`
```json
{
  "id": 4,
  "created_at": "2025-10-10T23:40:00.000000Z",
  "updated_at": "2025-10-10T23:41:00.000000Z",
  "name": "rabbits",
  "description": "Updated description"
}
```

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not admin
- `404 Not Found`: Group not found

#### Delete Group (Admin Only)

Delete a group.

**Endpoint:** `DELETE /admin/groups/:id`

**Authentication:** Required (Admin)

**URL Parameters:**
- `id` (integer): Group ID

**Response:** `200 OK`
```json
{
  "message": "Group deleted successfully"
}
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not admin
- `500 Internal Server Error`: Database error

---

### User-Group Management (Admin Only)

#### Add User to Group

Add a user to a group.

**Endpoint:** `POST /admin/users/:userId/groups/:groupId`

**Authentication:** Required (Admin)

**URL Parameters:**
- `userId` (integer): User ID
- `groupId` (integer): Group ID

**Response:** `200 OK`
```json
{
  "message": "User added to group successfully"
}
```

**Errors:**
- `400 Bad Request`: Invalid user or group ID
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not admin
- `404 Not Found`: User or group not found

#### Remove User from Group

Remove a user from a group.

**Endpoint:** `DELETE /admin/users/:userId/groups/:groupId`

**Authentication:** Required (Admin)

**URL Parameters:**
- `userId` (integer): User ID
- `groupId` (integer): Group ID

**Response:** `200 OK`
```json
{
  "message": "User removed from group successfully"
}
```

**Errors:**
- `400 Bad Request`: Invalid user or group ID
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not admin
- `404 Not Found`: User or group not found

---

### Group Admin Management (Site Admin Only)

Group admins are users who have admin privileges for a specific group. They can manage animals, protocols, and other group-related content without being site-wide admins.

#### Get Group Members

Get all members of a group with their admin status.

**Endpoint:** `GET /groups/:id/members`

**Authentication:** Required (Site Admin, Group Admin, or Group Member)

**URL Parameters:**
- `id` (integer): Group ID

**Response:** `200 OK`
```json
[
  {
    "user_id": 1,
    "username": "john_doe",
    "email": "john@example.com",
    "is_group_admin": true,
    "is_site_admin": false
  },
  {
    "user_id": 2,
    "username": "jane_doe",
    "email": "jane@example.com",
    "is_group_admin": false,
    "is_site_admin": false
  }
]
```

**Errors:**
- `400 Bad Request`: Invalid group ID
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not a member of the group

#### Add Member to Group

Add a user to a group (Group Admin or Site Admin only).

**Endpoint:** `POST /groups/:id/members/:userId`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `userId` (integer): User ID to add

**Response:** `200 OK`
```json
{
  "message": "User added to group successfully"
}
```

**Errors:**
- `400 Bad Request`: Invalid user or group ID, or user already in group
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not a site admin or group admin for this group
- `404 Not Found`: User or group not found

#### Remove Member from Group

Remove a user from a group (Group Admin or Site Admin only).

**Endpoint:** `DELETE /groups/:id/members/:userId`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `userId` (integer): User ID to remove

**Response:** `200 OK`
```json
{
  "message": "User removed from group successfully"
}
```

**Errors:**
- `400 Bad Request`: Invalid user or group ID, or user not in group
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not a site admin or group admin for this group
- `404 Not Found`: User or group not found

#### Promote Member to Group Admin

Promote a user to group admin for a specific group.

**Endpoint:** `POST /groups/:id/members/:userId/promote`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `userId` (integer): User ID

**Response:** `200 OK`
```json
{
  "message": "User promoted to group admin"
}
```

**Errors:**
- `400 Bad Request`: Invalid user or group ID, user not a member of group, or user already a group admin
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin for this group
- `404 Not Found`: User or group not found

#### Demote Member from Group Admin

Remove group admin privileges from a user for a specific group.

**Endpoint:** `POST /groups/:id/members/:userId/demote`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `userId` (integer): User ID

**Response:** `200 OK`
```json
{
  "message": "User demoted from group admin"
}
```

**Errors:**
- `400 Bad Request`: Invalid user or group ID, user not a member of group, or user not a group admin
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin for this group
- `404 Not Found`: User or group not found

#### Update Group Settings

Update settings for a group (Group Admin or Site Admin only).

**Endpoint:** `PUT /groups/:id/settings`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID

**Request Body:**
```json
{
  "name": "Group Name",
  "description": "Group description",
  "image_url": "/uploads/group-image.jpg",
  "hero_image_url": "/uploads/hero.jpg",
  "has_protocols": true,
  "groupme_bot_id": "",
  "groupme_enabled": false
}
```

**Response:** `200 OK`
Returns the updated group object.

**Errors:**
- `400 Bad Request`: Invalid request body or group ID
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin for this group
- `404 Not Found`: Group not found

#### Create Group Announcement

Create an announcement for a specific group with optional email and GroupMe notifications.

**Endpoint:** `POST /groups/:id/announcements`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID

**Request Body:**
```json
{
  "title": "Important Update",
  "content": "This is an important announcement for the group.",
  "send_email": true,
  "send_groupme": true
}
```

**Response:** `201 Created`
Returns the created announcement object.

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin for this group
- `404 Not Found`: Group not found

---

### Content Moderation

#### View Deleted Comments

View soft-deleted comments for a group (Group Admin or Site Admin only).

**Endpoint:** `GET /groups/:id/deleted-comments`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID

**Response:** `200 OK`
Returns a list of deleted comments with animal information.

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin for this group

#### View Deleted Images

View soft-deleted images for a group (Group Admin or Site Admin only).

**Endpoint:** `GET /groups/:id/deleted-images`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID

**Response:** `200 OK`
Returns a list of deleted images.

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin for this group

---

### Comments

#### Update Comment

Update a comment on an animal (can only edit your own comments).

**Endpoint:** `PUT /groups/:id/animals/:animalId/comments/:commentId`

**Authentication:** Required

**URL Parameters:**
- `id` (integer): Group ID
- `animalId` (integer): Animal ID
- `commentId` (integer): Comment ID

**Request Body:**
```json
{
  "content": "Updated comment content",
  "image_url": "/uploads/image.jpg",
  "tag_ids": [1, 2]
}
```

**Response:** `200 OK`
Returns the updated comment.

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not the comment owner or not a member of the group
- `404 Not Found`: Animal or comment not found

---

### Animals

#### List Animals

Get all animals in a group.

**Endpoint:** `GET /groups/:id/animals`

**Authentication:** Required

**URL Parameters:**
- `id` (integer): Group ID

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "created_at": "2025-10-10T23:38:12.410610268Z",
    "updated_at": "2025-10-10T23:38:12.410610268Z",
    "group_id": 1,
    "name": "Buddy",
    "species": "Dog",
    "breed": "Golden Retriever",
    "age": 3,
    "description": "Friendly and energetic dog",
    "image_url": "https://example.com/buddy.jpg",
    "status": "available"
  }
]
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User not in group
- `500 Internal Server Error`: Database error

#### Get Animal

Get details of a specific animal.

**Endpoint:** `GET /groups/:id/animals/:animalId`

**Authentication:** Required

**URL Parameters:**
- `id` (integer): Group ID
- `animalId` (integer): Animal ID

**Response:** `200 OK`
```json
{
  "id": 1,
  "created_at": "2025-10-10T23:38:12.410610268Z",
  "updated_at": "2025-10-10T23:38:12.410610268Z",
  "group_id": 1,
  "name": "Buddy",
  "species": "Dog",
  "breed": "Golden Retriever",
  "age": 3,
  "description": "Friendly and energetic dog",
  "image_url": "https://example.com/buddy.jpg",
  "status": "available"
}
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User not in group
- `404 Not Found`: Animal not found

#### Create Animal

Create a new animal in a group.

**Endpoint:** `POST /groups/:id/animals`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID

**Request Body:**
```json
{
  "name": "Buddy",
  "species": "Dog",
  "breed": "Golden Retriever",
  "age": 3,
  "description": "Friendly and energetic dog",
  "image_url": "https://example.com/buddy.jpg",
  "status": "available"
}
```

**Response:** `201 Created`
```json
{
  "id": 1,
  "created_at": "2025-10-10T23:38:12.410610268Z",
  "updated_at": "2025-10-10T23:38:12.410610268Z",
  "group_id": 1,
  "name": "Buddy",
  "species": "Dog",
  "breed": "Golden Retriever",
  "age": 3,
  "description": "Friendly and energetic dog",
  "image_url": "https://example.com/buddy.jpg",
  "status": "available"
}
```

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User not in group
- `500 Internal Server Error`: Database error

#### Update Animal

Update an existing animal.

**Endpoint:** `PUT /groups/:id/animals/:animalId`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `animalId` (integer): Animal ID

**Request Body:**
```json
{
  "name": "Buddy",
  "species": "Dog",
  "breed": "Golden Retriever",
  "age": 4,
  "description": "Very friendly and energetic dog",
  "image_url": "https://example.com/buddy-new.jpg",
  "status": "adopted"
}
```

**Response:** `200 OK`
```json
{
  "id": 1,
  "created_at": "2025-10-10T23:38:12.410610268Z",
  "updated_at": "2025-10-10T23:39:00.000000Z",
  "group_id": 1,
  "name": "Buddy",
  "species": "Dog",
  "breed": "Golden Retriever",
  "age": 4,
  "description": "Very friendly and energetic dog",
  "image_url": "https://example.com/buddy-new.jpg",
  "status": "adopted"
}
```

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User not in group
- `404 Not Found`: Animal not found

#### Delete Animal

Delete an animal.

**Endpoint:** `DELETE /groups/:id/animals/:animalId`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `animalId` (integer): Animal ID

**Response:** `200 OK`
```json
{
  "message": "Animal deleted successfully"
}
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin
- `500 Internal Server Error`: Database error

#### Upload Protocol Document

Upload a protocol document (PDF or DOCX) for an animal.

**Endpoint:** `POST /groups/:id/animals/:animalId/protocol-document`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `animalId` (integer): Animal ID

**Request Body:** `multipart/form-data`
- `document` (file): Protocol document file (PDF or DOCX, max 20MB)

**Response:** `200 OK`
```json
{
  "url": "/api/documents/550e8400-e29b-41d4-a716-446655440000",
  "name": "protocol.pdf",
  "size": 1048576,
  "type": "application/pdf",
  "uploaded_by": 5
}
```

**Errors:**
- `400 Bad Request`: Invalid file type or size
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin
- `404 Not Found`: Animal not found
- `500 Internal Server Error`: Upload failed

#### View Protocol Document

Get the protocol document for an animal.

**Endpoint:** `GET /documents/:uuid`

**Authentication:** Required

**URL Parameters:**
- `uuid` (string): Document UUID from the document URL

**Response:** `200 OK`
- Returns the document file with appropriate `Content-Type` header
- `Content-Disposition: inline; filename="protocol.pdf"`

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `404 Not Found`: Document not found

#### Delete Protocol Document

Remove the protocol document from an animal.

**Endpoint:** `DELETE /groups/:id/animals/:animalId/protocol-document`

**Authentication:** Required (Site Admin or Group Admin)

**URL Parameters:**
- `id` (integer): Group ID
- `animalId` (integer): Animal ID

**Response:** `200 OK`
```json
{
  "message": "Protocol document removed successfully"
}
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User is not site admin or group admin
- `404 Not Found`: Animal not found
- `500 Internal Server Error`: Database error

---

### Updates

#### List Updates

Get all updates/posts for a group.

**Endpoint:** `GET /groups/:id/updates`

**Authentication:** Required

**URL Parameters:**
- `id` (integer): Group ID

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "created_at": "2025-10-10T23:40:00.000000Z",
    "updated_at": "2025-10-10T23:40:00.000000Z",
    "group_id": 1,
    "user_id": 1,
    "title": "Great day at the shelter!",
    "content": "We had so many visitors today. Buddy got lots of attention!",
    "image_url": "https://example.com/shelter-day.jpg",
    "user": {
      "id": 1,
      "username": "john_doe",
      "email": "john@example.com",
      "is_admin": false
    }
  }
]
```

**Errors:**
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User not in group
- `500 Internal Server Error`: Database error

#### Create Update

Create a new update/post in a group.

**Endpoint:** `POST /groups/:id/updates`

**Authentication:** Required

**URL Parameters:**
- `id` (integer): Group ID

**Request Body:**
```json
{
  "title": "Great day at the shelter!",
  "content": "We had so many visitors today. Buddy got lots of attention!",
  "image_url": "https://example.com/shelter-day.jpg"
}
```

**Response:** `201 Created`
```json
{
  "id": 1,
  "created_at": "2025-10-10T23:40:00.000000Z",
  "updated_at": "2025-10-10T23:40:00.000000Z",
  "group_id": 1,
  "user_id": 1,
  "title": "Great day at the shelter!",
  "content": "We had so many visitors today. Buddy got lots of attention!",
  "image_url": "https://example.com/shelter-day.jpg",
  "user": {
    "id": 1,
    "username": "john_doe",
    "email": "john@example.com",
    "is_admin": false
  }
}
```

**Errors:**
- `400 Bad Request`: Invalid request body
- `401 Unauthorized`: Invalid or missing token
- `403 Forbidden`: User not in group
- `500 Internal Server Error`: Database error

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

### Common Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request body or parameters
- `401 Unauthorized`: Authentication required or invalid token
- `403 Forbidden`: User doesn't have permission
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists
- `500 Internal Server Error`: Server error

## Rate Limiting

Currently, there is no rate limiting implemented. For production deployments, consider adding rate limiting middleware.

## Pagination

Currently, pagination is not implemented. All list endpoints return all available results. For production use with large datasets, consider implementing pagination.
