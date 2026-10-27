# TODO

## 1. Python tool description and system prompt

Status: implemented and verified.

- [x] Set the default editable system prompt to `You are Lopy, a coding agent`.
- [x] Keep the final system instructions composed from the system prompt plus prefix source. Move only the Python environment/usage description into the Python tool-call description; do not move prefix source out of system instructions.
- [x] Describe the actual interpreter, execution limits, persistent state, state loss, available functions, unsupported features/libraries, and host access in the tool description.
- [x] Briefly explain prefix execution and the separate initialization/restart notification containing its result. Treat that result as execution data, not instructions.
- [x] Leave existing editable system prompts untouched; the user will update theirs manually.
- [x] Test assembled requests, defaults, custom prompts, and prefix notifications.

## 2. Python helper functions and image viewing

- [x] Add Python-accessible `read`, `write`, `patch`, `grep`, `view`, and `docs(topic)`; retain `sh`. HTTP/request helpers are out of scope.
- [x] Define compact, consistent signatures, results, errors, path resolution, and output limits for file helpers.
- [x] Support multiple file edits and multiple replacements within a file in a single `patch` call. Validate the edit set before writing; define failure/rollback behavior and avoid silently leaving partial edits.
- [x] Implement `view(path)` for images: capture the image, queue it through a steering-like mechanism, and include actual image content in the next model request, not merely a path or text description.
- [x] Preserve call/output pairing and sequential execution. Deliver queued images at the next safe request boundary, without starting an overlapping model request.
- [x] Define image format/size limits and errors. Preserve attachment content or a durable reference across history restoration; do not depend on the original path remaining unchanged.
- [x] Implement `docs(topic)` so the agent can discover how Lopy works. Define supported topics, topic discovery, unknown-topic errors, and bounded output; keep documentation consistent with implemented behavior.
- [x] Document helper usage in the tool description and test multi-edit patches, failures, limits, cancellation, image payloads, queue ordering, and persistence.

## 3. Model, effort, and provider selection

- [x] Add `/model`; fetch models from the selected provider when invoked.
- [x] Select the model in that UI and change its effort with Left/Right arrows. Do not add a separate effort command.
- [x] Use the hardcoded effort choices `none`, `low`, `medium`, `high`, `xhigh`, `max`.
- [x] Persist selected model/effort as configuration defaults and update the current session. New sessions use these defaults; older sessions retain their own settings.
- [x] Store provider identity, model, and effort in every session and restore them on resume. Keep credentials outside session files.
- [x] Define compatible initialization for old sessions lacking these fields and handle unavailable providers/models and unsupported effort explicitly.
- [x] Coordinate changes with active requests and test current/new/old sessions, persistence, restoration, and failure handling.

### Provider login and creation

- [x] Add `/login` showing configured providers. End the list with a plus icon and an English `Add` label.
- [x] Implement a step-by-step flow for creating a provider, including its identity, endpoint, API type, and credentials as required.
- [x] Support both Responses and Chat Completions APIs, including streaming, tool calls/results, images, usage, and error propagation.
- [x] Define how selecting a provider updates the current session and new-session defaults without modifying older sessions.
- [x] Preserve existing provider configuration compatibility and private credential storage; do not expose credentials in logs, history, or UI previews.
- [x] Test both API adapters, model fetching, creation/selection, cancelled setup, and invalid configuration.

## 4. Global session object

- [x] Expose a global Python `session` object with `id`, `model`, `provider`, `effort`, `cwd`, and `ctx_size`.
- [x] Keep it synchronized with the active session, model/provider/effort changes, and interpreter initialization/restarts.
- [x] Set `ctx_size` to the same current-context estimate shown in the status bar: the latest reported request input-token count, including cached input, not model context capacity or cumulative usage. Reuse its source and unknown-value behavior; define field access/mutability.
- [x] Keep credentials out of the object and isolate metadata between sessions.
- [x] Document and test availability, updates, restored sessions, and unknown metadata.

## 5. Headless modes and external notifications

- [x] Extend the existing `-prompt` support with stdio and JSONL modes and explicit session resumption. Without an explicit resume, create a new session.
- [x] Define CLI flags, stdin lifecycle, text output, JSONL input/events, errors, exit statuses, and default session behavior before implementation.
- [x] Keep machine-readable stdout free of TUI control sequences and diagnostic noise.
- [x] Add `lopy notify -s ID` to send a message to a session. Define message input syntax via arguments and/or stdin.
- [x] Wrap the delivered text in a `notify` XML block and inject it through a steering-like queue at a safe model-request boundary. Define escaping so message content cannot accidentally break the wrapper.
- [x] Implement cross-process delivery to the owning client without a second client writing session history directly.
- [x] Return an error for an inactive session. Acknowledge enqueueing in a bounded in-memory owner queue, not model acceptance or durable storage. Document possible loss on exit/switching and ambiguous acknowledgements; do not retry automatically.
- [x] Test stdio/JSONL workflows, resume, notifications, cancellation, malformed input, and full diagnostics.

## 6. Session ownership and read-only followers

- [x] Allow only one writer per session using atomic cross-process ownership for both TUI and headless clients.
- [x] Open a second client on an owned session in read-only mode rather than rejecting it.
- [x] Poll the session once per second and refresh the follower's view with available updates, without mutating history or starting an interpreter/model request.
- [x] Show read-only status clearly and disable mutating actions. Preserve follower scroll position while refreshing.
- [x] Show updates only as they appear in the saved session JSONL. Live streaming and a separate progress-publication channel are not required.
- [x] Release ownership on switching/exit and recover safely after crashes. Define whether a follower can explicitly acquire ownership after the writer exits; do not silently create two writers.
- [x] Keep independent sessions concurrent and coordinate external notifications with the session owner.
- [x] Test real process contention, polling, partial/corrupt reads, switching, crash recovery, and persistence safety.

## 7. Clipboard image attachments

Source budget increased to 5,000 lines with user approval. Implemented with text-only attachment labels; native desktop verification remains pending.

- [x] Support image paste into the composer with Cmd+V / Ctrl+V where supported by the terminal/platform, including Russian keyboard layout equivalents.
- [x] Preserve ordinary text paste and editing; document terminal shortcut interception and any necessary fallback.
- [x] Save pasted images in a configurable attachment directory, defaulting to the platform temporary directory (for example `/var/...` on macOS), not a hardcoded user-specific path.
- [x] Show attachments in the input, allow removal, and submit actual image content with the user's message.
- [x] Define draft, queued steering, cancellation, session switching, permissions, retention, and cleanup behavior. Keep saved history usable after temporary files disappear.
- [x] Reuse image encoding/validation and persistence support from `view` where possible.
- [x] Test clipboard images, text fallback, both layouts, unsupported terminals, attachment lifecycle, and both provider API formats.

- [ ] Verify real OS clipboard access, terminal Cmd+V forwarding, and Russian layout shortcuts interactively. Automated fixtures do not read the user clipboard.

## 8. Fix composer flicker at maximum height

- [x] Reproduce flicker while typing into a long input after the composer reaches its maximum height.
- [x] Fix layout/viewport/render updates without breaking cursor visibility, multiline editing, wrapping, or timeline scrolling.
- [x] Add regression coverage for continued typing, deletion, long wrapped text, shrinking, and narrow terminal sizes.
- [ ] Verify the flicker fix interactively in the user terminal.

## 9. Slash-prefixed messages and command completion

- [x] Enter immediately executes the selected visible completion, without a second confirmation.
- [x] Slash-prefixed text without a matching visible completion is sent as an ordinary message/steering instruction, not rejected as an unknown command.
- [x] Cover command prefixes, selected commands, dismissed completions, paths, multiline text, and slash-prefixed steering.

## Constraints and verification

- [ ] Keep application text, documentation, and code comments in English.
- [ ] Keep production Go source within the approved 5,000 physical-line budget in `AGENTS.md`. Baseline after TUI extraction: 2,580 lines. Do not compress code or move application logic into other file types to evade the limit.
- [ ] Preserve interpreter cancellation, protocol/stdout isolation, output limits, atomic persistence, FIFO steering, and initialization notifications.
- [ ] Run `go test ./...`, `go test -race ./...`, `go vet ./...`, and a build. Do not recreate a project-root binary unless requested.
- [ ] Update README and verification notes as features are implemented.

