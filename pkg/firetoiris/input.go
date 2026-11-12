package firetoiris

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Bundle struct {
	Version      string  `json:"version"`
	SourceSHA256 string  `json:"source_sha256"`
	Inputs       []Input `json:"inputs"`
}

func ReadInputs(r io.Reader) (Bundle, string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxInputBytes+1))
	if err != nil || len(raw) > maxInputBytes {
		return Bundle{}, "", fmt.Errorf("input_unreadable_or_exceeds_128_MiB")
	}
	if err = uniqueJSON(raw); err != nil {
		return Bundle{}, "", err
	}
	var keys map[string]json.RawMessage
	if err = json.Unmarshal(raw, &keys); err != nil {
		return Bundle{}, "", fmt.Errorf("form_input_requires_JSON_object")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	b := Bundle{Version: Version}
	if _, ok := keys["inputs"]; ok {
		err = d.Decode(&b)
	} else {
		var input Input
		err = d.Decode(&input)
		b.Inputs = []Input{input}
	}
	if err != nil || d.Decode(&struct{}{}) != io.EOF || b.Version != Version || len(b.Inputs) == 0 {
		return Bundle{}, "", fmt.Errorf("invalid_form_input_bundle")
	}
	for i, input := range b.Inputs {
		if err = CheckEnvelope(input); err != nil {
			return Bundle{}, "", fmt.Errorf("batch %d: %w", i+1, err)
		}
	}
	return b, hashBytes(raw), nil
}

func uniqueJSON(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var visit func(int) error
	visit = func(depth int) error {
		if depth > 48 {
			return fmt.Errorf("JSON_nesting_limit")
		}
		token, err := d.Token()
		if err != nil {
			return fmt.Errorf("invalid_JSON")
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return fmt.Errorf("invalid_JSON")
				}
				s, ok := key.(string)
				if !ok || seen[s] {
					return fmt.Errorf("duplicate_JSON_key")
				}
				seen[s] = true
				if err = visit(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err = visit(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid_JSON")
		}
		if _, err = d.Token(); err != nil {
			return fmt.Errorf("invalid_JSON")
		}
		return nil
	}
	if err := visit(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing_JSON")
	}
	return nil
}

func WriteBundle(path string, b Bundle) error {
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("bundle_encoding_failed")
	}
	f, err := openPrivate(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY)
	if err != nil {
		return fmt.Errorf("output_must_be_a_new_private_file")
	}
	if _, err = f.Write(raw); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("output_write_failed")
	}
	return nil
}
