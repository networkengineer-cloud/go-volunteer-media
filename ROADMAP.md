# Go Volunteer Media - Product Roadmap

**Last Updated:** November 5, 2025  
**Product Owner:** Product Team  
**Version:** 1.0

---

## 🎯 Product Vision

To create a vibrant, easy-to-use platform that connects shelter volunteers with the animals they care for, enabling real-time collaboration, photo sharing, and meaningful impact tracking. We empower volunteers to stay engaged, informed, and connected to their mission.

---

## 📊 Success Metrics

### Engagement
- **Daily active users:** Track volunteer login frequency
- **Time on platform:** Average session duration
- **Comments per animal:** Measure collaboration depth
- **Photos uploaded:** Visual storytelling engagement

### Adoption
- **Volunteer signup rate:** New user onboarding success
- **Feature adoption:** % of users using key features
- **Mobile usage:** Responsive design effectiveness
- **Return rate:** % of users who return weekly

### Impact
- **Animals with updates:** Coverage of shelter population
- **Volunteer participation:** Active contributors per group
- **Response time:** How quickly volunteers respond to needs
- **Shelter outcomes:** Successful adoptions/placements

---

## 🚀 Now (Q4 2025 - Current)

### 1. Production Deployment Infrastructure (Priority: P0)

**As a** system administrator  
**I want to** deploy the application to Azure with Terraform  
**So that** volunteers can access the platform reliably and securely

**Acceptance Criteria:**
- [x] Terraform infrastructure code complete
- [x] Azure Container Apps configured
- [x] PostgreSQL Flexible Server deployed
- [x] SendGrid SMTP integration active
- [x] GitHub Actions CI/CD pipeline operational
- [ ] Production deployment verified
- [ ] Monitoring and alerting configured
- [ ] Cost optimization implemented (<$20/month)

**Success Metrics:** 99.5% uptime, <2s response time, <$20/month cost  
**Target Release:** v1.1.0  
**Status:** In Progress (Infrastructure ready, deployment pending)

---

### 2. Test Coverage Improvements (Priority: P1)

**As a** development team  
**I want to** increase backend test coverage to 80%+  
**So that** we can confidently deploy changes without regressions

**Acceptance Criteria:**
- [x] Backend test coverage: 58.9% (target: 80%+)
- [ ] All critical handlers tested (auth, animals, comments)
- [ ] Frontend unit tests infrastructure set up
- [ ] E2E tests cover critical user journeys
- [ ] CI/CD fails on coverage drops

**Success Metrics:** 80% backend, 70% frontend coverage  
**Target Release:** v1.1.0  
**Status:** In Progress (Backend 58.9%, Frontend infrastructure ready)

---

## 🔮 Next (Q1 2026)

### 1. Multiple Image Upload & Enhanced Gallery (Priority: P0)

**As a** volunteer  
**I want to** upload multiple images at once and view them in an organized gallery with comments  
**So that** I can efficiently document and share animal activities and progress

**Acceptance Criteria:**
- [ ] Support multiple image uploads per comment/update
- [ ] Drag-and-drop interface for uploading multiple files
- [ ] Progress indicators for each file being uploaded
- [ ] Image gallery with:
  - Grid layout showing all animal images
  - Click to view full size with navigation (left/right arrows)
  - Display associated comment text with each image
  - Show uploader name and timestamp
  - Filter by date range or tags
- [ ] Bulk image management for admins:
  - Select multiple images for deletion
  - Batch caption editing
  - Move images between animals
- [ ] Improved image viewing:
  - Zoom in/out functionality
  - Download original image
  - Image metadata (upload date, size, dimensions)
- [ ] Comment-image association improvements:
  - Display thumbnail grid within comments
  - Expandable image carousel in comment view
  - Support for 1-10 images per comment

**Success Metrics:** 80%+ of volunteers use multiple upload feature, 50% increase in photo sharing  
**Target Release:** v1.2.0  
**Status:** Planned (High Priority - Team Request)

**Technical Requirements:**
- Backend: New `animal_images` table with many-to-one relationship to animals and comments
- Frontend: Multiple file input handling, drag-and-drop library (react-dropzone)
- Storage: Batch upload API endpoint supporting multiple files
- Database: Image metadata storage (filename, size, dimensions, upload_date)
- UI: Enhanced gallery component with carousel/lightbox improvements

**Current State Assessment:**
- ✅ Single image upload working for comments
- ✅ Basic photo gallery page exists (`PhotoGallery.tsx`)
- ✅ Image validation and optimization in place (resize to 1200px, JPEG compression)
- ✅ File upload infrastructure ready (`animal_upload.go`, `UploadAnimalImage` handler)
- ❌ Multiple images per comment not supported
- ❌ No drag-and-drop interface
- ❌ Gallery lacks comment association and filtering
- ❌ No bulk image management tools

---

### 2. Real-Time Notifications (Priority: P1)

**As a** volunteer  
**I want to** receive real-time notifications when someone interacts with my updates  
**So that** I can respond quickly and stay engaged

**Acceptance Criteria:**
- [ ] WebSocket connection for real-time updates
- [ ] In-app notifications for:
  - New comments on animals I follow
  - Replies to my comments
  - Mentions in updates
  - New announcements
- [ ] Notification preferences (enable/disable by type)
- [ ] Unread count badge
- [ ] Mark as read functionality
- [ ] Mobile push notifications (future enhancement)

**Success Metrics:** 70%+ of users enable notifications, <5s notification delivery  
**Target Release:** v1.2.0  
**Status:** Planned

**Technical Requirements:**
- WebSocket server in Go (gorilla/websocket)
- Frontend WebSocket client
- Notification persistence table
- Real-time event broadcasting

---

### 3. Advanced Animal Search & Filtering (Priority: P2)

**As a** volunteer  
**I want to** quickly find specific animals using search and filters  
**So that** I can access information efficiently

**Acceptance Criteria:**
- [ ] Full-text search across animal names, descriptions, breeds
- [ ] Filter by:
  - Status (available, adopted, fostered)
  - Species & breed
  - Age range
  - Date added
  - Last updated
- [ ] Save filter presets
- [ ] Search suggestions as you type
- [ ] Search result highlighting

**Success Metrics:** 80%+ search success rate, <500ms search response time  
**Target Release:** v1.2.0  
**Status:** Planned

**Technical Requirements:**
- PostgreSQL full-text search or Elasticsearch
- Indexed search columns
- Search query optimization

---

## 💭 Later (Q2 2026 and Beyond)

### 1. Volunteer Activity Timeline (Priority: P2)

**As a** volunteer coordinator  
**I want to** see a timeline of volunteer activities  
**So that** I can recognize contributions and identify engagement patterns

**Features:**
- Personal activity timeline for each volunteer
- Shows:
  - Comments posted
  - Photos uploaded
  - Updates created
  - Animals added/updated
- Filter by date range
- Export activity report (CSV)
- Leaderboard for top contributors

**Business Value:** Recognize volunteer contributions, identify engagement patterns  
**Success Metrics:** 50%+ of coordinators use timeline weekly  
**Status:** Backlog (Moved from Q1 to accommodate image gallery priority)

---

### 1. Animal Adoption Workflow (Priority: P2)

**As a** shelter coordinator  
**I want to** track the adoption process from application to placement  
**So that** we can manage adoptions efficiently and improve success rates

**Features:**
- Adoption application form
- Application status tracking (pending, approved, denied)
- Adopter profiles
- Follow-up scheduling
- Adoption success metrics
- Post-adoption check-ins

**Business Value:** Streamline adoption process, increase successful placements  
**Status:** Backlog

---

### 2. Mobile App (Native iOS/Android) (Priority: P3)

**As a** volunteer  
**I want to** use a native mobile app  
**So that** I have a better mobile experience with offline capabilities

**Features:**
- Native iOS and Android apps
- Offline mode for viewing animal profiles
- Push notifications
- Camera integration for easier photo uploads
- Faster performance than web
- App store presence

**Business Value:** Increase mobile engagement, better UX  
**Status:** Backlog (Web-first strategy)

---

### 3. Integration with Shelter Management Systems (Priority: P3)

**As a** shelter administrator  
**I want to** sync data with our existing shelter management software  
**So that** we avoid duplicate data entry and maintain consistency

**Features:**
- API endpoints for external integrations
- Webhook support for real-time sync
- Import/export standards (PetFinder, Adoptapet)
- SSO with existing systems
- Bi-directional sync

**Business Value:** Reduce administrative burden, wider adoption  
**Status:** Backlog (Requires partner discussions)

---

### 4. Performance Optimization (Priority: P2)

**As a** user  
**I want** faster page loads and smoother interactions  
**So that** I have a better experience using the platform

**Features:**
- Redis caching layer for frequently accessed data
- Image CDN integration
- Code splitting and lazy loading (frontend)
- Database query optimization
- Bundle size monitoring
- Lighthouse CI integration
- API response caching

**Business Value:** Better UX, reduced infrastructure costs  
**Status:** Backlog

---

### 5. Advanced Security Features (Priority: P2)

**As a** platform user  
**I want** enhanced security features  
**So that** my account and data are protected

**Features:**
- Two-factor authentication (2FA) with TOTP
- OAuth 2.0 providers (Google, Microsoft)
- API key authentication for integrations
- Session management (view/revoke active sessions)
- Security audit log
- IP whitelisting for admin accounts

**Business Value:** Increased trust, enterprise readiness  
**Status:** Backlog

---

## ✅ Shipped

### Basic Photo Gallery (v1.0.1)
**Shipped:** October 10, 2025  
**Impact:** Volunteers can view all photos shared for each animal in a dedicated gallery page with lightbox  
**Current Capability:** 
- Single image upload per comment
- Gallery view showing all images with comments
- Lightbox with prev/next navigation
- Photo attribution (uploader name + timestamp)

**Known Limitations:**
- Only one image per comment (multiple upload feature planned for v1.2.0)
- No drag-and-drop interface
- Limited filtering options (no date/tag filters)
- No bulk management tools

**Learning:** Photo sharing increased engagement significantly. Users immediately requested ability to upload multiple images at once for documenting activities

---

### Multi-Tag Comment Filtering (v1.0.8)
**Shipped:** November 3, 2025  
**PR:** #40  
**Impact:** 100% of volunteers can now filter animal comments by multiple tags using OR logic  
**Learning:** OR logic was more intuitive than AND logic for tag filtering. Users wanted to see "any of these tags" rather than "all these tags"

---

### Animal Handler Refactoring (v1.0.9)
**Shipped:** November 4, 2025  
**PR:** #78  
**Impact:** Reduced animal.go from 962 lines to 6 focused modules (<200 lines each)  
**Learning:** Breaking handlers into smaller modules significantly improved maintainability and testability. Test coverage for animal handlers increased from 8.7% to 17.2%

---

### Security Hardening (v1.0.10)
**Shipped:** October 31, 2025  
**PR:** #41  
**Impact:** Fixed critical rate limiter bug, added JWT entropy validation, enhanced DB connection pooling  
**Learning:** Security scanning tools (govulncheck, npm audit) caught issues before production. Automated security scanning now required in CI/CD

---

### CI/CD Pipeline & E2E Testing (v1.0.6)
**Shipped:** November 1, 2025  
**PR:** #67  
**Impact:** Automated testing catches regressions before merge. Backend coverage increased from 0% to 58.9%  
**Learning:** E2E tests with Playwright caught UI regressions that unit tests missed. PostgreSQL database required in CI for realistic E2E testing

---

### Dark Mode Support (v1.0.5)
**Shipped:** October 30, 2025  
**PR:** #73  
**Impact:** 40%+ of users prefer dark mode. Improved accessibility and reduced eye strain  
**Learning:** CSS custom properties made theming straightforward. Contrast ratios required careful testing to maintain WCAG compliance

---

### Bulk Animal Management (v1.0.4)
**Shipped:** October 25, 2025  
**Impact:** Admins can now bulk edit, import CSV, and export CSV for efficient animal management  
**Learning:** CSV import/export was most-requested admin feature. Reduced data entry time by 80%

---

### Announcement System with Email (v1.0.3)
**Shipped:** October 20, 2025  
**Impact:** Coordinators can broadcast announcements with optional email notifications  
**Learning:** Email notifications had 85% open rate. Email preferences were critical to avoid overwhelming users

---

### Password Reset Functionality (v1.0.2)
**Shipped:** October 15, 2025  
**Impact:** Users can reset forgotten passwords via email  
**Learning:** Password resets reduced support tickets by 60%. Token expiration (1 hour) balanced security and usability

---

## ⛔ Won't Do (Out of Scope)

### Social Media Cross-Posting
**Why:** Too complex for v1. Users can manually share if needed. Future consideration if demand is high.

### Built-in Chat/Messaging
**Why:** Comments serve as lightweight communication. Full chat requires significant infrastructure (WebSocket, message persistence, notifications). May revisit in v2.0.

### Public-Facing Adoption Portal
**Why:** Shelters typically have existing adoption websites. Our focus is internal volunteer coordination, not public adoption listings. Integration (later) is better than building from scratch.

### Gamification/Points System
**Why:** Risk of perverse incentives (gaming the system vs genuine engagement). Focus on intrinsic motivation first. May revisit with careful design.

### Video Upload Support
**Why:** Storage and bandwidth costs too high for current scale. Photos are sufficient. CDN integration required before considering video.

### Multi-Language Support (i18n)
**Why:** Current user base is English-speaking. Will add when we have non-English users. i18n architecture can be added later without major refactor.

---

## 📝 Roadmap Notes

### Release Cadence
- **Minor releases (v1.x):** Every 2-3 weeks
- **Major releases (v2.0):** Annually
- **Hotfixes:** As needed for critical bugs

### Priority Definitions
- **P0 (Critical):** Blocks launch or core functionality
- **P1 (High):** Important for user satisfaction
- **P2 (Medium):** Nice to have, improves experience
- **P3 (Low):** Future enhancement, not urgent

### Feature Evaluation Criteria
1. **User Impact:** How many users benefit?
2. **Business Value:** Does it support our mission?
3. **Technical Complexity:** Engineering effort required?
4. **Strategic Fit:** Aligns with product vision?
5. **Data-Driven:** Based on user feedback/metrics?

### How to Suggest Features
1. Create a GitHub issue with the `enhancement` label
2. Describe the problem you're trying to solve (not the solution)
3. Explain who would benefit and how
4. Product Owner will evaluate and update roadmap

---

**Questions about the roadmap?** Open a GitHub issue or contact the Product Team.

**Want to contribute?** Check `CONTRIBUTING.md` for guidelines.
