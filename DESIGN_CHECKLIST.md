# Design Checklist - Volunteer Portal

## Overview
This checklist outlines design updates to align the volunteer portal with the HAWS brand identity, inspired by hawspets.org's professional and warm aesthetic while maintaining focus on internal volunteer workflows.

**Scope**: Internal volunteer management system (excludes public-facing marketing elements like donation buttons, social media links, mission statements)

---

## Phase 1: Critical Visual Updates (High Priority)

### 1.1 Color Palette
- [ ] **Primary Brand Color**: Update to HAWS teal/turquoise `#00A3AD` (from hawspets.org)
- [ ] **Secondary/Accent**: Implement warm yellow/gold `#F2B705` for CTAs and highlights
- [ ] **Tertiary**: Add purple accent `#7B68A6` for badges, status indicators
- [ ] **Navy/Dark Blue**: Use `#1E3A5F` for headers, navigation, professional text
- [ ] **Neutral Grays**: Establish consistent gray scale for backgrounds, borders, disabled states
- [ ] **Semantic Colors**: Define success (green), warning (amber), error (red), info (blue)

### 1.2 Logo & Branding
- [ ] Add HAWS logo to navigation header (top-left)
- [ ] Implement consistent "HAWS Volunteer Portal" branding text
- [ ] Ensure logo scales properly on mobile devices
- [ ] Add favicon with HAWS branding

### 1.3 Typography
- [ ] **Headings**: Use clean, professional sans-serif (e.g., Montserrat, Open Sans, or similar to hawspets.org)
- [ ] **Body Text**: Implement readable sans-serif for content (16px base, 1.5 line-height)
- [ ] **Font Weights**: Establish hierarchy (Bold for headings, Regular for body, Medium for emphasis)
- [ ] **Letter Spacing**: Slight increase for headers for modern feel

---

## Phase 2: Layout & Structure Improvements

### 2.1 Navigation
- [ ] Implement fixed/sticky top navigation bar with HAWS navy background
- [ ] Add hover states with teal underline or background
- [ ] Mobile: Implement hamburger menu with smooth slide-in animation
- [ ] Active page indicator with teal accent
- [ ] User menu dropdown (top-right) with profile, settings, logout

### 2.2 Page Headers
- [ ] Consistent page header design with:
  - Large, bold page title (navy color)
  - Optional subtitle or breadcrumb navigation
  - Action buttons aligned to the right (teal primary, white secondary)
  - Generous padding/whitespace (40-60px vertical)

### 2.3 Card-Based Layout
- [ ] Implement card containers for content sections
- [ ] Subtle shadows for depth (e.g., `box-shadow: 0 2px 8px rgba(0,0,0,0.1)`)
- [ ] Rounded corners (8-12px border-radius)
- [ ] Consistent padding (24-32px)
- [ ] White/light gray backgrounds for cards

### 2.4 Grid Systems
- [ ] Dashboard: 2-3 column grid for stats/widgets (responsive: 1 col mobile, 2 col tablet, 3 col desktop)
- [ ] Animal listings: Grid or card layout with filters sidebar
- [ ] Responsive breakpoints: 320px, 768px, 1024px, 1440px

---

## Phase 3: Component Polish

### 3.1 Buttons
- [ ] **Primary**: Teal background `#00A3AD`, white text, rounded corners
- [ ] **Secondary**: White background, teal border and text
- [ ] **Hover States**: Darken primary by 10%, add subtle lift/shadow
- [ ] **Sizes**: Small (32px height), Medium (40px), Large (48px)
- [ ] **Icons**: Add icons to buttons where appropriate (save, delete, add, etc.)

### 3.2 Forms
- [ ] Input fields: Consistent border (1px solid gray), focus state (teal border, subtle shadow)
- [ ] Labels: Above inputs, medium weight, slightly darker gray
- [ ] Placeholder text: Light gray, helpful examples
- [ ] Error states: Red border, error message below in red
- [ ] Success states: Green border/checkmark for completed fields
- [ ] Required field indicators: Red asterisk or "(required)" text

### 3.3 Tables
- [ ] Header row: Navy or dark background with white text
- [ ] Alternating row colors (zebra striping) for readability
- [ ] Hover state: Light teal background on row hover
- [ ] Action buttons: Icon buttons (edit, delete) aligned right
- [ ] Responsive: Scroll horizontally on mobile or stack columns
- [ ] Pagination: Teal active page, gray inactive

### 3.4 Modals/Dialogs
- [ ] Overlay: Semi-transparent dark background (rgba(0,0,0,0.5))
- [ ] Modal: White card, centered, max-width 600px, rounded corners
- [ ] Header: Navy background with white text, close button (X)
- [ ] Footer: Action buttons aligned right (cancel, confirm)
- [ ] Animation: Fade in with slight scale effect

---

## Phase 4: Imagery & Visual Elements

### 4.1 Animal Photos
- [ ] Consistent aspect ratio (e.g., 4:3 or 1:1 for thumbnails)
- [ ] Rounded corners to match card style
- [ ] Hover effect: Slight zoom or overlay with animal name
- [ ] Placeholder image for animals without photos (branded HAWS placeholder)
- [ ] Image optimization: Compressed, responsive srcset

### 4.2 Icons
- [ ] Consistent icon library (e.g., Feather Icons, Heroicons, Font Awesome)
- [ ] Icons for: add, edit, delete, save, cancel, user, group, settings, logout, calendar, photo
- [ ] Color: Match text color or use teal for interactive elements
- [ ] Size: 16px, 20px, 24px variants

### 4.3 Empty States
- [ ] Friendly illustration or icon for empty lists/tables
- [ ] Helpful message: "No animals added yet" + CTA button
- [ ] Lighter gray background to differentiate from content areas

---

## Phase 5: Accessibility & Usability

### 5.1 Accessibility (WCAG AA)
- [ ] Color contrast: Ensure 4.5:1 for normal text, 3:1 for large text
- [ ] Keyboard navigation: All interactive elements accessible via Tab
- [ ] Focus indicators: Visible outline/ring on focused elements
- [ ] Alt text: Descriptive alt text for all images
- [ ] ARIA labels: For icon buttons, form inputs, navigation
- [ ] Screen reader testing: Test with VoiceOver (macOS) or NVDA (Windows)

### 5.2 Loading States
- [ ] Skeleton screens for page loads (gray pulsing placeholders)
- [ ] Spinner/loader for button actions (save, submit)
- [ ] Progress indicators for multi-step forms or uploads
- [ ] Disable buttons during async operations

### 5.3 Feedback & Notifications
- [ ] Toast notifications: Top-right corner, auto-dismiss after 5s
- [ ] Success: Green background, checkmark icon
- [ ] Error: Red background, X icon
- [ ] Info: Blue background, info icon
- [ ] Warning: Yellow background, alert icon

---

## Phase 6: Page-Specific Enhancements

### 6.1 Dashboard
- [ ] Welcome banner with user's name and HAWS branding
- [ ] Stats cards: Total animals, updates this week, active groups
- [ ] Recent activity feed: List of recent updates/announcements
- [ ] Quick action buttons: "Add Animal", "Post Update", "View Schedule"

### 6.2 Animal Management
- [ ] Filter/search bar: Prominent at top with species, status, group filters
- [ ] Grid view toggle: Grid vs. list view option
- [ ] Animal cards: Photo, name, species, status badge, edit/delete buttons
- [ ] Detail view: Full animal profile with photo gallery, history, notes

### 6.3 Updates/Announcements
- [ ] Timeline/feed layout: Most recent at top
- [ ] Update cards: Author, timestamp, content, attached images
- [ ] Compose update: Rich text editor with image upload
- [ ] Filter by group or date range

### 6.4 Group Management
- [ ] Group list: Card layout with group name, member count, description
- [ ] Group detail: Member list, activity feed, settings
- [ ] Add members: Search/select from user list

### 6.5 User Management (Admin)
- [ ] User table: Name, email, role, groups, actions
- [ ] Role badges: Color-coded (admin = red, volunteer = blue)
- [ ] Invite user: Modal with email input and role selection
- [ ] User detail: Edit profile, change role, view activity

### 6.6 Settings
- [ ] Tabbed interface: Profile, Security, Notifications, Preferences
- [ ] Profile: Avatar upload, name, email, bio
- [ ] Security: Change password, 2FA toggle, session management
- [ ] Notifications: Email preferences, in-app notification toggles

---

## Phase 7: Responsive & Mobile Optimization

### 7.1 Mobile Navigation
- [ ] Hamburger menu: Smooth slide-in from left or top
- [ ] Full-screen overlay menu on mobile
- [ ] Large touch targets (minimum 44x44px)

### 7.2 Mobile Layout
- [ ] Single column layout for cards and forms
- [ ] Stack buttons vertically on small screens
- [ ] Enlarge text for readability (min 16px to prevent zoom)
- [ ] Optimize images for mobile bandwidth

### 7.3 Touch Interactions
- [ ] Swipe gestures for carousels/image galleries
- [ ] Pull-to-refresh for feeds/lists
- [ ] Touch-friendly dropdowns and date pickers

---

## Implementation Notes

### Design Tools
- Consider creating a design system document or Figma/Sketch file
- Use CSS variables for colors, spacing, typography for easy theming
- Component library: Consider React component library (e.g., Material-UI, Chakra UI) with HAWS theme customization

### CSS Architecture
- Use CSS modules or styled-components for scoped styling
- Establish spacing scale (4px, 8px, 16px, 24px, 32px, 48px)
- Create reusable utility classes for common patterns

### Testing
- Cross-browser testing: Chrome, Firefox, Safari, Edge
- Device testing: iOS Safari, Android Chrome
- Accessibility audit: Use axe DevTools or Lighthouse

### Performance
- Lazy load images and heavy components
- Code splitting for routes
- Optimize bundle size
- Implement caching strategies

---

## Success Metrics
- [ ] Volunteer feedback: Collect feedback on new design
- [ ] Task completion time: Measure if common tasks are faster
- [ ] Mobile usage: Track increase in mobile device usage
- [ ] Accessibility score: Achieve 95+ Lighthouse accessibility score

---

## Timeline Suggestion
- **Week 1-2**: Phase 1 (Colors, branding, typography)
- **Week 3-4**: Phase 2 (Layout, navigation, cards)
- **Week 5-6**: Phase 3 (Component polish)
- **Week 7**: Phase 4 (Imagery, icons)
- **Week 8**: Phase 5 (Accessibility, feedback)
- **Week 9-10**: Phase 6 (Page-specific enhancements)
- **Week 11**: Phase 7 (Mobile optimization)
- **Week 12**: Testing, refinement, deployment

---

## Resources
- **HAWS Brand Colors**: Extracted from hawspets.org
- **Design Inspiration**: hawspets.org (professional, warm, animal-focused)
- **Icon Library**: Heroicons, Feather Icons, or Font Awesome
- **Font Suggestions**: Montserrat, Open Sans, Inter, Roboto
- **Component Examples**: Material-UI, Chakra UI, Ant Design (for reference)

---

*This checklist is a living document. Update as design evolves and new requirements emerge.*
