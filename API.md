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

**Endpoint:** `GET /admin/groups/:id/members`

**Authentication:** Required (Site Admin or Group Member)

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

#### Promote User to Group Admin

Promote a user to group admin for a specific group.

**Endpoint:** `POST /admin/groups/:groupId/admins/:userId`

**Authentication:** Required (Site Admin)

**URL Parameters:**
- `groupId` (integer): Group ID
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
- `403 Forbidden`: User is not site admin
- `404 Not Found`: User or group not found

#### Demote User from Group Admin

Remove group admin privileges from a user for a specific group.

**Endpoint:** `DELETE /admin/groups/:groupId/admins/:userId`

**Authentication:** Required (Site Admin)

**URL Parameters:**
- `groupId` (integer): Group ID
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
- `403 Forbidden`: User is not site admin
- `404 Not Found`: User or group not found

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
