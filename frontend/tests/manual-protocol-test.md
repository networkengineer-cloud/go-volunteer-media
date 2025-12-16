# Manual Protocol Document Mobile Testing Guide

## Test Checklist for Mobile Responsive Protocol Viewer

### Setup
1. Ensure backend is running: `go run cmd/api/main.go`
2. Ensure frontend is running: `cd frontend && npm run dev`
3. Login as admin (username: `admin`, password: `demo1234`)
4. Navigate to any animal with a protocol document

### Desktop Testing (1280px+)

**Expected Behavior:**
- [ ] Download button visible in modal header
- [ ] Mobile hint is NOT visible
- [ ] Modal has rounded corners and padding
- [ ] Modal is centered with max-width constraint
- [ ] Protocol document displays clearly (PDF in iframe or DOCX rendered)

### Tablet Testing (768px - 1024px)

**Expected Behavior:**
- [ ] Download button visible and full-width
- [ ] Mobile hint IS visible
- [ ] Modal takes full screen (width and height)
- [ ] No rounded corners on modal
- [ ] Header elements stack vertically
- [ ] Close button positioned absolutely in top-right

**Test on:**
- [ ] iPad (768 x 1024)
- [ ] iPad Pro (1024 x 1366)

### Phone Testing (< 768px)

**Expected Behavior:**
- [ ] Download button visible, full-width, prominent
- [ ] Mobile hint IS visible with helpful text
- [ ] Modal takes full screen (100vw x 100vh)
- [ ] No horizontal scrolling on page
- [ ] DOCX content fits within viewport (no cutoff on sides)
- [ ] PDF iframe has minimum height of 400px
- [ ] Header stacks vertically with close button in top-right
- [ ] Can download document successfully

**Test on:**
- [ ] iPhone 12 (390 x 844)
- [ ] iPhone SE (375 x 667)
- [ ] Pixel 5 (393 x 851)
- [ ] Galaxy S21 (360 x 800)

### DOCX-Specific Tests

**On Mobile (< 768px):**
- [ ] DOCX tables fit within viewport (no horizontal overflow)
- [ ] DOCX images scale to fit viewport
- [ ] DOCX text is readable without zooming
- [ ] Horizontal scrolling is NOT needed to read content
- [ ] Can use pinch-to-zoom if needed

### PDF-Specific Tests

**On Mobile (< 768px):**
- [ ] PDF iframe displays at minimum 400px height
- [ ] Download button works and triggers download
- [ ] Mobile hint suggests using download if display issues occur

### Download Button Tests

**On All Devices:**
- [ ] Download button has proper icon and label
- [ ] Clicking download button triggers file download
- [ ] Downloaded file has correct filename
- [ ] Downloaded file opens correctly in native viewer
- [ ] Button has proper focus state (keyboard accessible)
- [ ] Button has proper hover state (desktop)

### Accessibility Tests

**Keyboard Navigation:**
- [ ] Can tab to download button
- [ ] Can activate download with Enter/Space
- [ ] Can close modal with Escape key
- [ ] Focus is trapped within modal when open

**Screen Reader:**
- [ ] Download button has aria-label "Download protocol document"
- [ ] Close button has aria-label "Close protocol document"
- [ ] Modal has role="dialog" and aria-modal="true"
- [ ] Mobile hint is announced when visible

### Browser Compatibility

**Test in:**
- [ ] Chrome (desktop and mobile)
- [ ] Safari (desktop and iOS)
- [ ] Firefox (desktop)
- [ ] Edge (desktop)

## Known Issues to Verify Fixed

1. **DOCX portions cut off on phone** - FIXED
   - Verify: Open DOCX protocol on iPhone 12 viewport
   - Expected: Content fits within viewport, no horizontal scrolling needed

2. **PDF not working on tablet/phone** - FIXED
   - Verify: Open PDF protocol on mobile device
   - Expected: Download button provides reliable access
   - Fallback: If iframe doesn't work, download works

## Screenshots to Take

For PR documentation:
1. Desktop view (1280px) - Modal with protocol
2. Tablet view (768px) - Full-screen modal
3. Phone view (375px) - Full-screen modal with download button
4. Phone view (375px) - DOCX content fitting within viewport
5. Phone view (375px) - Mobile hint visible

## Test Results

Date: ___________
Tester: ___________

### Issues Found:
- None / List issues below:

### Notes:
