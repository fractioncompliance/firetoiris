package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIInspectConvertAndUnknownCommand(t *testing.T) {
	path := "../../examples/nec.fire.txt"
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"inspect", "--input", path, "--tax-year", "2026"}, &stdout, &stderr); code != 0 {
		t.Fatalf("%d %s %s", code, &stdout, &stderr)
	}
	if strings.Contains(stdout.String(), "000654321") || strings.Contains(stdout.String(), "SYNTHETIC RECIPIENT") {
		t.Fatal("inspect disclosed taxpayer identity")
	}
	output := filepath.Join(t.TempDir(), "converted.json")
	stdout.Reset()
	stderr.Reset()
	if code := run(context.Background(), []string{"convert", "--input", path, "--tax-year", "2026", "--output", output}, &stdout, &stderr); code != 0 {
		t.Fatalf("%s %s", &stdout, &stderr)
	}
	info, err := os.Stat(output)
	if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0600) {
		t.Fatal("output is not private")
	}
	if code := run(context.Background(), []string{"convert", "--input", path, "--tax-year", "2026", "--output", output}, &stdout, &stderr); code == 0 {
		t.Fatal("overwrote existing output")
	}
	if code := run(context.Background(), []string{"typo"}, &stdout, &stderr); code == 0 {
		t.Fatal("unknown command accepted")
	}
}
