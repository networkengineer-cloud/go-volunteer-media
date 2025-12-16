# Third-Party Software Licenses

This document contains information about third-party software used in the go-volunteer-media project and their respective licenses. This application is licensed under the MIT License, and all dependencies listed here are compatible with this license.

## License Compatibility Summary

All dependencies use licenses that are compatible with the MIT license used by this project:
- **MIT License**: Most permissive, allows commercial use, modification, distribution
- **Apache-2.0**: Permissive, compatible with MIT, includes patent grants
- **BSD-3-Clause**: Permissive, compatible with MIT
- **BSD-0-Clause (0BSD)**: Most permissive BSD variant, public domain equivalent
- **ISC**: Functionally equivalent to MIT
- **MPL-2.0**: Weak copyleft, compatible when used as library
- **Zlib**: Permissive, compatible with MIT

**✅ No GPL or incompatible licenses found** - All dependencies are safe for commercial and proprietary use.

---

## Backend Dependencies (Go)

### Direct Dependencies

#### github.com/gin-gonic/gin v1.11.0
- **License**: MIT
- **Purpose**: HTTP web framework
- **Repository**: https://github.com/gin-gonic/gin
- **License Text**: https://github.com/gin-gonic/gin/blob/master/LICENSE

#### github.com/go-playground/validator/v10 v10.27.0
- **License**: MIT
- **Purpose**: Struct and field validation
- **Repository**: https://github.com/go-playground/validator
- **License Text**: https://github.com/go-playground/validator/blob/master/LICENSE

#### github.com/golang-jwt/jwt/v5 v5.3.0
- **License**: MIT
- **Purpose**: JSON Web Token (JWT) implementation
- **Repository**: https://github.com/golang-jwt/jwt
- **License Text**: https://github.com/golang-jwt/jwt/blob/main/LICENSE

#### github.com/google/uuid v1.6.0
- **License**: BSD-3-Clause
- **Purpose**: UUID generation
- **Repository**: https://github.com/google/uuid
- **License Text**: https://github.com/google/uuid/blob/master/LICENSE

#### github.com/joho/godotenv v1.5.1
- **License**: MIT
- **Purpose**: Environment variable loading from .env files
- **Repository**: https://github.com/joho/godotenv
- **License Text**: https://github.com/joho/godotenv/blob/main/LICENSE

#### github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
- **License**: ISC
- **Purpose**: Image resizing
- **Repository**: https://github.com/nfnt/resize
- **License Text**: https://github.com/nfnt/resize/blob/master/LICENSE

#### github.com/stretchr/testify v1.11.1
- **License**: MIT
- **Purpose**: Testing toolkit
- **Repository**: https://github.com/stretchr/testify
- **License Text**: https://github.com/stretchr/testify/blob/master/LICENSE

#### golang.org/x/crypto v0.43.0
- **License**: BSD-3-Clause
- **Purpose**: Cryptographic libraries (bcrypt, etc.)
- **Repository**: https://cs.opensource.google/go/x/crypto
- **License Text**: https://cs.opensource.google/go/x/crypto/+/refs/heads/master:LICENSE

#### gorm.io/gorm v1.31.0
- **License**: MIT
- **Purpose**: ORM library
- **Repository**: https://github.com/go-gorm/gorm
- **License Text**: https://github.com/go-gorm/gorm/blob/master/LICENSE

#### gorm.io/driver/postgres v1.6.0
- **License**: MIT
- **Purpose**: PostgreSQL driver for GORM
- **Repository**: https://github.com/go-gorm/postgres
- **License Text**: https://github.com/go-gorm/postgres/blob/master/LICENSE

#### gorm.io/driver/sqlite v1.6.0
- **License**: MIT
- **Purpose**: SQLite driver for GORM
- **Repository**: https://github.com/go-gorm/sqlite
- **License Text**: https://github.com/go-gorm/sqlite/blob/master/LICENSE

### Notable Indirect Dependencies

#### github.com/bytedance/sonic v1.14.0
- **License**: Apache-2.0
- **Purpose**: High-performance JSON library
- **Repository**: https://github.com/bytedance/sonic
- **License Text**: https://github.com/bytedance/sonic/blob/main/LICENSE

#### github.com/jackc/pgx/v5 v5.6.0
- **License**: MIT
- **Purpose**: PostgreSQL driver and toolkit
- **Repository**: https://github.com/jackc/pgx
- **License Text**: https://github.com/jackc/pgx/blob/master/LICENSE

#### github.com/mattn/go-sqlite3 v1.14.22
- **License**: MIT
- **Purpose**: SQLite3 driver
- **Repository**: https://github.com/mattn/go-sqlite3
- **License Text**: https://github.com/mattn/go-sqlite3/blob/master/LICENSE

#### golang.org/x/* packages
- **License**: BSD-3-Clause
- **Purpose**: Extended Go standard library packages
- **Repository**: https://cs.opensource.google/go/x
- **Note**: All golang.org/x packages are part of the Go project and use the same BSD-3-Clause license

#### google.golang.org/protobuf v1.36.9
- **License**: BSD-3-Clause
- **Purpose**: Protocol Buffers implementation
- **Repository**: https://github.com/protocolbuffers/protobuf-go
- **License Text**: https://github.com/protocolbuffers/protobuf-go/blob/master/LICENSE

---

## Frontend Dependencies (npm)

### Direct Production Dependencies

#### axios v1.12.2
- **License**: MIT
- **Purpose**: HTTP client for API requests
- **Repository**: https://github.com/axios/axios
- **Copyright**: Copyright (c) 2014-present Matt Zabriskie and contributors

#### docx-preview v0.3.7
- **License**: Apache-2.0
- **Purpose**: DOCX document preview/rendering
- **Repository**: https://github.com/VolodymyrBaydalka/docxjs
- **Copyright**: Copyright (c) Volodymyr Baydalka

#### dompurify v3.2.3
- **License**: MPL-2.0 OR Apache-2.0 (Dual-licensed)
- **Purpose**: XSS sanitizer for HTML
- **Repository**: https://github.com/cure53/DOMPurify
- **Copyright**: Copyright (c) 2015 Mario Heiderich
- **Note**: Used under Apache-2.0 license terms, which is compatible with MIT

#### pdfjs-dist v4.9.155
- **License**: Apache-2.0
- **Purpose**: PDF rendering library from Mozilla
- **Repository**: https://github.com/mozilla/pdf.js
- **Copyright**: Copyright (c) Mozilla Foundation

#### react v19.2.0
- **License**: MIT
- **Purpose**: UI library
- **Repository**: https://github.com/facebook/react
- **Copyright**: Copyright (c) Meta Platforms, Inc. and affiliates

#### react-dom v19.2.0
- **License**: MIT
- **Purpose**: React DOM renderer
- **Repository**: https://github.com/facebook/react
- **Copyright**: Copyright (c) Meta Platforms, Inc. and affiliates

#### react-easy-crop v5.5.6
- **License**: MIT
- **Purpose**: Image cropping component
- **Repository**: https://github.com/ValentinH/react-easy-crop
- **Copyright**: Copyright (c) 2018 Valentin Hervieu

#### react-router-dom v7.9.4
- **License**: MIT
- **Purpose**: Routing library for React
- **Repository**: https://github.com/remix-run/react-router
- **Copyright**: Copyright (c) React Training LLC 2015-2019, Remix Software Inc. 2020-2025

### Notable Indirect Production Dependencies

#### jszip v3.10.1
- **License**: MIT OR GPL-3.0-or-later (Dual-licensed)
- **Purpose**: ZIP file creation and reading
- **Repository**: https://github.com/Stuk/jszip
- **Note**: Used under MIT license terms, which is compatible with this project

#### pako v1.0.11
- **License**: MIT AND Zlib
- **Purpose**: Compression library (used by jszip)
- **Repository**: https://github.com/nodeca/pako
- **Note**: Both MIT and Zlib licenses are permissive and compatible

#### normalize-wheel v1.0.1
- **License**: BSD-3-Clause
- **Purpose**: Mouse wheel event normalization
- **Repository**: https://github.com/basilfx/normalize-wheel

#### tslib v2.8.1
- **License**: 0BSD (BSD Zero Clause)
- **Purpose**: TypeScript helper library
- **Repository**: https://github.com/Microsoft/tslib
- **Copyright**: Copyright (c) Microsoft Corporation
- **Note**: 0BSD is a public domain equivalent license, extremely permissive

---

## License Texts

### MIT License
```
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### Apache License 2.0 (Summary)
```
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

### BSD-3-Clause License (Summary)
```
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice,
   this list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.
3. Neither the name of the copyright holder nor the names of its contributors
   may be used to endorse or promote products derived from this software
   without specific prior written permission.
```

### ISC License (Summary)
```
Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.
```

---

## Attribution Requirements

The following packages require specific attribution when distributed:

### React (MIT License)
Copyright (c) Meta Platforms, Inc. and affiliates.

### React Router (MIT License)
Copyright (c) React Training LLC 2015-2019  
Copyright (c) Remix Software Inc. 2020-2025

### DOMPurify (Apache-2.0 / MPL-2.0)
Copyright (c) 2015 Mario Heiderich

### PDF.js (Apache-2.0)
Copyright (c) Mozilla Foundation

### DOCX Preview (Apache-2.0)
Copyright (c) Volodymyr Baydalka

### Axios (MIT License)
Copyright (c) 2014-present Matt Zabriskie and contributors

### TypeScript Library - tslib (0BSD)
Copyright (c) Microsoft Corporation

### Google UUID (BSD-3-Clause)
Copyright (c) 2009, 2014 Google Inc. All rights reserved.

### Golang Extended Packages (BSD-3-Clause)
Copyright (c) 2009 The Go Authors. All rights reserved.

---

## Compliance Notes

1. **MIT License Compliance**: For all MIT-licensed dependencies, this document serves as the required attribution notice. The MIT license text is included above, and copyright notices for each package are listed.

2. **Apache-2.0 Compliance**: Apache-licensed dependencies (docx-preview, pdfjs-dist, dompurify, bytedance/sonic) are compatible with MIT. Attribution and license notices are provided above. No NOTICE files require inclusion for these packages.

3. **BSD-3-Clause Compliance**: BSD-licensed dependencies (Google UUID, golang.org/x packages) are compatible with MIT. Copyright notices and license summary are provided above.

4. **Dual-Licensed Packages**: 
   - **jszip**: Used under MIT license terms
   - **dompurify**: Used under Apache-2.0 license terms

5. **No GPL Dependencies**: This project intentionally avoids GPL-licensed dependencies to maintain maximum compatibility with proprietary and commercial use cases.

---

## Verification

This license audit was performed on 2025-12-16. To verify current dependencies:

### Backend (Go)
```bash
go list -m all
```

### Frontend (npm)
```bash
cd frontend && npx license-checker --production --csv
```

---

## Updates

When adding new dependencies, please:
1. Verify the license is compatible with MIT
2. Add the dependency to this document with its license information
3. Ensure proper attribution is included if required by the license
4. Avoid GPL or other copyleft licenses unless absolutely necessary

---

*Last Updated: December 16, 2025*
