package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thanhhaudev/phpsyms"
)

func loadCorpus(b *testing.B) [][]byte {
	b.Helper()
	var files [][]byte
	root := "../testdata/laravel-framework"
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(p) != ".php" {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		files = append(files, src)
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	if len(files) == 0 {
		b.Skip("no corpus")
	}
	return files
}

func BenchmarkExtract(b *testing.B) {
	files := loadCorpus(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, src := range files {
			_, _ = phpsyms.Extract("f.php", src)
		}
	}
	// Compute the per-second metric AFTER the loop using the b.Elapsed() helper.
	b.ReportMetric(float64(len(files))*float64(b.N)/b.Elapsed().Seconds(), "files/sec")
}
