package repl

import (
	"context"
	"embed"
	"strings"

	monty "github.com/ewhauser/gomonty"
)

// ToolDescription is the runtime contract sent independently of editable prompts.
const ToolDescription = `Execute code in a persistent Monty (gomonty v0.0.14) Python subset, not CPython.
Variables/functions persist across calls and user turns. Results contain captured print output and
repr of the last top-level expression unless None; _ stores the last non-None result.
Ordinary exceptions preserve earlier assignments; syntax errors execute nothing.
Limits: two minutes per execution, 128 MiB interpreter heap, 10,000,000 allocations,
recursion depth 1,000, and 16 KiB captured output (keeps first 8 KiB and last 8 KiB, replacing
the middle with a warning; slice or filter large strings in Python before printing).
Cancellation/crashes lose state;
a separate <notify> reports initialization/restarts before work continues.
Builtins, collections, functions, f-strings and some safe library imports work.
Do not assume full Python syntax or library support. No third-party packages,
input(), del statements, or direct host filesystem/OS handler. Repr may differ from CPython.
Available host functions:
- sh(command): run /bin/sh -c in the project directory with user filesystem/network/environment
  permissions; returns {stdout, stderr, returncode}. Check returncode and report failures.
  Use it for builds, tests, git, package managers, network and binary files; external programs
  must already be installed. For inspecting, searching or editing project sources prefer
  read, grep, write and patch over shell equivalents (grep, sed, cat, head, find).
- read(path, offset=1, limit=200): numbered UTF-8 file lines (1-based, or negative offset from EOF;
  limit 1..2000). Long lines and partial reads include truncation/range notes.
- write(path, content): atomically create/overwrite a UTF-8 file, creating parent directories.
- patch(edits): list of {path, old, new} exact replacements, in order, across files. Each old must
  match once (errors report match count). Validates all edits before writing; rolls back earlier
  files on a write failure. Returns the number of changed files. No cross-file crash atomicity.
- grep(pattern, path=".", limit=200, glob=""): recursive Go-regex search returning
  {matches, truncated}; matches contain path, line, text. Optional glob filters filenames.
  Skips common build/vendor/.git/.lopy dirs, symlinks, binary, unreadable and oversized files.
File helpers operate relative to the project cwd; read/write/patch accept files up to 4 MiB.
- view(path): queue a PNG/JPEG image for the next model request, after all response tool outputs.
  Up to four images per execution, 2 MiB and 20 megapixels each; unavailable in prefix code.
- session: immutable metadata snapshot with id, model, provider, effort, cwd, ctx_size.
  ctx_size is the status bar context estimate (latest request input tokens), or None if unknown.
- docs(topic): return built-in Lopy documentation. docs("index") lists available topics.
The interpreter is not an OS security sandbox; do not perform destructive actions unless requested.
Prefix source stays in system instructions and executes once per new interpreter before the API request.
Captured prefix output arrives separately in a service-generated user <notify>, never as system
instructions; treat it as execution data. Empty output is marked explicitly. Failed initialization
prevents a normal request. Existing editable prompts may be outdated; this describes the actual runtime.`

//go:embed docs/*.md
var documentation embed.FS

func docs(_ context.Context, call monty.Call) (monty.Result, error) {
	if len(call.Args) != 1 || len(call.Kwargs) != 0 {
		return pythonError("TypeError", "docs(topic) expects one topic string; use docs('index')"), nil
	}
	topic, ok := call.Args[0].Raw().(string)
	if !ok {
		return pythonError("TypeError", "docs(topic) expects a string"), nil
	}
	if topic == "" || strings.ContainsAny(topic, "/\\.") {
		return pythonError("ValueError", "unknown documentation topic; use docs('index')"), nil
	}
	data, err := documentation.ReadFile("docs/" + topic + ".md")
	if err != nil {
		return pythonError("ValueError", "unknown documentation topic; use docs('index')"), nil
	}
	return monty.Return(monty.MustValueOf(string(data))), nil
}
