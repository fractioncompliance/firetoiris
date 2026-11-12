package firetoiris

import (
	"encoding/json"
	"fmt"
)

// CheckEnvelope checks local size and shape limits only. TaxComposer's private
// filing module validates form fields, builds XML and evaluates filing rules.
func CheckEnvelope(input Input) error {
	c, err := schemaFor(input.TaxYear, input.Form)
	if err != nil {
		return err
	}
	if len(input.Records) < 1 || len(input.Records) > c.MaxRecords {
		return fmt.Errorf("record_count_outside_contract")
	}
	if len(input.Header) == 0 {
		return fmt.Errorf("header_required")
	}
	for _, r := range input.Records {
		if len(r) == 0 {
			return fmt.Errorf("record_required")
		}
	}
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > 20<<20 {
		return fmt.Errorf("input_not_JSON_or_over_20_MiB")
	}
	return uniqueJSON(raw)
}
