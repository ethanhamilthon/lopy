package repl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileHelpersThroughWorker(t *testing.T) {
	dir := t.TempDir()
	r := &REPL{Dir: dir}
	defer r.Close()
	if _, err := r.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		code, want string
		failed     bool
	}{
		{`write('one.txt', 'alpha\nbeta\n')`, "", false},
		{`read('one.txt')`, `1: alpha\n2: beta\n`, false},
		{`read('one.txt', offset=1, limit=1)`, `1: alpha\n[lines 1-1 of 2]\n`, false},
		{`read('one.txt', offset=2, limit=1)`, `2: beta\n`, false},
		{`read('one.txt', offset=-1)`, `2: beta\n`, false},
		{`read('one.txt', offset=99)`, `[EOF: 2 lines]\n`, false},
		{`write('nested/dir/two.txt', 'gamma delta')`, "", false},
		{`patch([{'path':'one.txt','old':'alpha','new':'ALPHA'}, {'path':'one.txt','old':'beta','new':'BETA'}, {'path':'nested/dir/two.txt','old':'gamma','new':'GAMMA'}])`, "2\n", false},
		{`read('one.txt')`, `1: ALPHA\n2: BETA\n`, false},
		{`read('nested/dir/two.txt')`, `1: GAMMA delta`, false},
		{`patch([{'path':'one.txt','old':'ALPHA','new':'oops'}, {'path':'nested/dir/two.txt','old':'missing','new':'x'}])`, "matched 0 times", true},
		{`read('one.txt')`, `1: ALPHA\n2: BETA\n`, false},
		{`patch([{'path':'one.txt','old':'','new':'x'}])`, "nonempty old", true},
		{`patch([{'path':'one.txt','old':'A','new':'x'}])`, "matched 3 times", true},
		{`write('one.txt', 12)`, "expects two strings", true},
		{`write('one.txt/file', 'text')`, "not a directory", true},
		{`read('one.txt', offset=0)`, "non-zero offset", true},
		{`read('one.txt', 1, offset=2)`, "duplicate argument", true},
		{`read('one.txt', typo=1)`, "unknown argument", true},
		{`read('one.txt', limit=2001)`, "limit 1..2000", true},
		{`grep('A')`, `unable to find "grep"`, true},
		{`x = 42`, "", false},
		{`read('missing')`, "no such file", true},
		{`x`, "42\n", false},
	}
	for _, tc := range cases {
		got, err := r.Execute(context.Background(), tc.code)
		if err != nil {
			t.Fatalf("%s: %v", tc.code, err)
		}
		if got.Error != tc.failed || !strings.Contains(got.Output, tc.want) {
			t.Errorf("%s: got %+v; want %q error=%v", tc.code, got, tc.want, tc.failed)
		}
	}
	info, err := os.Stat(filepath.Join(dir, "one.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("new file permissions: %v", info.Mode())
	}
}

func TestFileHelpersLimitsAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("old old"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := atomicText(path, "new"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("permissions: %v %v", info, err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := atomicText(link, "bad"); err == nil {
		t.Fatal("overwrote symlink")
	}
	for i, content := range []string{"\x00binary", "\xff", strings.Repeat("x", fileLimit+1)} {
		target := filepath.Join(dir, fmt.Sprint(i))
		if err := os.WriteFile(target, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := textFile(target); err == nil {
			t.Fatal("accepted binary/oversized file")
		}
	}
	lines, err := readText([]any{path, int64(1), int64(10)})
	if err != nil || !strings.Contains(fmt.Sprint(lines), "1: new") {
		t.Fatalf("read after write: %v %v", lines, err)
	}
	if _, err := readText([]any{path, int64(20), int64(1)}); err != nil {
		t.Fatal(err)
	}
}

func TestPatchRollsBackEarlierFilesWhenCommitFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires ordinary filesystem permission enforcement")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0700); err != nil {
		t.Fatal(err)
	}
	first, second := filepath.Join(dir, "first"), filepath.Join(locked, "second")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(locked, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0700)
	r := &REPL{Dir: dir}
	defer r.Close()
	if _, err := r.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := r.Execute(context.Background(), `patch([{'path':'first','old':'original','new':'changed'}, {'path':'locked/second','old':'original','new':'changed'}])`)
	if err != nil || !result.Error {
		t.Fatalf("expected commit failure: %+v %v", result, err)
	}
	for _, path := range []string{first, second} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "original" {
			t.Fatalf("partial edit left behind in %s: %q %v", path, data, err)
		}
	}
}
