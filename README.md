# YTL Language Support (1.2.0)
<img src="icon.jpg" width="96" height="96" align="right" />


A powerful VS Code extension providing comprehensive syntax highlighting and deep structural analysis for **YTL (Yggdrasil Template Language)** — a modern, indented data language designed for configuration and object mapping.

## Key Features

- **Infinite Indentation System:** Seamlessly create new nodes without ever worrying about limit constraints.
- **Built-in Network Format Recognition:** Instantly detects and parses critical network entities including IPv4, IPv6, MAC addresses, ports, sockets, URLs, and emails.
- **Multiple Data Inheritance:** Allows merging multiple parent structures simultaneously (e.g., `from Net, Time`), giving you maximum flexibility without code duplication.
- **Automated Structure Expansion:** Instantly populates inherited properties and nodes from parent structures upon declaration, eliminating manual boilerplate setup.


## Language Specification & Code Example

Save your configuration file with the `.ytl` extension. Below is an overview of how your structures look in the editor:

```puppet
YTL:
    name "YTL (Yggdrasil Template Language)"
    version_id 1.2.0

    Network_base:
        IPv4[
            0.0.0.0, 127.0.0.1:80, 0.0.0.0:8080
        ]

        IPv6[
            [::1]:8080, fe80::, 2001:4860:4860::
        ]

        CIDR    0.0.0.0/0, ::/0
        URI     127.0.0.1:8000
        URL     http://0.0.0.0
        MAC     00:1A:2B:3C:4D:5E, 00-1A-2B-3C-4D-5E, 001A.2B3C.4D5E

    Date:
        ISO_ZULU_Time   2026-10-26T19:50:47Z
        RFC_OFFSET_Time 2026-10-26T14:20:00+03:30
        Local_calendar  2026-10-26
        Time            14.5s
```

Inheritance Example:


```puppet
Date:
    Time 2026-11-26T14:30
    Time2 1995-06-21

Date2 from Date:
Date2:
    Time 2021-11-26T14:10
    Time2 1999-01-10
```

## Implementations

| Language | Repo | v1.2.0 | Remarks |
| :--- | :--- | :--- | :--- |
| **Go** | https://github.com/Regentq/YTL-GO | ✔️ | WIP |
| **C** | *Not implemented* | N/A | N/A |
| **C++** | *Not implemented* | N/A | N/A |
| **C#** | *Not implemented* | N/A | N/A |

### Contributing

You can become a part of the YTL development team simply by finding bugs and submitting them! Feel free to open an issue or submit a pull request if you want to help improve the YTL syntax or its parser implementations :)
