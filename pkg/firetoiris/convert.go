package firetoiris

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Input struct {
	TaxYear int              `json:"tax_year"`
	Form    string           `json:"form"`
	Header  map[string]any   `json:"header"`
	Records []map[string]any `json:"records"`
}

type Options struct{ NameOrder string }

func Convert(batch Batch, options Options) ([]Input, error) {
	if options.NameOrder != "" && options.NameOrder != "last-first" && options.NameOrder != "first-last" {
		return nil, problem(0, "name_order", "choose_first_last_or_last_first")
	}
	profiles := map[string]Profile{}
	for _, p := range Profiles() {
		profiles[p.Code] = p
	}
	out := []Input{}
	for _, g := range batch.Groups {
		p, ok := profiles[g.Code]
		if !ok {
			return nil, problem(g.Number, "26-27", "unknown_FIRE_code_use_schema_driven_form_input")
		}
		if field(g.Raw, 6, 6) != "" {
			return nil, problem(g.Number, "6", "state_election_requires_reviewed_form_input")
		}
		if field(g.Raw, 25, 25) != "" || field(g.Raw, 52, 52) != "" || field(g.Raw, 133, 133) != "0" {
			return nil, problem(g.Number, "issuer", "foreign_final_or_transfer_agent_requires_reviewed_form_input")
		}
		if err := blankRanges(g.Raw, g.Number, [][2]int{{7, 11}, {46, 51}, {240, 499}, {508, 750}}); err != nil {
			return nil, err
		}
		payer, err := businessParty(g.Raw, g.Number, 12, 53, 92, 93, 132, 134, 173, 174, 213, 214, 215, 216, 224)
		if err != nil {
			return nil, err
		}
		payer["ForeignEntityInd"] = "0"
		if v := field(g.Raw, 21, 24); v != "" {
			payer["BusinessNameControlTxt"] = v
		}
		if v := field(g.Raw, 225, 239); v != "" {
			payer["PhoneNum"] = v
		}
		schema, err := schemaFor(batch.TaxYear, p.Form)
		if err != nil {
			return nil, problem(g.Number, "form", "schema_unavailable")
		}
		header := map[string]any{"IssuerDetail": payer}
		if hasKey(schema.Header, "CFSFElectionInd") {
			header["CFSFElectionInd"] = "0"
		}
		if name := field(batch.Transmitter, 304, 343); name != "" {
			contact := map[string]any{"ContactPersonNm": name, "ContactPhoneNum": field(batch.Transmitter, 344, 358)}
			if email := field(batch.Transmitter, 359, 408); email != "" {
				contact["ContactEmailAddressTxt"] = email
			}
			header["ContactPersonInformationGrp"] = contact
		}
		input := Input{TaxYear: batch.TaxYear, Form: p.Form, Header: header, Records: []map[string]any{}}
		codes := field(g.Raw, 28, 45)
		if codes == "" {
			return nil, problem(g.Number, "28-45", "amount_codes_required")
		}
		last := -1
		for _, c := range codes {
			idx := strings.IndexRune(AmountCodes, c)
			if idx < 0 || idx <= last {
				return nil, problem(g.Number, "28-45", "amount_codes_must_be_unique_and_sorted")
			}
			last = idx
			if p.Amounts[string(c)] == "" && !(p.Form == "1099R" && c == 'A') {
				return nil, problem(g.Number, "28-45", "amount_code_requires_reviewed_form_input")
			}
		}
		for _, r := range g.Records {
			yearProfile := p
			if batch.TaxYear == 2025 && p.Form == "1099S" {
				yearProfile = profile(p.Code, p.Form, "2:GrossProceedsAmt 5:BuyerRealEstateTaxAmt")
				yearProfile.Fields = p.Fields
			}
			rec, err := convertPayee(r, yearProfile, codes, schema.Record, options)
			if err != nil {
				return nil, err
			}
			input.Records = append(input.Records, rec)
			if len(input.Records) == 5000 {
				out = append(out, input)
				input.Records = []map[string]any{}
			}
		}
		if len(input.Records) > 0 {
			out = append(out, input)
		}
	}
	return out, nil
}

func convertPayee(r Payee, p Profile, codes string, schema ContractNode, options Options) (map[string]any, error) {
	s := r.Raw
	if field(s, 6, 6) != "" {
		return nil, problem(r.Number, "6", "corrections_require_source_linked_workflow")
	}
	if field(s, 287, 287) != "" {
		return nil, problem(r.Number, "287", "foreign_address_requires_reviewed_form_input")
	}
	if err := blankRanges(s, r.Number, [][2]int{{45, 54}, {271, 286}, {408, 447}, {499, 499}, {508, 543}, {749, 750}}); err != nil {
		return nil, err
	}
	party, err := businessParty(s, r.Number, 12, 288, 327, 328, 367, 368, 407, 448, 487, 488, 489, 490, 498)
	if err != nil {
		return nil, err
	}
	switch field(s, 11, 11) {
	case "1":
		if v := field(s, 7, 10); v != "" {
			party["BusinessNameControlTxt"] = v
		}
	case "2":
		parts := strings.Fields(field(s, 288, 327))
		if options.NameOrder == "" || len(parts) != 2 || field(s, 328, 367) != "" {
			return nil, problem(r.Number, "288-367", "ambiguous_person_name_requires_explicit_order_or_form_input")
		}
		first, last := parts[0], parts[1]
		if options.NameOrder == "last-first" {
			first, last = last, first
		}
		delete(party, "BusinessName")
		party["TINSubmittedTypeCd"] = "INDIVIDUAL_TIN"
		party["PersonName"] = map[string]any{"PersonFirstNm": first, "PersonLastNm": last}
		if v := field(s, 7, 10); v != "" {
			party["PersonNameControlTxt"] = v
		}
	default:
		return nil, problem(r.Number, "11", "TIN_type_required")
	}
	rec := map[string]any{"RecipientDetail": party}
	if v := field(s, 21, 40); v != "" {
		if p.Form == "1099LS" || p.Form == "1099SB" {
			rec["PolicyNum"] = v
		} else {
			rec["RecipientAccountNum"] = v
		}
	}
	if v := field(s, 41, 44); v != "" {
		rec["IssuerOfficeCd"] = v
	}
	for j, amount := range r.Amounts {
		code := string(AmountCodes[j])
		declared := strings.Contains(codes, code)
		if !declared {
			if amount != 0 {
				return nil, problem(r.Number, "amount_"+code, "undeclared_nonzero_amount")
			}
			continue
		}
		if p.Form == "1099R" && code == "A" {
			if amount != r.Amounts[0] || field(s, 548, 548) != "1" {
				return nil, problem(r.Number, "amount_A", "IRA_total_requires_review")
			}
			continue
		}
		if amount < 0 && !((p.Form == "1099B" && strings.Contains("29ABC", code)) || (p.Form == "1099OID" && code == "6") || (p.Form == "1099Q" && code == "2")) {
			return nil, problem(r.Number, "amount_"+code, "negative_not_allowed")
		}
		if err := setPath(rec, p.Amounts[code], decimal(amount)); err != nil {
			return nil, problem(r.Number, "amount_"+code, "target_conflict")
		}
	}
	for _, target := range p.Amounts {
		if !strings.Contains(target, ".") && requiredElement(schema, target) {
			if _, exists := rec[target]; !exists {
				rec[target] = "0.00"
			}
		}
	}
	used := [750]bool{}
	for i := 0; i < 543; i++ {
		used[i] = true
	}
	for _, m := range p.Fields {
		value, exists, err := mappedValue(field(s, m.First, m.Last), m.Format)
		if m.Format == "person" {
			value, err = personName(field(s, m.First, m.Last), options.NameOrder)
			exists = err == nil
		}
		if err != nil {
			return nil, problem(r.Number, fmt.Sprintf("%d-%d", m.First, m.Last), "invalid_"+m.Format)
		}
		if exists {
			if err = setPath(rec, m.Target, value); err != nil {
				return nil, problem(r.Number, m.Target, "target_conflict")
			}
		}
		for i := m.First - 1; i < m.Last; i++ {
			used[i] = true
		}
	}
	for _, bounds := range [][2]int{{723, 734}, {735, 746}} {
		v := field(s, bounds[0], bounds[1])
		if v != "" && strings.Trim(v, "0") == "" {
			for i := bounds[0] - 1; i < bounds[1]; i++ {
				used[i] = true
			}
		}
	}
	for i := 543; i < 748; i++ {
		if !used[i] && s[i] != ' ' {
			return nil, problem(r.Number, fmt.Sprintf("position_%d", i+1), "unmapped_material_data_use_reviewed_form_input")
		}
	}
	if p.Form == "1099B" {
		noncovered, checkbox := field(s, 545, 545), field(s, 617, 617)
		reported := noncovered == "2" || strings.Contains("AD", checkbox) && checkbox != ""
		if noncovered == "1" && reported {
			return nil, problem(r.Number, "545/617", "conflicting_basis_reporting")
		}
		if noncovered == "2" && checkbox != "" && !strings.Contains("AD", checkbox) {
			return nil, problem(r.Number, "545/617", "conflicting_basis_reporting")
		}
		rec["BasisReportedInd"] = "0"
		if reported {
			rec["BasisReportedInd"] = "1"
		}
	}
	requiredBooleans(schema, rec)
	return rec, nil
}

func requiredBooleans(n ContractNode, obj map[string]any) {
	for _, c := range n.Children {
		if c.Kind == "element" {
			if c.MinOccurs > 0 && !c.Automatic && (strings.HasSuffix(c.Type, "DigitBooleanType") || strings.HasSuffix(c.Type, "BooleanType")) {
				if _, ok := obj[c.Key]; !ok {
					obj[c.Key] = "0"
				}
			}
		} else if c.Kind != "choice" {
			requiredBooleans(c, obj)
		}
	}
}
func hasKey(n ContractNode, key string) bool {
	if n.Key == key {
		return true
	}
	for _, c := range n.Children {
		if hasKey(c, key) {
			return true
		}
	}
	return false
}
func businessParty(s string, n, tin, n1, n2, n3, n4, a1, a2, c1, c2, s1, s2, z1, z2 int) (map[string]any, error) {
	id := field(s, tin, tin+8)
	if len(id) != 9 {
		return nil, problem(n, "TIN", "nine_digits_required")
	}
	if _, err := unsigned(id); err != nil {
		return nil, problem(n, "TIN", "nine_digits_required")
	}
	name := map[string]any{"BusinessNameLine1Txt": field(s, n1, n2)}
	if v := field(s, n3, n4); v != "" {
		name["BusinessNameLine2Txt"] = v
	}
	return map[string]any{"TIN": id, "TINSubmittedTypeCd": "BUSINESS_TIN", "BusinessName": name, "MailingAddressGrp": map[string]any{"USAddress": map[string]any{"AddressLine1Txt": field(s, a1, a2), "CityNm": field(s, c1, c2), "StateAbbreviationCd": field(s, s1, s2), "ZIPCd": field(s, z1, z2)}}}, nil
}
func blankRanges(s string, n int, ranges [][2]int) error {
	for _, r := range ranges {
		if field(s, r[0], r[1]) != "" {
			return problem(n, fmt.Sprintf("%d-%d", r[0], r[1]), "reserved_data_requires_review")
		}
	}
	return nil
}
func setPath(obj map[string]any, path string, value any) error {
	keys := strings.Split(path, ".")
	if path == "" {
		return fmt.Errorf("empty target")
	}
	for _, k := range keys[:len(keys)-1] {
		v, ok := obj[k]
		if !ok {
			v = map[string]any{}
			obj[k] = v
		}
		nested, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("conflict")
		}
		obj = nested
	}
	key := keys[len(keys)-1]
	if _, ok := obj[key]; ok {
		return fmt.Errorf("duplicate")
	}
	obj[key] = value
	return nil
}
func mappedValue(raw, format string) (any, bool, error) {
	if format == "bool" || format == "notice" {
		yes := "1"
		if format == "notice" {
			yes = "2"
		}
		if raw == "" {
			return "0", true, nil
		}
		if raw == yes {
			return "1", true, nil
		}
		return nil, false, fmt.Errorf("invalid indicator")
	}
	if strings.HasPrefix(format, "enum:") {
		pieces := strings.Split(format, ":")
		if raw != "" && (len(raw) != 1 || !strings.Contains(pieces[1], raw)) {
			return nil, false, fmt.Errorf("invalid code")
		}
		value := "0"
		if raw != "" && strings.Contains(pieces[2], raw) {
			value = "1"
		}
		return value, true, nil
	}
	if raw == "" {
		return nil, false, nil
	}
	switch format {
	case "text":
		return raw, true, nil
	case "percent":
		v, err := unsigned(raw)
		return fmt.Sprintf("0.%02d", v), true, err
	case "integer":
		v, err := unsigned(raw)
		return strconv.FormatInt(v, 10), true, err
	case "date":
		v, err := time.Parse("20060102", raw)
		return v.Format("2006-01-02"), true, err
	case "codes":
		out := []any{}
		for _, c := range raw {
			out = append(out, string(c))
		}
		return out, true, nil
	}
	return nil, false, fmt.Errorf("unknown format")
}

func personName(raw, order string) (map[string]any, error) {
	parts := strings.Fields(raw)
	if len(parts) != 2 || (order != "first-last" && order != "last-first") {
		return nil, fmt.Errorf("unambiguous_person_name_and_explicit_order_required")
	}
	first, last := parts[0], parts[1]
	if order == "last-first" {
		first, last = last, first
	}
	return map[string]any{"PersonFirstNm": first, "PersonLastNm": last}, nil
}

func requiredElement(n ContractNode, key string) bool {
	for _, c := range n.Children {
		if c.Key == key {
			return c.MinOccurs > 0
		}
		if c.Kind != "element" && requiredElement(c, key) {
			return true
		}
	}
	return false
}
