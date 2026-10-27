package repl

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestViewCapturesImageBytesAndResetsQueue(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "image.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(file, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	r := &REPL{Dir: dir}
	defer r.Close()
	if _, err = r.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := r.Execute(context.Background(), "view('image.png')\nraise ValueError('after capture')")
	if err != nil || !result.Error || len(result.Images) != 1 || !strings.HasPrefix(result.Images[0].URL, "data:image/png;base64,") {
		t.Fatalf("image lost after ordinary error: %+v %v", result, err)
	}
	result, err = r.Execute(context.Background(), "42")
	if err != nil || result.Error || len(result.Images) != 0 {
		t.Fatalf("queue not reset: %+v %v", result, err)
	}
	result, err = r.Execute(context.Background(), "for i in range(5):\n    view('image.png')")
	if err != nil || !result.Error || len(result.Images) != 4 || !strings.Contains(result.Output, "four images") {
		t.Fatalf("image limit: %+v %v", result, err)
	}
	for _, code := range []string{"view('missing')", "view(42)", "view()"} {
		result, err = r.Execute(context.Background(), code)
		if err != nil || !result.Error || len(result.Images) != 0 {
			t.Fatalf("%s: %+v %v", code, result, err)
		}
	}
	r.Close()
	r.Prefix = "view('image.png')"
	if _, err = r.Ensure(context.Background()); err == nil || !strings.Contains(err.Error(), "prefix initialization") {
		t.Fatalf("unexpected prefix result: %v", err)
	}
}

func TestViewRejectsInvalidAndOversizedImages(t *testing.T) {
	dir := t.TempDir()
	for _, contents := range []string{"not an image", strings.Repeat("x", 2*1024*1024+1)} {
		path := filepath.Join(dir, "bad.png")
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := CaptureImage(path); err == nil {
			t.Fatal("accepted invalid/oversized image")
		}
	}
	if _, err := CaptureImage(dir); err == nil {
		t.Fatal("accepted directory")
	}
}
