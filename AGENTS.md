# Project Rules

## Language

All application text, UI labels and messages, documentation, and code comments
must be in English. New and updated content must follow this rule.

## Go Source Budget

Production Go source must not exceed 5,000 physical lines, including blank lines
and comments. Count project-owned `.go` files; exclude `*_test.go` files and
third-party dependencies.

Before implementing a feature that would exceed this limit, discuss it with the
user and obtain agreement on the approach. First look for unnecessary code to
remove, opportunities to simplify or reuse existing code, or ways to provide the
required functionality without adding production source lines.

Do not evade the limit by compressing code or moving application logic into
other file types. Keep the implementation readable and maintainable.
