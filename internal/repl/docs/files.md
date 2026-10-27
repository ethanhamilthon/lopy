# File helpers

Paths are relative to the project working directory; absolute paths are also
accepted. Functions run with user permissions, not in an OS sandbox. Files must
be UTF-8 and at most 4 MiB. Host failures raise Python exceptions and do not clear
ordinary interpreter state. Use sh for binary files or other specialized tasks.

read(path, offset=1, limit=200) returns numbered lines starting at a 1-based
offset, or from the end of the file when offset is negative (for example -50 for
the last 50 lines). Limit is 1..2000. Lines over 4,000 bytes are clipped with a
marker, partial reads append [lines X-Y of N], and oversized outputs stop cleanly
with [truncated at line X of Y].
write(path, content) creates parent directories if needed and creates or replaces
a file using a same-directory temporary file and atomic rename. It returns None,
preserves existing permission bits, and uses 0600 for new files. It refuses to
replace symlinks or non-regular files.

patch(edits) accepts at most 200 dictionaries with string fields path, old, new:
patch([{'path': 'a.go', 'old': 'before', 'new': 'after'},
       {'path': 'b.go', 'old': 'other', 'new': 'replacement'}])
Each nonempty old string must occur exactly once at that point in the edit
sequence; mismatch errors report the actual match count. Multiple edits can
target one file. All matching and size validation occurs before writing. Total
original input is limited to 16 MiB. Returns the number of changed files. On a
write failure, previously written files are rolled back; rollback failures are
reported too. Each file replacement is atomic, but multiple files are NOT a
crash-atomic filesystem transaction. Hard cancellation can interrupt the commit.
Do not use this to coordinate concurrent writers.
