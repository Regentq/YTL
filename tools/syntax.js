// Generator for the YTL syntax highlighting.
//
//   node tools/syntax.js
//
// It writes two things:
//   1. syntaxes/ytl.tmLanguage.json   (the grammar: what is an IP, a date, a string...)
//   2. the color rules inside package.json (everything else in package.json is kept)
//
// Edit THIS file, never the generated JSON. Here you can:
//   - write regular expressions with single backslashes (no \\ doubling),
//   - give pieces names and reuse them,
//   - write comments.
// After every change run the command above and press F5 / reload the window.
//
// IMPORTANT IDEA: the grammar recognises a value by how it LOOKS, exactly like the
// language server does. A broken IP address still looks like an IP address, so it keeps
// the IP color; the red squiggle from the server tells the user what is wrong.
// Nothing here is ever colored red.

const fs = require('fs');
const path = require('path');

// The optional first argument is the extension folder (used only for testing).
const root = process.argv[2] ? path.resolve(process.argv[2]) : path.join(__dirname, '..');

// r`...` keeps backslashes exactly as written: r`\d+` is the regex \d+
const r = String.raw;

// =====================================================================
// 1. REUSABLE PIECES
// =====================================================================
const ident = r`[a-zA-Z_][a-zA-Z0-9_]*`; // a name: letters, digits, _

// The colon of a header: nothing may follow it except spaces or a comment that
// is separated by whitespace. So "SS ://google.com" is NOT a header.
const headerEnd = r`:(?=[ \t]*$|[ \t]+//.*$)`;

// A character that can belong to a value (not a space, comma or closing bracket).
const tok = r`[^\s,\]}]`;

// A value ends at whitespace, a comma, a bracket or the end of the line.
const tokenEnd = r`(?=[\s,\]}]|$)`;

// A section header "Name:" (the leading indentation is captured as group 1).
const blockBegin = r`^([ \t]*)(${ident})[ \t]*${headerEnd}`;
// The section ends at the first line that is not indented deeper than the header.
const blockEnd = r`^(?!\1[ \t]+)\s*(?=[\S])`;

const include = (name) => ({ include: name });

// =====================================================================
// 2. THE GRAMMAR (what is what)
// =====================================================================
const grammar = {
  $schema: 'https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json',
  name: 'YTL',

  // The order matters: when two rules start at the same place, the first one wins.
  patterns: [
    include('#comments'),
    include('#quoted-key'), // before #strings: it wins when both start on the same column
    include('#strings'),
    include('#enum_values'),
    include('#keywords'),
    include('#block'),
    include('#entities'),
    include('#properties'),
    include('#loose-keys'),
    include('#constants'),
    include('#mac-addresses'),
    include('#dates'), // before #networks: 2026-10-26T20:43:12+03:00 has many colons
    include('#networks'),
    include('#numbers'),
    include('#brackets'),
    include('#unknown'), // must stay LAST: it only catches what nothing above recognised
  ],

  repository: {
    comments: {
      // Only at the start of a line or after whitespace: the "//" inside
      // https://site.com is NOT a comment.
      patterns: [{ name: 'comment.line.double-slash.ytl', match: r`(?<!\S)//.*$` }],
    },

    // "Name" "value"  -> keys are written without quotes (the server reports it);
    // the key still gets the key color.
    'quoted-key': {
      patterns: [
        {
          match: r`^[ \t]*("(?:[^"\\]|\\.)*")(?=[ \t]+[^\s/])`,
          captures: { 1: { name: 'variable.other.property.ytl' } },
        },
      ],
    },

    keywords: {
      patterns: [{ name: 'keyword.control.ytl', match: r`\b(from|enum)\b` }],
    },

    // enum Name
    enum_values: {
      patterns: [
        {
          match: r`\b(enum)\s+(${ident})\b`,
          captures: {
            1: { name: 'keyword.control.ytl' },
            2: { name: 'constant.language.enum.ytl' },
          },
        },
      ],
    },

    // Nested sections get a different color per level (up to 3 levels).
    block: {
      begin: blockBegin,
      beginCaptures: { 2: { name: 'entity.name.tag.level1.ytl' } },
      end: blockEnd,
      patterns: [
        {
          begin: blockBegin,
          beginCaptures: { 2: { name: 'entity.name.tag.level2.ytl' } },
          end: blockEnd,
          patterns: [
            {
              begin: blockBegin,
              beginCaptures: { 2: { name: 'entity.name.tag.level3.ytl' } },
              end: blockEnd,
              patterns: [include('#block'), include('$self')],
            },
            include('$self'),
          ],
        },
        include('$self'),
      ],
    },

    // Top-level headers:  "Dog from Animal:"  and  "Animal:"
    entities: {
      patterns: [
        {
          match: r`^(${ident})\s+(from)\s+(${ident})\s*${headerEnd}`,
          captures: {
            1: { name: 'entity.name.class.ytl' },
            2: { name: 'keyword.control.ytl' },
            3: { name: 'entity.other.inherited-class.ytl' },
          },
        },
        { name: 'entity.name.class.ytl', match: r`^${ident}\b(?=\s*:(?=[ \t]*$|[ \t]+//.*$))` },
      ],
    },

    // A key at the start of a line (a whole word).
    properties: {
      patterns: [
        {
          match: r`^[ \t]*(${ident})(?=[ \t\[]|$)(?!\s*${headerEnd})`,
          captures: { 1: { name: 'variable.other.property.ytl' } },
        },
      ],
    },

    // A key with a wrong name (42Name, Name-With-Dash) still looks like a key.
    // It must be followed by a value, so a lone array element is not taken for a key.
    'loose-keys': {
      patterns: [
        {
          match: r`^[ \t]*([^\s\[\]"@#:,/]+)(?=[ \t]+[^\s/])`,
          captures: { 1: { name: 'variable.other.property.ytl' } },
        },
      ],
    },

    dates: {
      patterns: [
        // 2026-10-26T17:43:12Z  /  2026-10-26T20:43:12+03:00
        { name: 'constant.other.date.ytl', match: r`\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})\b` },
        // 17:43:12Z
        { name: 'constant.other.date.ytl', match: r`\b\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})\b` },
        // 2026-10-26  /  2026-10-26 17:43
        { name: 'constant.other.date.ytl', match: r`\b\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2}(?::\d{2})?)?\b` },
        // 26.10.2026
        { name: 'constant.other.date.ytl', match: r`\b\d{1,2}[./]\d{1,2}[./]\d{4}\b` },
        // 17:43  /  17:43:12
        { name: 'constant.other.time.ytl', match: r`\b\d{1,2}:\d{2}(?::\d{2})?\b` },
      ],
    },

    numbers: {
      patterns: [
        // #88ff00   (also #12345: the wrong number of digits is reported by the server)
        { name: 'constant.numeric.hex.ytl', match: r`#[0-9A-Za-z]+` },
        // 0x59544C00
        { name: 'constant.numeric.hex.ytl', match: r`\b0[xX][0-9A-Za-z]*` },
        // 102  /  10.2  /  -5
        { name: 'constant.numeric.ytl', match: r`(?<!\d|\.)-?\d+(?:\.\d+)?\b(?![.:-][0-9a-fA-F])` },
        // 1.2.0-rs   (also 01.2.3)
        {
          name: 'constant.numeric.version.ytl',
          match: r`(?<![\w.])\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.+-]*)?${tokenEnd}`,
        },
        // 14.5s  14d  14ms   (also an unknown unit, like 14x)
        { name: 'constant.numeric.duration.ytl', match: r`(?<![\w.])\d+(?:\.\d+)?[a-zA-Z]+${tokenEnd}` },
      ],
    },

    constants: {
      patterns: [
        { name: 'constant.language.boolean.ytl', match: r`@(true|false)\b` },
        { name: 'constant.language.null.ytl', match: r`@(null|nil|none)\b` },
        { name: 'constant.numeric.special.ytl', match: r`@(NaN|Inf)\b` },
      ],
    },

    networks: {
      patterns: [
        // https://example.com/a   (also a broken one: http://  https://google..com  ://x)
        { name: 'constant.other.network.url.ytl', match: r`\bhttps?://${tok}*` },
        { name: 'constant.other.network.url.ytl', match: r`(?<!\w)://${tok}*` },

        // 127.0.0.1:8080   (dotted numbers followed by ":")
        { name: 'constant.other.network.socket.ipv4.ytl', match: r`(?<![\w.])\d+(?:\.\d+){2,}:${tok}*` },

        // 127.0.0.1  /  10.0.0.0/24   (also 10.0.0/16 and 192.168.1.300/24)
        {
          name: 'constant.other.network.ip.ipv4.ytl',
          match: r`(?<![\w.])(?:\d+(?:\.\d+){3,}(?:/${tok}*)?|\d+\.\d+\.\d+/${tok}*)`,
        },

        // [2001:db8::1]:8080   (brackets with a colon inside)
        {
          name: 'constant.other.network.socket.ipv6.ytl',
          match: r`(?<![\w\]])\[(?=[^\s\]]*:)[^\s\]]*\](?::${tok}*)?`,
        },

        // 2001:db8::/32   (needs "::" or three colons, so 17:43:12 stays a time)
        {
          name: 'constant.other.network.ip.ipv6.ytl',
          match: r`(?<![\w:.])(?!(?:[0-9A-Za-z]{2}:){2,})(?=${tok}*?(?:::|(?::[^\s:,\]}/]*){3}))[0-9A-Za-z.:%]+(?:/${tok}*)?`,
        },
      ],
    },

    'mac-addresses': {
      patterns: [
        // 00:1A:2B:3C:4D:5E   (also with the wrong number of pairs or a wrong digit)
        { name: 'constant.numeric.mac.classic.ytl', match: r`(?<![\w:-])[0-9A-Za-z]{2}:[0-9A-Za-z]{2}(?:[:-][0-9A-Za-z]{2}|::[0-9A-Za-z]{1,2}){1,}(?![\w:-])` },
        // 00-0a-2b-3c-4d-5e
        { name: 'constant.numeric.mac.alternative.ytl', match: r`(?<![\w:-])[0-9A-Za-z]{2}-[0-9A-Za-z]{2}(?:[:-][0-9A-Za-z]{2}){2,}(?![\w:-])` },
        // 001a.2b3c.4d5e   (also 001A.2B3C.4D5; it needs a letter, so 100.200.300 stays a version)
        { name: 'constant.numeric.mac.short.ytl', match: r`(?<![\w.])(?=[0-9.]*[A-Fa-f])[0-9A-Fa-f]{3,4}\.[0-9A-Fa-f]{3,4}\.[0-9A-Fa-f]{3,4}(?![\w.])` },
      ],
    },

    // [ and ] of arrays:  os[1.0, 100]
    brackets: {
      patterns: [{ name: 'punctuation.definition.array.ytl', match: r`[\[\]]` }],
    },

    // Anything else (for example an unquoted word) keeps the normal text color.
    unknown: {
      patterns: [{ name: 'meta.unknown-value.ytl', match: r`[^\s,\[\]"]+` }],
    },

    strings: {
      patterns: [
        // """ multi-line """
        {
          name: 'string.quoted.triple.ytl',
          begin: '"""',
          end: '"""',
          patterns: [{ name: 'constant.character.escape.ytl', match: r`\\.` }],
        },
        // "normal"
        {
          name: 'string.quoted.double.ytl',
          begin: '"',
          end: '"',
          patterns: [{ name: 'constant.character.escape.ytl', match: r`\\.` }],
        },
      ],
    },
  },

  scopeName: 'source.ytl',
};

// =====================================================================
// 3. COLORS   [scope, color, optional font style]
// =====================================================================
// One line per color. A scope is the name used in the grammar above.
const colors = [
  ['comment.line.double-slash.ytl', '#5C6370', 'italic'],

  // sections and keys
  ['entity.name.class.ytl', '#E5C07B', 'bold'], //   top-level section name
  ['entity.name.tag.level1.ytl', '#E5C07B'], //      nested section, level 1
  ['entity.name.tag.level2.ytl', '#5dd47b'], //      nested section, level 2
  ['entity.name.tag.level3.ytl', '#6bc5da'], //      nested section, level 3
  ['variable.other.property.ytl', '#61AFEF'], //     key
  ['keyword.control.ytl', '#C678DD'], //             from, enum
  ['constant.language.enum.ytl', '#81a2ff', 'bold'],

  // numbers, dates
  ['constant.numeric.ytl', '#f5b641'],
  ['constant.numeric.hex.ytl', '#4996ee'],
  ['constant.other.date.ytl', '#56B6C2'],
  ['constant.other.time.ytl', '#28a1f1'],

  // strings
  ['string.quoted.double.ytl', '#98C379'],
  ['constant.character.escape.ytl', '#D19A66', 'bold'],

  // network
  ['constant.other.network.ip.ipv4.ytl', '#acdf5b'],
  ['constant.other.network.socket.ipv4.ytl', '#8bb34c'],
  ['constant.other.network.ip.ipv6.ytl', '#57abc5'],
  ['constant.other.network.socket.ipv6.ytl', '#4991a7'],
  ['constant.other.network.url.ytl', '#00ffbf', 'underline'],

  // constants
  ['constant.language.boolean.ytl', '#ff4b81'],
  ['constant.language.null.ytl', '#ff9100'],
  ['constant.numeric.special.ytl', '#00e1ff'],

  // MAC addresses
  ['constant.numeric.mac.classic.ytl', '#4cff9d'],
  ['constant.numeric.mac.alternative.ytl', '#4cff9d'],
  ['constant.numeric.mac.short.ytl', '#4cff9d'],
];

// =====================================================================
// 4. WRITE THE FILES (nothing to edit below)
// =====================================================================
const grammarPath = path.join(root, 'syntaxes', 'ytl.tmLanguage.json');
fs.mkdirSync(path.dirname(grammarPath), { recursive: true });
fs.writeFileSync(grammarPath, JSON.stringify(grammar, null, 4) + '\n');

const pkgPath = path.join(root, 'package.json');
const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));

pkg.contributes = pkg.contributes || {};
pkg.contributes.configurationDefaults = pkg.contributes.configurationDefaults || {};
pkg.contributes.configurationDefaults['editor.tokenColorCustomizations'] = {
  textMateRules: colors.map(([scope, foreground, fontStyle]) => ({
    scope,
    settings: fontStyle ? { foreground, fontStyle } : { foreground },
  })),
};

fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');

console.log('written: syntaxes/ytl.tmLanguage.json and the colors in package.json');