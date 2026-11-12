package firetoiris

import (
	"fmt"
	"sort"

	"strconv"
	"strings"
)

type Capability struct {
	TaxYear            int    `json:"tax_year"`
	Form               string `json:"form"`
	FIRECode           string `json:"fire_code,omitempty"`
	InputMode          string `json:"input_mode"`
	ProductionVerified bool   `json:"production_verified"`
}

func Catalog() ([]Capability, error) {
	entries, err := contracts.ReadDir("contracts")
	if err != nil {
		return nil, fmt.Errorf("contract_catalog_unavailable")
	}
	codes := map[string]string{}
	for _, p := range Profiles() {
		codes[p.Form] = p.Code
	}
	out := []Capability{}
	for _, year := range entries {
		if !year.IsDir() {
			continue
		}
		forms, err := contracts.ReadDir("contracts/" + year.Name())
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(year.Name())
		if err != nil {
			return nil, err
		}
		for _, f := range forms {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			form := strings.TrimSuffix(f.Name(), ".json")
			mode := "reviewed_form_input"
			if codes[form] != "" {
				mode = "FIRE_mapping_with_blocking_checks_or_reviewed_form_input"
			}
			out = append(out, Capability{TaxYear: n, Form: form, FIRECode: codes[form], InputMode: mode})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TaxYear != out[j].TaxYear {
			return out[i].TaxYear < out[j].TaxYear
		}
		return out[i].Form < out[j].Form
	})
	return out, nil
}
func Schema(year int, form string) (any, error) { return schemaFor(year, form) }
