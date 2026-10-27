# Image viewing

view(path) reads a PNG or JPEG image immediately, captures its bytes, and queues
it for the next model request. Relative paths resolve against the project cwd.
Limits: four images per Python execution, 2 MiB and 20 megapixels per image.
view is unavailable during prefix initialization. Unsupported/invalid files raise
an exception. A successful call returns a short acknowledgement, not image data.

All tool calls in the completed response still run sequentially. After their
outputs are committed, queued images are appended as actual image content in a
separate user notification before the next request. This does not execute a
request early or overlap requests. Images captured before an ordinary Python
exception remain queued. Cancellation that kills the worker before its result
arrives cannot preserve those pending images.

Captured image data is stored directly in API history; restoration does not
require the original file. Images can make session files larger and may contain
sensitive data. Provider image support is required. Both provider API types accept image input
through the wire adapter. docs("clipboard") describes paste attachments;
images are shown as labels only, never rendered as terminal graphics.
