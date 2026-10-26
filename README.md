# YTL Language Support

A powerful VS Code extension providing comprehensive syntax highlighting and deep structural analysis for **YTL (Yugol Token Language)** — a modern, indented data language designed for configuration and object mapping.

## Key Features

- **Robust 5-Level Indentation System:** Accurately parses nested nodes (`level0` through `level4`) using recursive context tracking, completely isolating property keys.
- **Native Network Primitives:** First-class syntax highlighting for IP addresses, CIDR blocks, and URLs directly inside your code—**without using strings or quotes**.
- **Hardware Address Detection:** Full support for three standard MAC address formats (`classic`, `alternative`, and `short`).
- **Comprehensive Chronological Types:** Supports four distinct date and time representations including ISO 8601 (with Zulu time and offsets), RFC 3339, and local calendar schemas.
- **Collision-Safe System Constants:** Logic literals (`@true`, `@false`), null tokens (`@null`, `@nil`), and calculation states (`@NaN`, `@Inf`) are strictly prefixed with `@` to prevent collisions with variable descriptors.
- **Advanced Data Patterns:** Native tokens for inheritance mappings (`from` keywords) and custom Enumerations (`enum`).

---

## Language Specification & Code Example

Save your configuration file with the `.ytl` extension. Below is an overview of how your structures look in the editor:

<p align="center">
  <img width="443" height="748" alt="photo_2026-10-26_17-43-12" src="https://github.com/user-attachments/assets/b90821af-fa1e-45ef-838f-64adfa3b1ed2" />
</p>

## Token Mapping

The parser intelligently maps syntax patterns into distinct semantic tokens:

| Category | Concrete Type | Context Example | TextMate Token Scope |
| :--- | :--- | :--- | :--- |
| **Nesting** | Core Classes | `YTL:` | `entity.name.class.ytl` |
| | Sub-tags / Blocks | `  Level2:` | `entity.name.tag.level2.ytl` |
| **Properties**| Variables | `  int_value:` | `variable.other.property.ytl` |
| **Networks** | IP / Subnets | `192.168.1.1` | `constant.other.network.ip.ytl` |
| | Remote URLs | `https://...` | `constant.other.network.url.ytl` |
| **Constants** | Boolean Literals | `@true`, `@false` | `constant.language.boolean.ytl` |
| | Null Pointers | `@null`, `@nil` | `constant.language.null.ytl` |
| | Special Math | `@NaN`, `@Inf` | `constant.numeric.special.ytl` |
| **OOP & Data**| Enumerations | `enum` | `constant.language.enum.ytl` |
| | Control Flows | `from` | `keyword.control.ytl` |

## Ecosystem & Parser Support

The table below tracks the current implementation status of the official YTL parsers and compilers across different programming languages:

| Programming Language | Parser Status | Package / Repository Link | Planned Version |
| :--- | :--- | :--- | :--- |
| **Go (Golang)** | 🟡 *In Development* | Coming Soon | v1.2.0 |
| **Rust** | ❌ *Not Implemented* | N/A | TBD |
| **Python** | ❌ *Not Implemented* | N/A | TBD |
| **JavaScript / TS** | ❌ *Not Implemented* | N/A | TBD |
| **C++** | ❌ *Not Implemented* | N/A | TBD |

*Want to build a YTL parser for your favorite language? Feel free to open an Issue or submit a Pull Request*

## Installation & Requirements

1. Install the extension in VS Code.
2. Open or create any file with the extension `.ytl`.
3. Syntax parsing and token matching will automatically initialize.
