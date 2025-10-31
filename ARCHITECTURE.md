# Go Volunteer Media - Architecture Documentation

This document provides comprehensive architecture diagrams showing how the Go Volunteer Media application works.

## Table of Contents
1. [High-Level System Architecture](#high-level-system-architecture)
2. [Application Stack](#application-stack)
3. [Request Flow Architecture](#request-flow-architecture)
4. [Database Schema](#database-schema)
5. [Authentication Flow](#authentication-flow)
6. [API Route Structure](#api-route-structure)
7. [Frontend Component Architecture](#frontend-component-architecture)
8. [Middleware Pipeline](#middleware-pipeline)

---

## High-Level System Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        Browser[Web Browser]
        Mobile[Mobile Browser]
    end
    
    subgraph "Frontend - React SPA"
        Router[React Router]
        Auth[Auth Context]
        Components[React Components]
        API_Client[Axios API Client]
    end
    
    subgraph "Backend - Go API Server"
        GinRouter[Gin HTTP Router]
        Middleware[Middleware Stack]
        Handlers[HTTP Handlers]
        Models[GORM Models]
    end
    
    subgraph "Data Layer"
        PostgreSQL[(PostgreSQL Database)]
        FileSystem[File System<br/>Image Uploads]
    end
    
    subgraph "External Services"
        SMTP[SMTP Email Service]
    end
    
    Browser --> Router
    Mobile --> Router
    Router --> Auth
    Auth --> Components
    Components --> API_Client
    API_Client -->|HTTP/JSON| GinRouter
    GinRouter --> Middleware
    Middleware --> Handlers
    Handlers --> Models
    Models --> PostgreSQL
    Handlers --> FileSystem
    Handlers --> SMTP
```

---

## Application Stack

```mermaid
graph LR
    subgraph "Frontend Stack"
        React[React 18]
        TS[TypeScript]
        Vite[Vite Build Tool]
        Router[React Router v6]
        Axios[Axios HTTP Client]
        CSS[CSS Custom Properties]
    end
    
    subgraph "Backend Stack"
        Go[Go 1.21+]
        Gin[Gin Web Framework]
        GORM[GORM ORM]
        JWT[JWT Auth]
        Bcrypt[Bcrypt Password Hash]
        Logrus[Logrus Logging]
    end
    
    subgraph "Infrastructure"
        Postgres[PostgreSQL]
        Docker[Docker]
        Nginx[Nginx Optional]
    end
    
    React --> TS
    TS --> Vite
    Vite --> Router
    Router --> Axios
    Axios --> CSS
    
    Go --> Gin
    Gin --> GORM
    GORM --> JWT
    JWT --> Bcrypt
    Bcrypt --> Logrus
    
    GORM --> Postgres
    Go --> Docker
    Docker --> Nginx
```

---

## Request Flow Architecture

```mermaid
sequenceDiagram
    participant Client as Browser/Client
    participant Router as Gin Router
    participant MW as Middleware Pipeline
    participant Handler as HTTP Handler
    participant Model as GORM Model
    participant DB as PostgreSQL
    
    Client->>Router: HTTP Request (GET /api/groups/1/animals)
    Router->>MW: Route Match
    
    Note over MW: Security Headers
    MW->>MW: Add Security Headers
    
    Note over MW: Request ID
    MW->>MW: Generate Request ID
    
    Note over MW: Structured Logging
    MW->>MW: Log Request Start
    
    Note over MW: CORS
    MW->>MW: Check CORS Headers
    
    Note over MW: Authentication
    MW->>MW: Validate JWT Token
    MW->>MW: Extract User ID & Admin Status
    
    Note over MW: Authorization
    MW->>Handler: Check Group Access
    
    Handler->>Handler: Validate Input
    Handler->>Model: Query Animals
    Model->>DB: SELECT * FROM animals WHERE group_id = $1
    DB-->>Model: Result Set
    Model-->>Handler: []Animal
    Handler->>Handler: Format Response
    Handler-->>MW: JSON Response
    
    Note over MW: Logging
    MW->>MW: Log Request Complete
    
    MW-->>Router: Response
    Router-->>Client: HTTP 200 + JSON Body
```

---

## Database Schema

```mermaid
erDiagram
    User ||--o{ AnimalComment : creates
    User ||--o{ Update : creates
    User ||--o{ Announcement : creates
    User }o--o{ Group : "member of"
    
    Group ||--o{ Animal : contains
    Group ||--o{ Update : has
    
    Animal ||--o{ AnimalComment : has
    
    AnimalComment }o--o{ CommentTag : tagged_with
    
    User {
        uint id PK
        string username UK
        string email UK
        string password
        bool is_admin
        int failed_login_attempts
        timestamp locked_until
        string reset_token
        timestamp reset_token_expiry
        bool email_notifications_enabled
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    Group {
        uint id PK
        string name UK
        string description
        string image_url
        string hero_image_url
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    Animal {
        uint id PK
        uint group_id FK
        string name
        string species
        string breed
        int age
        string description
        string image_url
        string status
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    Update {
        uint id PK
        uint group_id FK
        uint user_id FK
        string title
        string content
        string image_url
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    Announcement {
        uint id PK
        uint user_id FK
        string title
        string content
        bool send_email
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    AnimalComment {
        uint id PK
        uint animal_id FK
        uint user_id FK
        string content
        string image_url
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    CommentTag {
        uint id PK
        string name UK
        string color
        bool is_system
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    SiteSetting {
        uint id PK
        string key UK
        string value
        timestamp created_at
        timestamp updated_at
    }
```

---

## Authentication Flow

```mermaid
sequenceDiagram
    participant Client as React App
    participant Login as Login Component
    participant API as Auth API
    participant Handler as Auth Handler
    participant DB as PostgreSQL
    participant JWT as JWT Service
    participant Context as Auth Context
    
    Client->>Login: User enters credentials
    Login->>API: POST /api/login {username, password}
    API->>Handler: Process Login
    
    Handler->>DB: Find user by username
    DB-->>Handler: User record
    
    alt User not found
        Handler-->>API: 401 Unauthorized
        API-->>Login: Error: Invalid credentials
    else User found
        Handler->>Handler: Check account locked
        alt Account locked
            Handler-->>API: 403 Forbidden
            API-->>Login: Error: Account locked
        else Account active
            Handler->>Handler: bcrypt.CompareHashAndPassword
            alt Password invalid
                Handler->>DB: Increment failed_login_attempts
                Handler-->>API: 401 Unauthorized
                API-->>Login: Error: Invalid credentials
            else Password valid
                Handler->>DB: Reset failed_login_attempts
                Handler->>JWT: Generate JWT Token
                JWT-->>Handler: token_string
                Handler-->>API: 200 OK {token, user}
                API->>Context: Store token in localStorage
                API->>Context: Update auth state
                Context-->>Client: Authenticated
                Client->>Client: Navigate to /dashboard
            end
        end
    end
```

---

## API Route Structure

```mermaid
graph TB
    Root["/api"]
    
    Root --> Public[Public Routes<br/>No Auth Required]
    Root --> Protected[Protected Routes<br/>Auth Required]
    Root --> Admin[Admin Routes<br/>Admin Required]
    
    Public --> Login["/login<br/>POST"]
    Public --> Register["/register<br/>POST"]
    Public --> ResetReq["/request-password-reset<br/>POST"]
    Public --> ResetPass["/reset-password<br/>POST"]
    Public --> Settings["/settings<br/>GET"]
    
    Protected --> Me["/me<br/>GET"]
    Protected --> Groups["/groups<br/>GET"]
    Protected --> Announcements["/announcements<br/>GET"]
    Protected --> CommentTags["/comment-tags<br/>GET"]
    Protected --> ImageUpload["/animals/upload-image<br/>POST"]
    Protected --> GroupRoutes["Group Routes<br/>/groups/:id/*"]
    
    GroupRoutes --> GetGroup["/groups/:id<br/>GET"]
    GroupRoutes --> Animals["/groups/:id/animals<br/>GET/POST/PUT/DELETE"]
    GroupRoutes --> Comments["/groups/:id/animals/:animalId/comments<br/>GET/POST"]
    GroupRoutes --> Updates["/groups/:id/updates<br/>GET/POST"]
    
    Admin --> Users["/admin/users<br/>CRUD Operations"]
    Admin --> GroupsAdmin["/admin/groups<br/>CRUD Operations"]
    Admin --> AnnouncementsAdmin["/admin/announcements<br/>CRUD Operations"]
    Admin --> TagsAdmin["/admin/comment-tags<br/>CRUD Operations"]
    Admin --> SettingsAdmin["/admin/settings<br/>Update Operations"]
    Admin --> BulkAnimals["/admin/animals<br/>Bulk Operations"]
    
    BulkAnimals --> GetAll["/admin/animals<br/>GET - All animals"]
    BulkAnimals --> BulkUpdate["/admin/animals/bulk-update<br/>POST"]
    BulkAnimals --> Import["/admin/animals/import-csv<br/>POST"]
    BulkAnimals --> Export["/admin/animals/export-csv<br/>GET"]
    BulkAnimals --> ExportComments["/admin/animals/export-comments-csv<br/>GET"]
    
    style Public fill:#e1f5ff
    style Protected fill:#fff4e6
    style Admin fill:#ffe6e6
```

---

## Frontend Component Architecture

```mermaid
graph TB
    App[App.tsx<br/>Router Setup]
    
    App --> AuthProvider[AuthProvider<br/>Global Auth State]
    App --> Navigation[Navigation Component<br/>Header & Menu]
    App --> Routes[React Router Routes]
    
    Routes --> PublicRoutes[Public Routes]
    Routes --> PrivateRoutes[Private Routes]
    Routes --> AdminRoutes[Admin Routes]
    
    PublicRoutes --> Home[Home Page]
    PublicRoutes --> Login[Login Page]
    PublicRoutes --> ResetPassword[Reset Password Page]
    
    PrivateRoutes --> Dashboard[Dashboard<br/>Group Cards]
    PrivateRoutes --> GroupPage[Group Page<br/>Animals & Updates]
    PrivateRoutes --> AnimalDetail[Animal Detail Page<br/>Comments & Photos]
    PrivateRoutes --> PhotoGallery[Photo Gallery]
    PrivateRoutes --> SettingsPage[User Settings]
    
    AdminRoutes --> UsersPage[Users Management]
    AdminRoutes --> GroupsPage[Groups Management]
    AdminRoutes --> AdminSettings[Site Settings]
    AdminRoutes --> BulkEdit[Bulk Edit Animals]
    AdminRoutes --> AnimalForm[Animal Form<br/>Create/Edit]
    
    AuthProvider --> APIClient[Axios API Client<br/>HTTP Interceptors]
    
    APIClient --> AuthAPI[Auth API]
    APIClient --> GroupsAPI[Groups API]
    APIClient --> AnimalsAPI[Animals API]
    APIClient --> UpdatesAPI[Updates API]
    APIClient --> CommentsAPI[Comments API]
    APIClient --> UsersAPI[Users API - Admin]
    APIClient --> AnnouncementsAPI[Announcements API]
    
    style PublicRoutes fill:#e1f5ff
    style PrivateRoutes fill:#fff4e6
    style AdminRoutes fill:#ffe6e6
```

---

## Middleware Pipeline

```mermaid
graph TB
    Request[Incoming HTTP Request]
    
    Request --> Recovery[Recovery Middleware<br/>Panic Handler]
    Recovery --> Security[Security Headers<br/>X-Frame-Options, CSP, etc.]
    Security --> RequestID[Request ID<br/>Generate Unique ID]
    RequestID --> Logging[Structured Logging<br/>Log Request Start]
    Logging --> CORS[CORS Middleware<br/>Handle Cross-Origin]
    
    CORS --> RouteCheck{Route Type?}
    
    RouteCheck -->|Public| PublicHandler[Public Handler<br/>No Auth Required]
    RouteCheck -->|Protected| AuthRequired[Auth Required<br/>Validate JWT Token]
    RouteCheck -->|Admin| AdminRequired[Admin Required<br/>Check Admin Flag]
    
    AuthRequired --> GroupAccess{Group-Specific<br/>Route?}
    GroupAccess -->|Yes| CheckGroupAccess[Check Group Access<br/>User in Group?]
    GroupAccess -->|No| Handler[Execute Handler]
    
    CheckGroupAccess -->|Access Granted| Handler
    CheckGroupAccess -->|Access Denied| Forbidden403[403 Forbidden]
    
    AdminRequired --> CheckAdmin{Is Admin?}
    CheckAdmin -->|Yes| Handler
    CheckAdmin -->|No| Forbidden403
    
    AuthRequired --> CheckAuth{Token Valid?}
    CheckAuth -->|Yes| ExtractUser[Extract User Info<br/>user_id, is_admin]
    CheckAuth -->|No| Unauthorized401[401 Unauthorized]
    
    ExtractUser --> Handler
    PublicHandler --> Handler
    
    Handler --> RateLimiter{Rate Limited<br/>Route?}
    RateLimiter -->|Yes| CheckRate[Check Rate Limit]
    RateLimiter -->|No| Execute[Execute Handler Logic]
    
    CheckRate -->|Within Limit| Execute
    CheckRate -->|Exceeded| TooManyRequests429[429 Too Many Requests]
    
    Execute --> LogComplete[Log Request Complete]
    LogComplete --> Response[Send Response]
    
    Unauthorized401 --> LogComplete
    Forbidden403 --> LogComplete
    TooManyRequests429 --> LogComplete
    
    style Recovery fill:#ff9999
    style Security fill:#99ccff
    style AuthRequired fill:#ffcc99
    style AdminRequired fill:#ff9999
    style Handler fill:#99ff99
```

---

## Data Flow: Creating an Animal Comment

```mermaid
sequenceDiagram
    participant User as User Browser
    participant React as React Component
    participant Axios as Axios Client
    participant Router as Gin Router
    participant Auth as Auth Middleware
    participant Handler as Comment Handler
    participant GORM as GORM Model
    participant DB as PostgreSQL
    participant Logger as Structured Logger
    
    User->>React: Click "Add Comment"
    React->>React: Show comment form
    User->>React: Enter comment text + optional image
    React->>React: Validate input
    React->>Axios: POST /api/groups/:id/animals/:animalId/comments
    
    Note over Axios: Add JWT token to headers
    
    Axios->>Router: HTTP POST Request
    Router->>Auth: Check Authentication
    Auth->>Auth: Validate JWT Token
    Auth->>Auth: Extract user_id from token
    Auth->>Handler: User authenticated (user_id: 123)
    
    Handler->>Handler: Validate group access
    Handler->>Logger: Log comment creation attempt
    
    Handler->>Handler: Parse request body
    Handler->>Handler: Validate comment content
    Handler->>Handler: Validate optional image URL
    
    Handler->>GORM: Create AnimalComment
    GORM->>DB: INSERT INTO animal_comments
    DB-->>GORM: Comment ID returned
    GORM-->>Handler: Comment object with ID
    
    Handler->>Logger: Log comment created successfully
    Handler-->>Axios: 201 Created {comment}
    Axios-->>React: Response data
    React->>React: Update UI with new comment
    React-->>User: Display new comment
```

---

## File Upload Flow

```mermaid
graph TB
    User[User Selects Image]
    
    User --> FormData[Create FormData Object]
    FormData --> Upload[POST /api/animals/upload-image]
    
    Upload --> Handler[Upload Handler]
    
    Handler --> Validate[Validate Image Upload]
    Validate --> CheckSize{Size < 10MB?}
    CheckSize -->|No| Error400[400 Bad Request]
    CheckSize -->|Yes| CheckType{Valid Type?<br/>jpg/png/gif}
    
    CheckType -->|No| Error400
    CheckType -->|Yes| CheckContent[Validate Image Content]
    
    CheckContent --> Decode[Decode Image]
    Decode --> Resize{Needs Resize?<br/>> 1200px}
    
    Resize -->|Yes| ResizeImg[Resize Image<br/>Lanczos3 Algorithm]
    Resize -->|No| UseOriginal[Use Original]
    
    ResizeImg --> Optimize[Encode as JPEG<br/>Quality: 85]
    UseOriginal --> Optimize
    
    Optimize --> GenerateFilename[Generate Unique Filename<br/>timestamp_uuid.jpg]
    GenerateFilename --> SaveFile[Save to public/uploads/]
    SaveFile --> ReturnURL[Return URL: /uploads/filename.jpg]
    
    ReturnURL --> Client[Client Receives URL]
    Client --> UseInForm[Use URL in Animal/Comment Form]
```

---

## Security Architecture

```mermaid
graph TB
    subgraph "Authentication Layer"
        JWT[JWT Token Authentication]
        Bcrypt[Bcrypt Password Hashing]
        RateLimiting[Rate Limiting<br/>5 req/min for auth]
        AccountLocking[Account Locking<br/>After failed attempts]
    end
    
    subgraph "Authorization Layer"
        RoleCheck[Role-Based Access<br/>Admin vs User]
        GroupAccess[Group Membership Check]
        ResourceOwnership[Resource Ownership Check]
    end
    
    subgraph "Input Validation"
        ParamValidation[URL Parameter Validation]
        BodyValidation[JSON Body Validation]
        FileValidation[File Upload Validation<br/>Type, Size, Content]
        UsernameValidation[Username Character Validation]
    end
    
    subgraph "Security Headers"
        XFrameOptions[X-Frame-Options: DENY]
        ContentType[X-Content-Type-Options: nosniff]
        XSSProtection[X-XSS-Protection: 1]
        CSP[Content-Security-Policy]
    end
    
    subgraph "Data Protection"
        PasswordHiding[Password Field<br/>json:"-"]
        SoftDeletes[Soft Deletes<br/>DeletedAt field]
        ParameterizedQueries[GORM Parameterized Queries<br/>SQL Injection Prevention]
    end
    
    Request[Incoming Request] --> JWT
    JWT --> RateLimiting
    RateLimiting --> AccountLocking
    AccountLocking --> RoleCheck
    RoleCheck --> GroupAccess
    GroupAccess --> ResourceOwnership
    ResourceOwnership --> ParamValidation
    ParamValidation --> BodyValidation
    BodyValidation --> FileValidation
    FileValidation --> UsernameValidation
    UsernameValidation --> XFrameOptions
    XFrameOptions --> ContentType
    ContentType --> XSSProtection
    XSSProtection --> CSP
    CSP --> PasswordHiding
    PasswordHiding --> SoftDeletes
    SoftDeletes --> ParameterizedQueries
    ParameterizedQueries --> SecureHandler[Secure Handler Execution]
```

---

## Deployment Architecture

```mermaid
graph TB
    subgraph "Production Environment"
        LB[Load Balancer<br/>Optional]
        
        subgraph "Application Container"
            App[Go API Server<br/>Port 8080]
            Static[Static Files<br/>Frontend Dist]
            Uploads[Uploads Directory<br/>public/uploads]
        end
        
        subgraph "Database"
            PG[PostgreSQL<br/>Port 5432]
            Backup[Automated Backups]
        end
        
        subgraph "External Services"
            SMTP[SMTP Server<br/>Email Notifications]
        end
    end
    
    Internet[Internet] --> LB
    LB --> App
    App --> Static
    App --> Uploads
    App --> PG
    App --> SMTP
    PG --> Backup
    
    subgraph "Development Environment"
        DevFE[Vite Dev Server<br/>Port 5173]
        DevBE[Go API Server<br/>Port 8080]
        DevDB[PostgreSQL<br/>Docker Compose]
    end
    
    Developer[Developer] --> DevFE
    DevFE -->|Proxy /api| DevBE
    DevBE --> DevDB
```

---

## Logging and Monitoring Flow

```mermaid
graph LR
    subgraph "Request Lifecycle"
        Start[Request Start]
        Process[Request Processing]
        End[Request Complete]
    end
    
    subgraph "Logging Points"
        Start --> StartLog[Log Request Start<br/>method, path, request_id]
        Process --> ActionLog[Log Handler Actions<br/>database queries, errors]
        End --> CompleteLog[Log Request Complete<br/>status, duration, user_id]
    end
    
    subgraph "Log Structure"
        StartLog --> Fields1[Structured Fields:<br/>- request_id<br/>- method<br/>- path<br/>- ip_address]
        ActionLog --> Fields2[Structured Fields:<br/>- action<br/>- user_id<br/>- resource_id<br/>- error details]
        CompleteLog --> Fields3[Structured Fields:<br/>- status_code<br/>- duration_ms<br/>- response_size]
    end
    
    subgraph "Output"
        Fields1 --> JSON[JSON Log Format]
        Fields2 --> JSON
        Fields3 --> JSON
        JSON --> Stdout[Standard Output]
        JSON --> LogFile[Log File Optional]
        LogFile --> Aggregator[Log Aggregation<br/>ELK, Splunk, etc.]
    end
```

---

## Key Architectural Patterns

### 1. **Separation of Concerns**
- **Frontend**: React SPA handles all UI/UX
- **Backend**: Go API handles business logic and data
- **Database**: PostgreSQL stores persistent data

### 2. **RESTful API Design**
- Resource-based URLs: `/api/groups/:id/animals`
- Standard HTTP methods: GET, POST, PUT, DELETE
- Consistent JSON responses

### 3. **Middleware Pattern**
- Composable middleware pipeline
- Request/response transformation
- Cross-cutting concerns (logging, auth, CORS)

### 4. **Context-Based Authentication**
- JWT tokens for stateless auth
- User context propagated through middleware
- Role-based access control (RBAC)

### 5. **Repository Pattern (via GORM)**
- Models define data structure and relationships
- GORM handles database abstraction
- Handlers contain business logic

### 6. **Structured Logging**
- Request ID for request tracing
- Contextual fields for debugging
- JSON output for log aggregation

### 7. **Graceful Degradation**
- Email service optional (checks configuration)
- Soft deletes preserve data
- Account locking for security

### 8. **Image Optimization**
- Client-side image preview
- Server-side validation and optimization
- Resize large images automatically
- Convert to JPEG for consistency

---

## Technology Decisions

| Component | Technology | Reasoning |
|-----------|------------|-----------|
| Backend Language | Go | Performance, simplicity, strong typing, excellent concurrency |
| Web Framework | Gin | Fast HTTP router, middleware support, good documentation |
| ORM | GORM | Feature-rich, supports associations, migrations, soft deletes |
| Database | PostgreSQL | Robust, ACID compliant, excellent for relational data |
| Authentication | JWT + Bcrypt | Stateless, secure, industry standard |
| Frontend Framework | React 18 | Component-based, large ecosystem, hooks API |
| Language | TypeScript | Type safety, better IDE support, catches errors early |
| Build Tool | Vite | Fast HMR, modern tooling, optimized builds |
| HTTP Client | Axios | Interceptors, request/response transformation |
| Styling | CSS Custom Properties | Native browser support, theming, no build step |
| Logging | Logrus | Structured logging, contextual fields, JSON output |

---

## Performance Considerations

1. **Database Indexes**
   - Composite indexes on frequently queried fields
   - Index on `deleted_at` for soft delete queries
   - Foreign key indexes for relationships

2. **Connection Pooling**
   - SQL database connection pool configured
   - Reuse connections for better performance

3. **Image Optimization**
   - Automatic resizing to max 1200px
   - JPEG encoding at quality 85
   - Reduced storage and bandwidth

4. **Lazy Loading**
   - GORM relationships loaded on-demand
   - Frontend components code-split
   - Images loaded as needed

5. **Caching Opportunities**
   - Static files served efficiently
   - JWT tokens cached in localStorage
   - API responses can be cached client-side

---

## Security Checklist

- ✅ Password hashing with bcrypt
- ✅ JWT token authentication
- ✅ Rate limiting on auth endpoints
- ✅ Account locking after failed attempts
- ✅ CORS configuration
- ✅ Security headers
- ✅ Input validation
- ✅ File upload validation
- ✅ SQL injection prevention (parameterized queries)
- ✅ XSS prevention (JSON encoding)
- ✅ Soft deletes (data preservation)
- ✅ Role-based access control
- ✅ Group membership validation
- ✅ Password reset tokens with expiration
- ✅ Structured logging for audit trails

---

## Future Enhancements

1. **WebSocket Support** - Real-time notifications and updates
2. **Redis Caching** - Cache frequently accessed data
3. **Full-Text Search** - Better animal/comment search
4. **Background Jobs** - Async email sending, image processing
5. **File Storage** - S3/CloudFlare R2 for scalable image storage
6. **API Versioning** - Support multiple API versions
7. **GraphQL Option** - Alternative to REST for complex queries
8. **Metrics/Monitoring** - Prometheus metrics, health checks
9. **Rate Limiting per User** - More granular rate limits
10. **Audit Logging** - Dedicated audit trail table

---

*Generated on: October 31, 2025*
