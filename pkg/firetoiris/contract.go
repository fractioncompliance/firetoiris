package firetoiris

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
)

//go:embed contracts/*/*.json contracts/manifest.json
var contracts embed.FS

type ContractFacet struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
type ContractNode struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	Name      string          `json:"name,omitempty"`
	Key       string          `json:"key,omitempty"`
	Label     string          `json:"label"`
	MinOccurs int             `json:"min_occurs"`
	MaxOccurs int             `json:"max_occurs"`
	Type      string          `json:"type,omitempty"`
	Base      string          `json:"base,omitempty"`
	Facets    []ContractFacet `json:"facets,omitempty"`
	Enums     []string        `json:"enums,omitempty"`
	Children  []ContractNode  `json:"children,omitempty"`
	Automatic bool            `json:"automatic"`
	Sensitive bool            `json:"sensitive"`
	ValueKey  string          `json:"value_key,omitempty"`
}
type InputContract struct {
	TaxYear         int          `json:"tax_year"`
	Form            string       `json:"form"`
	SubmissionGroup int          `json:"submission_group"`
	Header          ContractNode `json:"header"`
	Record          ContractNode `json:"record"`
	MaxRecords      int          `json:"max_records"`
}

func schemaFor(year int, form string) (InputContract, error) {
	if year != 2025 && year != 2026 {
		return InputContract{}, fmt.Errorf("unsupported_tax_year")
	}
	if !safeID.MatchString(form) {
		return InputContract{}, fmt.Errorf("unsupported_form")
	}
	raw, err := contracts.ReadFile("contracts/" + strconv.Itoa(year) + "/" + form + ".json")
	if err != nil {
		return InputContract{}, fmt.Errorf("unsupported_form_year")
	}
	manifestRaw, err := contracts.ReadFile("contracts/manifest.json")
	if err != nil {
		return InputContract{}, fmt.Errorf("contract_manifest_unavailable")
	}
	var pins map[string]string
	if json.Unmarshal(manifestRaw, &pins) != nil {
		return InputContract{}, fmt.Errorf("invalid_contract_manifest")
	}
	digest := sha256.Sum256(raw)
	if pins[strconv.Itoa(year)+"/"+form+".json"] != hex.EncodeToString(digest[:]) {
		return InputContract{}, fmt.Errorf("contract_snapshot_hash_mismatch")
	}
	var c InputContract
	if err = json.Unmarshal(raw, &c); err != nil {
		return c, fmt.Errorf("invalid_contract_snapshot")
	}
	return c, nil
}
