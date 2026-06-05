// Command spike-validate runs phpsyms.Extract against a directory of .php
// files and reports symbol counts, elapsed time, and lexer panic incidence.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thanhhaudev/phpsyms"
)

func main() {
	dir := flag.String("dir", "", "directory of .php files")
	flag.Parse()
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "usage: spike-validate -dir <path>")
		os.Exit(2)
	}

	var files []string
	err := filepath.Walk(*dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(p) == ".php" {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var totalSyms, totalClass, totalMethod, totalFuncCall int
	var panics []string
	start := time.Now()
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", f, err)
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					panics = append(panics, fmt.Sprintf("%s: %v", f, r))
				}
			}()
			syms, _ := phpsyms.Extract(f, src)
			totalSyms += len(syms)
			for _, s := range syms {
				switch s.Kind {
				case phpsyms.KindClass:
					totalClass++
				case phpsyms.KindMethod:
					totalMethod++
				case phpsyms.KindFunctionCall:
					totalFuncCall++
				}
			}
		}()
	}
	elapsed := time.Since(start)

	fmt.Printf("files:         %d\n", len(files))
	fmt.Printf("symbols:       %d (class=%d method=%d funcCall=%d)\n", totalSyms, totalClass, totalMethod, totalFuncCall)
	fmt.Printf("elapsed:       %s\n", elapsed)
	if len(files) > 0 {
		fmt.Printf("per-file avg:  %s\n", elapsed/time.Duration(len(files)))
		fmt.Printf("files/sec:     %.0f\n", float64(len(files))/elapsed.Seconds())
	}
	fmt.Printf("panics:        %d\n", len(panics))
	for _, p := range panics {
		fmt.Printf("  - %s\n", p)
	}
	if len(panics) > 0 {
		os.Exit(1)
	}
}
