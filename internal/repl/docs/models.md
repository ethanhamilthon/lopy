# Models, effort, and providers

/model fetches the model lists of all configured providers when opened and shows each model with its provider. Choosing a model also makes its provider the default one. A provider that fails to answer is reported and skipped. Up/Down selects
an entry, typing filters it, Left/Right cycles none, low, medium, high, xhigh,
max, and Enter saves the selection. Esc cancels; late fetch results do not reopen
a cancelled menu. Settings changes are disabled while a turn is running.

/login lists configured providers and ends with + Add. Add asks for provider
name, HTTP(S) base URL, API type (responses or chat_completions), and an optional
API key. The key is masked, kept in private settings.toml, and never appended to
chat history. An omitted API type in older configuration means Responses.
Provider selection changes the active session's provider and new-session default;
use /model afterwards to choose a model supported by that provider.

Each session header stores provider identity, model, and effort, but no API key.
Changing defaults updates the current session and new sessions; older sessions
keep their own selection when resumed. A legacy header without selection uses
current defaults and pins them when next saved. Missing provider configuration
raises an error instead of silently switching providers. Provider credentials and
endpoints are resolved from current configuration by provider identity. API
errors for unsupported models/efforts are reported without an automatic fallback.

Responses and Chat Completions both support streaming text, tool argument
previews, sequential tool execution, image input, refusals, and available usage.
History uses one canonical representation; the Chat adapter translates it on the
wire. Unsupported nonstandard provider extensions are not inferred.

Python session is an immutable named-tuple snapshot refreshed before every
execution, including the prefix. Fields: id, model, provider, effort, cwd,
ctx_size. ctx_size is the latest recorded request input-token count, including
cached tokens, matching the status-bar estimate; it is None when unknown, not the
model context limit. Session credentials are not exposed. Interpreter restarts
and session changes recreate the snapshot without restoring old Python variables.
