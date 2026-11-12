package firetoiris

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Item struct {
	InputSHA256    string `json:"input_sha256"`
	PrepareKey     string `json:"prepare_key"`
	SubmitKey      string `json:"submit_key"`
	FilingID       string `json:"filing_id,omitempty"`
	Tenant         string `json:"tenant,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	TransmissionID string `json:"transmission_id,omitempty"`
	Valid          bool   `json:"schema_validated"`
}

type State struct {
	Version                 string `json:"version"`
	SourceSHA256            string `json:"source_sha256"`
	InputSHA256             string `json:"input_sha256"`
	BaseURL                 string `json:"api_origin"`
	Mode                    string `json:"mode"`
	PrepareCredentialSHA256 string `json:"prepare_credential_sha256"`
	Items                   []Item `json:"items"`
}

type Journal struct {
	State State
	path  string
	lock  *os.File
}

func OpenJournal(path string) (*Journal, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("journal_path_invalid")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return nil, fmt.Errorf("journal_parent_must_exist")
	}
	absolute = filepath.Join(parent, filepath.Base(absolute))
	lock, err := openPrivate(absolute+".lock", os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, fmt.Errorf("journal_lock_unavailable")
	}
	if err = lockJournal(lock); err != nil {
		lock.Close()
		return nil, fmt.Errorf("journal_in_use")
	}
	j := &Journal{path: absolute, lock: lock}
	f, err := openPrivate(absolute, os.O_RDONLY)
	if os.IsNotExist(err) {
		return j, nil
	}
	if err != nil {
		j.Close()
		return nil, fmt.Errorf("journal_unreadable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || checkPrivate(f) != nil || info.Size() > 4<<20 {
		j.Close()
		return nil, fmt.Errorf("journal_requires_private_regular_file_under_4_MiB")
	}
	d := json.NewDecoder(io.LimitReader(f, 4<<20))
	d.DisallowUnknownFields()
	err = d.Decode(&j.State)
	if err != nil || j.State.Version != Version || d.Decode(&struct{}{}) != io.EOF {
		j.Close()
		return nil, fmt.Errorf("journal_corrupt_or_incompatible")
	}
	return j, nil
}
func (j *Journal) Close() error {
	if j.lock == nil {
		return nil
	}
	_ = unlockJournal(j.lock)
	err := j.lock.Close()
	j.lock = nil
	return err
}
func (j *Journal) save() error {
	if j.lock == nil {
		return fmt.Errorf("journal_closed")
	}
	data, err := json.MarshalIndent(j.State, "", "  ")
	if err != nil {
		return fmt.Errorf("journal_encoding_failed")
	}
	f, err := createPrivateTemp(filepath.Dir(j.path))
	if err != nil {
		return fmt.Errorf("journal_write_failed")
	}
	name := f.Name()
	defer os.Remove(name)
	if err = checkPrivate(f); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = replacePrivate(name, j.path)
	}
	if err != nil {
		return fmt.Errorf("journal_write_failed_stop_and_preserve_journal")
	}
	if err = syncParent(filepath.Dir(j.path)); err != nil {
		return fmt.Errorf("journal_directory_sync_failed")
	}
	return nil
}
func hashBytes(raw []byte) string { d := sha256.Sum256(raw); return hex.EncodeToString(d[:]) }
func inputHash(input any) string  { raw, _ := json.Marshal(input); return hashBytes(raw) }

func (j *Journal) Prepare(ctx context.Context, c *Client, inputs []Input, mode string, sourceHashes ...string) error {
	source := inputHash(inputs)
	if len(sourceHashes) > 1 {
		return fmt.Errorf("one_source_hash_required")
	}
	if len(sourceHashes) == 1 {
		source = sourceHashes[0]
	}
	if !safeHash.MatchString(source) {
		return fmt.Errorf("invalid_source_hash")
	}
	if len(inputs) == 0 {
		return fmt.Errorf("empty_batch")
	}
	for i, input := range inputs {
		if err := CheckEnvelope(input); err != nil {
			return fmt.Errorf("batch %d: %w", i+1, err)
		}
	}
	digest := inputHash(inputs)
	credential := hashBytes([]byte(c.key))
	if j.State.Version == "" {
		j.State = State{Version: Version, SourceSHA256: source, InputSHA256: digest, BaseURL: c.base, Mode: mode, PrepareCredentialSHA256: credential, Items: []Item{}}
		for i, input := range inputs {
			h := inputHash(input)
			key := hashBytes([]byte(fmt.Sprintf("FIREtoIRIS:%s:%s:%d", Version, digest, i)))
			j.State.Items = append(j.State.Items, Item{InputSHA256: h, PrepareKey: "f2i-prepare-" + key, SubmitKey: "f2i-submit-" + key})
		}
		if err := j.save(); err != nil {
			return err
		}
	} else if j.State.SourceSHA256 != source || j.State.InputSHA256 != digest || j.State.BaseURL != c.base || j.State.Mode != mode || j.State.PrepareCredentialSHA256 != credential || len(j.State.Items) != len(inputs) {
		return fmt.Errorf("journal_input_destination_mode_or_preparation_key_changed")
	}
	if err := c.checkMode(ctx, mode); err != nil {
		return err
	}
	for i, input := range inputs {
		item := &j.State.Items[i]
		if item.InputSHA256 != inputHash(input) {
			return fmt.Errorf("journal_input_hash_mismatch")
		}
		if item.FilingID != "" {
			if !item.Valid {
				return fmt.Errorf("batch %d: server_validation_failed_review_saved_filing", i+1)
			}
			continue
		}
		var resource Resource
		if err := c.call(ctx, "POST", "/v1/filings/form-input", item.PrepareKey, input, &resource); err != nil {
			return fmt.Errorf("batch %d: %w", i+1, err)
		}
		if !validResource(resource, mode) {
			return fmt.Errorf("invalid_server_filing_response_preserve_journal")
		}
		item.FilingID = resource.ID
		item.Tenant = resource.Tenant
		item.SHA256 = resource.SHA256
		item.TransmissionID = resource.TransmissionID
		item.Valid = resource.Validation != nil && resource.Validation.Valid && resource.Validation.SchemaValidated
		if err := j.save(); err != nil {
			return err
		}
		if !item.Valid {
			return fmt.Errorf("batch %d: server_validation_failed_review_saved_filing", i+1)
		}
	}
	return nil
}

func (j *Journal) check(ctx context.Context, c *Client) error {
	if j.lock == nil || j.State.Version != Version || j.State.BaseURL != c.base || len(j.State.Items) == 0 {
		return fmt.Errorf("journal_missing_or_destination_changed")
	}
	return c.checkMode(ctx, j.State.Mode)
}
func (j *Journal) current(ctx context.Context, c *Client, item Item) (Resource, error) {
	if !safeID.MatchString(item.FilingID) || !safeHash.MatchString(item.SHA256) {
		return Resource{}, fmt.Errorf("journal_filing_invalid")
	}
	var r Resource
	err := c.call(ctx, "GET", "/v1/filings/"+item.FilingID, "", nil, &r)
	if err != nil {
		return r, err
	}
	if !validResource(r, j.State.Mode) || r.ID != item.FilingID || r.Tenant != item.Tenant || r.SHA256 != item.SHA256 {
		return r, fmt.Errorf("saved_filing_changed_or_tenant_mismatch_requires_review")
	}
	return r, nil
}
func (j *Journal) Approve(ctx context.Context, c *Client, reference string) error {
	if strings.TrimSpace(reference) == "" {
		return fmt.Errorf("external_authorization_reference_required")
	}
	if err := j.check(ctx, c); err != nil {
		return err
	}
	for _, item := range j.State.Items {
		r, err := j.current(ctx, c, item)
		if err != nil {
			return err
		}
		if r.TransmissionID != "" {
			continue
		}
		var validated struct {
			Valid           bool `json:"valid"`
			SchemaValidated bool `json:"schema_validated"`
		}
		if err = c.call(ctx, "POST", "/v1/filings/"+item.FilingID+"/validate", "", nil, &validated); err != nil {
			return err
		}
		if !validated.Valid || !validated.SchemaValidated {
			return fmt.Errorf("fresh_validation_failed")
		}
		if _, err = j.current(ctx, c, item); err != nil {
			return err
		}
		var approved Resource
		if err = c.call(ctx, "POST", "/v1/filings/"+item.FilingID+"/approve", "", map[string]any{"sha256": item.SHA256, "authorization_reference": reference, "confirm": true}, &approved); err != nil {
			return err
		}
		if !validResource(approved, j.State.Mode) || approved.ID != item.FilingID || approved.Tenant != item.Tenant || approved.SHA256 != item.SHA256 || approved.Approval == nil || approved.Approval.SHA256 != item.SHA256 {
			return fmt.Errorf("invalid_approval_response")
		}
	}
	return nil
}

func (j *Journal) Submit(ctx context.Context, c *Client, index int, challenge, captcha string) error {
	if err := j.check(ctx, c); err != nil {
		return err
	}
	if index < 0 || index >= len(j.State.Items) {
		return fmt.Errorf("batch_index_out_of_range")
	}
	item := &j.State.Items[index]
	r, err := j.current(ctx, c, *item)
	if err != nil {
		return err
	}
	if r.TransmissionID != "" {
		if !safeID.MatchString(r.TransmissionID) {
			return fmt.Errorf("invalid_transmission_id")
		}
		item.TransmissionID = r.TransmissionID
		return j.save()
	}
	if item.TransmissionID != "" {
		return fmt.Errorf("journal_server_transmission_mismatch")
	}
	if r.Approval == nil || r.Approval.SHA256 != item.SHA256 {
		return fmt.Errorf("owner_approval_required_for_current_hash")
	}
	var tx Resource
	err = c.call(ctx, "POST", "/v1/filings/"+item.FilingID+"/submit", item.SubmitKey, map[string]string{"sha256": item.SHA256, "challenge": challenge, "captcha_token": captcha}, &tx)
	if err != nil {
		return err
	}
	if !validResource(tx, j.State.Mode) || tx.FilingID != item.FilingID || tx.Tenant != item.Tenant || tx.SHA256 != item.SHA256 {
		return fmt.Errorf("invalid_submission_response_preserve_journal")
	}
	item.TransmissionID = tx.ID
	return j.save()
}

func (j *Journal) Status(ctx context.Context, c *Client) ([]Resource, error) {
	if err := j.check(ctx, c); err != nil {
		return nil, err
	}
	out := []Resource{}
	for i := range j.State.Items {
		item := &j.State.Items[i]
		r, err := j.current(ctx, c, *item)
		if err != nil {
			return nil, err
		}
		if r.TransmissionID != "" {
			if !safeID.MatchString(r.TransmissionID) {
				return nil, fmt.Errorf("invalid_transmission_id")
			}
			item.TransmissionID = r.TransmissionID
		}
		if item.TransmissionID != "" {
			var tx Resource
			if !safeID.MatchString(item.TransmissionID) {
				return nil, fmt.Errorf("invalid_transmission_id")
			}
			if err = c.call(ctx, "GET", "/v1/transmissions/"+item.TransmissionID, "", nil, &tx); err != nil {
				return nil, err
			}
			if !validResource(tx, j.State.Mode) || tx.ID != item.TransmissionID || tx.FilingID != item.FilingID || tx.Tenant != item.Tenant || tx.SHA256 != item.SHA256 {
				return nil, fmt.Errorf("invalid_transmission_response")
			}
			r = tx
		}
		r.Approval = nil
		r.Validation = nil
		out = append(out, r)
	}
	if err := j.save(); err != nil {
		return nil, err
	}
	return out, nil
}

type Challenge struct {
	Challenge string `json:"challenge"`
	Action    string `json:"action"`
	ExpiresAt string `json:"expires_at"`
}

func (j *Journal) Challenge(ctx context.Context, c *Client, index int) (Challenge, error) {
	if err := j.check(ctx, c); err != nil {
		return Challenge{}, err
	}
	if index < 0 || index >= len(j.State.Items) {
		return Challenge{}, fmt.Errorf("batch_index_out_of_range")
	}
	item := j.State.Items[index]
	if _, err := j.current(ctx, c, item); err != nil {
		return Challenge{}, err
	}
	var result Challenge
	err := c.call(ctx, "POST", "/v1/filings/"+item.FilingID+"/submit-challenge", "", map[string]string{"sha256": item.SHA256}, &result)
	return result, err
}
