# MIT License

Copyright (c) 2024 casjay-forks

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

---

# Third-Party Licenses and Attributions

CasPaste is built upon and includes code from various open-source projects. This document provides attribution and license information for all third-party components.

---

## Go Dependencies

### Chroma - Syntax Highlighting
- **Package:** `github.com/alecthomas/chroma/v2`
- **Version:** v2.4.0
- **License:** MIT License
- **Copyright:** © Alec Thomas
- **Source:** https://github.com/alecthomas/chroma

### PostgreSQL Driver
- **Package:** `github.com/jackc/pgx/v5`
- **Version:** v5.7.2
- **License:** MIT License
- **Copyright:** © 2013-2024 Jack Christensen
- **Source:** https://github.com/jackc/pgx

### MySQL/MariaDB Driver
- **Package:** `github.com/go-sql-driver/mysql`
- **Version:** v1.7.1
- **License:** Mozilla Public License 2.0 (MPL-2.0)
- **Copyright:** © 2012-2023 The Go-MySQL-Driver Authors
- **Source:** https://github.com/go-sql-driver/mysql

### SQLite Driver (Pure Go)
- **Package:** `modernc.org/sqlite`
- **Version:** v1.28.0
- **License:** BSD-3-Clause
- **Copyright:** © 2017 The Sqlite Authors
- **Source:** https://gitlab.com/cznic/sqlite

### Go Crypto Libraries
- **Package:** `golang.org/x/crypto`
- **Version:** v0.18.0
- **License:** BSD-3-Clause
- **Copyright:** © The Go Authors
- **Source:** https://golang.org/x/crypto
- **Used for:** Argon2id password hashing, bcrypt support

### Go Net Library
- **Package:** `golang.org/x/net`
- **Version:** v0.20.0
- **License:** BSD-3-Clause
- **Copyright:** © The Go Authors
- **Source:** https://golang.org/x/net
- **Used for:** Public Suffix List domain validation

### Go Term Library
- **Package:** `golang.org/x/term`
- **Version:** v0.16.0
- **License:** BSD-3-Clause
- **Copyright:** © The Go Authors
- **Source:** https://golang.org/x/term
- **Used for:** Terminal password input

### YAML Parser
- **Package:** `gopkg.in/yaml.v3`
- **Version:** v3.0.1
- **License:** MIT License + Apache License 2.0
- **Copyright:** © 2011-2019 Canonical Ltd., © 2006-2010 Kirill Simonov
- **Source:** https://github.com/go-yaml/yaml

---

## Indirect Dependencies

The following packages are used indirectly by our direct dependencies:

- `github.com/dlclark/regexp2` - BSD-3-Clause
- `github.com/dustin/go-humanize` - MIT
- `github.com/google/uuid` - BSD-3-Clause
- `github.com/kballard/go-shellquote` - MIT
- `github.com/mattn/go-isatty` - MIT
- `github.com/remyoudompheng/bigfft` - BSD-3-Clause
- `modernc.org/*` packages - BSD-3-Clause
- `golang.org/x/*` packages - BSD-3-Clause
- `lukechampine.com/uint128` - MIT

See `go.mod` and `go.sum` for complete dependency list with versions.

---

## Design Inspiration

### MicroBin
- **Source:** https://github.com/szabodanika/microbin
- **License:** BSD-3-Clause
- **Inspiration:** File upload support, URL shortening, QR codes, editable pastes, private pastes

CasPaste implements similar features to MicroBin but with independent implementation in Go.

---

## Fonts and Resources

### System Fonts
Uses system default fonts with fallbacks:
- `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`
- `"SF Mono", "Fira Code", "JetBrains Mono", Consolas, monospace`

No embedded fonts - relies on user's system fonts.

### QR Code Generation
- Uses Google Charts API for QR code generation
- No local QR code library dependencies

---

## Theme Color Schemes

The following color schemes inspired our built-in themes:

- **Dracula Theme:** https://draculatheme.com/ - MIT License
- **Nord Theme:** https://www.nordtheme.com/ - MIT License
- **Gruvbox:** https://github.com/morhetz/gruvbox - MIT License
- **Tokyo Night:** https://github.com/tokyo-night/tokyo-night-vscode-theme - MIT License
- **Catppuccin:** https://github.com/catppuccin/catppuccin - MIT License
- **One Dark:** https://github.com/atom/atom/tree/master/packages/one-dark-ui - MIT License
- **Solarized:** https://ethanschoonover.com/solarized/ - MIT License
- **GitHub:** https://github.com/primer/css - MIT License

All color values were independently selected for optimal readability and mobile-first design.

---

## License Compliance

### Distribution Requirements

When distributing CasPaste, you must:

1. **Include this LICENSE.md file** with all attributions
2. **Comply with all third-party licenses** listed above

### Contribution Guidelines

By contributing to CasPaste, you agree that your contributions will be licensed under the MIT License.

---

## Contact

For license questions or concerns:
- **Repository:** https://github.com/casjay-forks/caspaste
- **Issues:** https://github.com/casjay-forks/caspaste/issues

---

**Last Updated:** 2026-01-14
