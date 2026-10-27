package repl

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMontyPrototype(t *testing.T) {
	snippets := []struct {
		code, want string
		failed     bool
	}{
		{"x = 40\nprint('hello')\nx + 2", "hello\n42\n", false},
		{"x += 1\nx", "41\n", false},
		{"x = 41; print('hello'); x + 1; # end", "hello\n42\n", false},
		{"(\n x +\n 1\n)", "42\n", false},
		{"def add(n):\n    return n + 1\n\nadd(x)", "42\n", false},
		{"print('before')\n1/0", "ZeroDivisionError", true},
		{"x", "41\n", false},
		{"print('a', 'b', sep='-', end='!')", "a-b!", false},
		{"sh('printf shell-output')['stdout']", "'shell-output'\n", false},
		{"[True, None, {'x': (1,)}]", "[True, None, {'x': (1)}]\n", false},
		{"f'value={x}'", "'value=41'\n", false},
		{"x = 42\n1/0", "ZeroDivisionError", true},
		{"x", "42\n", false},
		{"print('must not run')\nx = (", "SyntaxError", true},
		{"x", "42\n", false},
	}
	var input, output bytes.Buffer
	enc := json.NewEncoder(&input)
	for _, s := range snippets {
		if err := enc.Encode(map[string]any{"code": s.code}); err != nil {
			t.Fatal(err)
		}
	}
	if err := runMontyWorker(&input, &output); err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(&output)
	for _, s := range snippets {
		var result Result
		if err := dec.Decode(&result); err != nil {
			t.Fatal(err)
		}
		if result.Error != s.failed || !strings.Contains(result.Output, s.want) {
			t.Errorf("%s: got %+v, want %q error=%v", s.code, result, s.want, s.failed)
		}
		if strings.Contains(result.Output, "must not run\n") {
			t.Error("syntax error executed side effect")
		}
	}
}

func TestMontyFinalExpressionBoundaries(t *testing.T) {
	cases := []struct{ code, want string }{
		{"x=1;\nx+1", "2\n"},
		{"x=2; x+1; # comment", "3\n"},
		{"x = 3\n# trailing comment\nx+1\n# last comment", "4\n"},
		{"x = ';'\nx", "';'\n"},
		{"s = '''multi\nline; # not comment'''\ns", "'multi\\nline; # not comment'\n"},
		{"x = [1,\n2]\nx", "[1, 2]\n"},
		{"if True:\n    x=7\n    x", ""},
		{"if True: x=8; print('eight')", "eight\n"},
		{"def f():\n    return 9", ""},
		{"f()", "9\n"},
		{"_ + 1", "10\n"},
		{"x = 4\\\n + 5\nx", "9\n"},
	}
	var input, output bytes.Buffer
	enc := json.NewEncoder(&input)
	for _, tc := range cases {
		if err := enc.Encode(map[string]any{"code": tc.code}); err != nil {
			t.Fatal(err)
		}
	}
	if err := runMontyWorker(&input, &output); err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(&output)
	for _, tc := range cases {
		var result Result
		if err := dec.Decode(&result); err != nil {
			t.Fatal(err)
		}
		if result.Error || result.Output != tc.want {
			t.Errorf("%q => %+v want %q", tc.code, result, tc.want)
		}
	}
}

func TestMontySessionHasNoLifetimeDeadline(t *testing.T) {
	limits := montyLimits()
	if limits.MaxDuration != 0 {
		t.Fatalf("persistent session must not have a lifetime deadline: %s", limits.MaxDuration)
	}
	if limits.MaxMemory != 128*1024*1024 || limits.MaxAllocations != 10_000_000 || limits.MaxRecursionDepth != 1000 {
		t.Fatal("persistent resource limits were removed")
	}
}
