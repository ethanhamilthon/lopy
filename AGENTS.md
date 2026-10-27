# Project Rules

## Language

All application text, UI labels and messages, documentation, and code comments
must be in English. New and updated content must follow this rule.

## Go Source Budget

Target 5,000 physical lines of production Go source, including blank lines and
comments. Count project-owned `.go` files; exclude `*_test.go` files and
third-party dependencies.

Hard cap: 5,500 lines. The 500-line gap above the target is reserve for work that
is already agreed, not a licence to grow. Treat 5,000 as the working limit: plan,
simplify and review against it, and mention the current count when it moves.

Before implementing a feature that would exceed the target, discuss it with the
user and obtain agreement on the approach. First look for unnecessary code to
remove, opportunities to simplify or reuse existing code, or ways to provide the
required functionality without adding production source lines.

Do not evade the limit by compressing code or moving application logic into
other file types. Keep the implementation readable and maintainable.
