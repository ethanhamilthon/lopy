package repl

import (
	"context"
	"strings"
	"testing"
)

func TestDocsThroughWorker(t *testing.T) {
	r := &REPL{Dir: t.TempDir(), Prefix: "print(docs('index'))"}
	defer r.Close()
	if _, err := r.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.PrefixOutput, "Lopy documentation") {
		t.Fatal(r.PrefixOutput)
	}
	for _, topic := range []string{"index", "runtime", "sessions", "steering", "interface", "files", "images", "models", "headless", "clipboard"} {
		got, err := r.Execute(context.Background(), "text = docs('"+topic+"')\nprint(text)")
		if err != nil || got.Error || len(got.Output) < 50 {
			t.Fatalf("%s: %+v %v", topic, got, err)
		}
	}
	for _, code := range []string{"docs()", "docs(12)", "docs('missing')", "docs('../system')", "docs('index', 'runtime')"} {
		got, err := r.Execute(context.Background(), code)
		if err != nil || !got.Error {
			t.Fatalf("%s: %+v %v", code, got, err)
		}
	}
	got, err := r.Execute(context.Background(), "len(text) > 50")
	if err != nil || got.Error || !strings.Contains(got.Output, "True") {
		t.Fatalf("state lost: %+v %v", got, err)
	}
}
