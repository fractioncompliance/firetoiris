package firetoiris

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFictionalInputEnvelopes(t *testing.T) {
	files, err := filepath.Glob("testdata/inputs/*/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 83 {
		t.Fatalf("fixture count %d", len(files))
	}
	for _, path := range files {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var input Input
			if err = json.Unmarshal(raw, &input); err != nil {
				t.Fatal(err)
			}
			if err = CheckEnvelope(input); err != nil {
				t.Fatal(err)
			}
		})
	}
}
