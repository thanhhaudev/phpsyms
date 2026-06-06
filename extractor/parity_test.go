package extractor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thanhhaudev/phpsyms"
)

// TestParity_LaravelFramework counts top-level declarations across the committed
// laravel-framework corpus. Pins a baseline so changes in the extractor don't
// silently drop coverage. The full tree-sitter parity comparison lives in
// llmreviewkit (Task 19 in the parent plan); this test is the local floor.
func TestParity_LaravelFramework(t *testing.T) {
	root := "../testdata/laravel-framework"
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("laravel-framework corpus not present")
	}
	var classCount, methodCount, useCount, callCount int
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(p) != ".php" {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		syms, _ := phpsyms.Extract(p, src)
		for _, s := range syms {
			switch s.Kind {
			case phpsyms.KindClass, phpsyms.KindInterface, phpsyms.KindTrait:
				classCount++
			case phpsyms.KindMethod:
				methodCount++
			case phpsyms.KindUseImport:
				useCount++
			case phpsyms.KindStaticCall, phpsyms.KindMethodCall, phpsyms.KindFunctionCall:
				callCount++
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("laravel-framework: classes=%d methods=%d uses=%d calls=%d", classCount, methodCount, useCount, callCount)

	// Baselines pinned during initial corpus collection. The numbers reflect
	// what the v0.1.0 extractor sees on the ~51-file laravel-framework slice.
	// Adjust DOWN only if a future change intentionally reduces coverage.
	// Adjust UP if you add to the corpus.
	// Floors set at ~85% of measured (classes=139, methods=843, uses=394, calls=2911).
	if classCount < 118 {
		t.Errorf("class+interface+trait coverage regression: got %d, want ≥118", classCount)
	}
	if methodCount < 716 {
		t.Errorf("method coverage regression: got %d, want ≥716", methodCount)
	}
	if useCount < 334 {
		t.Errorf("use coverage regression: got %d, want ≥334", useCount)
	}
	if callCount < 2474 {
		t.Errorf("call coverage regression: got %d, want ≥2474", callCount)
	}
}
