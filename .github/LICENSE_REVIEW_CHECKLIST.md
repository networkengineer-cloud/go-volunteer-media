# License Review Checklist

This checklist should be used when adding new dependencies to ensure license compliance.

## ✅ License Review Completed (December 16, 2025)

### What Was Reviewed
- [x] All Go dependencies (go.mod)
- [x] All npm production dependencies (package.json)
- [x] All transitive/indirect dependencies
- [x] Docker base images

### What Was Created
- [x] THIRD_PARTY_LICENSES.md - Complete license documentation
- [x] Updated README.md with license section
- [x] This checklist for future reviews

### Findings
- [x] All dependencies use MIT-compatible licenses
- [x] No GPL or restrictive copyleft licenses found
- [x] All required attributions documented
- [x] Safe for commercial and proprietary use

---

## Checklist for Adding New Dependencies

### Before Adding Any Dependency

1. **Check the license**
   ```bash
   # For Go packages
   go list -m -json github.com/package/name | grep -i license
   
   # For npm packages
   npm view package-name license
   ```

2. **Verify license compatibility**
   - ✅ **Allowed**: MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC, 0BSD, MPL-2.0 (as library)
   - ⚠️  **Review carefully**: LGPL (requires dynamic linking), Unlicense, Public Domain
   - ❌ **Avoid**: GPL, AGPL, proprietary licenses without permission

3. **Check for dual licensing**
   - Some packages offer multiple license options (e.g., "MIT OR GPL-3.0")
   - Always use the MIT-compatible option

### After Adding a Dependency

1. **Update THIRD_PARTY_LICENSES.md**
   - Add dependency name and version
   - Add license type
   - Add repository URL
   - Add copyright notice if required
   - Add link to license text

2. **Test the application**
   - Ensure the new dependency works as expected
   - Verify no license headers or notices are missing

3. **Commit with proper message**
   ```bash
   git add go.mod go.sum THIRD_PARTY_LICENSES.md
   # or
   git add frontend/package.json frontend/package-lock.json THIRD_PARTY_LICENSES.md
   
   git commit -m "Add [package-name] ([license-type]) for [purpose]"
   ```

---

## Quick Reference: Compatible Licenses

### ✅ Fully Compatible (No Restrictions)

| License | Use Case | Notes |
|---------|----------|-------|
| MIT | All uses | Most common, very permissive |
| Apache-2.0 | All uses | Includes patent grant |
| BSD-2-Clause | All uses | Requires attribution |
| BSD-3-Clause | All uses | Requires attribution, no endorsement |
| ISC | All uses | Functionally equivalent to MIT |
| 0BSD | All uses | Public domain equivalent |

### ⚠️ Compatible with Conditions

| License | Use Case | Notes |
|---------|----------|-------|
| MPL-2.0 | As library only | File-level copyleft, safe as dependency |
| Zlib | All uses | Similar to MIT, permissive |
| CC0 | All uses | Public domain dedication |

### ❌ Incompatible (Avoid)

| License | Reason |
|---------|--------|
| GPL-2.0 / GPL-3.0 | Strong copyleft, requires source release |
| AGPL-3.0 | Network copyleft, requires source for SaaS |
| LGPL-2.1 / LGPL-3.0 | Requires dynamic linking, complex compliance |
| Proprietary | May have restrictions incompatible with MIT |

---

## Tools for License Checking

### Go Dependencies
```bash
# List all dependencies
go list -m all

# Check specific package (manual)
# Visit the repository and check LICENSE file
```

### npm Dependencies
```bash
# Install license-checker
npm install -g license-checker

# Check production dependencies
cd frontend && license-checker --production --csv

# Check for specific licenses
license-checker --production --onlyAllow "MIT;Apache-2.0;BSD;ISC"
```

### Automated Scanning
```bash
# Consider adding to CI/CD:
# - npm audit for npm packages
# - govulncheck for Go packages
# - FOSSA, Snyk, or similar for license compliance
```

---

## Emergency Procedures

### If GPL Dependency Is Found

1. **Stop immediately** - Do not deploy
2. **Assess impact**:
   - Is it direct or transitive?
   - Can we use a different library?
   - Is there a dual-license option?
3. **Options**:
   - Replace with MIT-compatible alternative
   - Negotiate dual-license agreement
   - Isolate as separate service with clear boundaries
4. **Document decision** in THIRD_PARTY_LICENSES.md

### If Proprietary/Unknown License Is Found

1. **Contact the package maintainer** for clarification
2. **Check package.json or go.mod** for license field
3. **Review repository** for LICENSE or COPYING file
4. **If unclear**: Do not use, find alternative

---

## Annual Review Process

Perform annual license review:

1. **Re-run license checks** on all dependencies
2. **Update THIRD_PARTY_LICENSES.md** with new versions
3. **Check for license changes** in major version updates
4. **Verify links** to license texts are still valid
5. **Update this checklist** with any new learnings

**Next Review Due**: December 2026

---

## Contacts

- **License Questions**: Consult legal team or open source compliance officer
- **Technical Questions**: See CONTRIBUTING.md
- **This Checklist Maintainer**: See git history of this file

---

*Last Updated: December 16, 2025*
*Last Full Review: December 16, 2025*
