package firetoiris

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

const Version = "0.1.0"
const AmountCodes = "123456789ABCDEFGHJ"
const maxInputBytes = 128 << 20

type FieldError struct {
	Record int
	Field  string
	Code   string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("record %d, %s: %s", e.Record, e.Field, e.Code)
}
func problem(n int, field, code string) error { return &FieldError{n, field, code} }

type Payee struct {
	Number  int
	Raw     string
	Amounts [18]int64
}
type Group struct {
	Number  int
	Raw     string
	Code    string
	Records []Payee
}
type Batch struct {
	SHA256      string
	TaxYear     int
	Transmitter string
	Groups      []Group
	Payees      int
}

func field(s string, first, last int) string { return strings.TrimSpace(s[first-1 : last]) }

func Parse(r io.Reader, year int) (Batch, error) {
	var out Batch
	if year != 2025 && year != 2026 {
		return out, problem(0, "tax_year", "supported_years_are_2025_and_2026")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxInputBytes+1))
	if err != nil {
		return out, problem(0, "input", "read_failed")
	}
	if len(data) > maxInputBytes {
		return out, problem(0, "input", "exceeds_128_MiB")
	}
	digest := sha256.Sum256(data)
	out.SHA256 = hex.EncodeToString(digest[:])
	out.TaxYear = year
	rows, err := splitRecords(data)
	if err != nil {
		return Batch{}, err
	}
	state := "start"
	total := 0
	var sums [18]int64
	for i, s := range rows {
		n := i + 1
		seq, err := unsigned(field(s, 500, 507))
		if err != nil || seq != int64(n) {
			return Batch{}, problem(n, "500-507", "invalid_record_sequence")
		}
		if s[0] == 'T' || s[0] == 'A' || s[0] == 'B' {
			if field(s, 2, 5) != strconv.Itoa(year) {
				return Batch{}, problem(n, "2-5", "tax_year_mismatch")
			}
		}
		switch s[0] {
		case 'T':
			if state != "start" {
				return Batch{}, problem(n, "record_type", "transmitter_must_be_first_and_unique")
			}
			out.Transmitter = s
			state = "transmitter"
		case 'A':
			if state != "transmitter" && state != "closed" {
				return Batch{}, problem(n, "record_type", "issuer_out_of_order")
			}
			out.Groups = append(out.Groups, Group{Number: n, Raw: s, Code: field(s, 26, 27)})
			sums = [18]int64{}
			state = "issuer"
		case 'B':
			if state != "issuer" && state != "payee" {
				return Batch{}, problem(n, "record_type", "payee_without_issuer")
			}
			rec := Payee{Number: n, Raw: s}
			for j := range rec.Amounts {
				v, e := money(s[54+j*12 : 66+j*12])
				if e != nil {
					return Batch{}, problem(n, fmt.Sprintf("amount_%c", AmountCodes[j]), "invalid_cents")
				}
				if (v > 0 && sums[j] > math.MaxInt64-v) || (v < 0 && sums[j] < math.MinInt64-v) {
					return Batch{}, problem(n, "totals", "overflow")
				}
				rec.Amounts[j] = v
				sums[j] += v
			}
			g := &out.Groups[len(out.Groups)-1]
			g.Records = append(g.Records, rec)
			total++
			state = "payee"
		case 'C':
			if state != "payee" {
				return Batch{}, problem(n, "record_type", "issuer_total_without_payees")
			}
			g := out.Groups[len(out.Groups)-1]
			count, e := unsigned(field(s, 2, 9))
			if e != nil || count != int64(len(g.Records)) {
				return Batch{}, problem(n, "2-9", "payee_count_mismatch")
			}
			for j, want := range sums {
				got, e := money(s[15+j*18 : 33+j*18])
				if e != nil || got != want {
					return Batch{}, problem(n, fmt.Sprintf("control_total_%c", AmountCodes[j]), "amount_total_mismatch")
				}
			}
			state = "closed"
		case 'K':
			return Batch{}, problem(n, "record_type", "combined_state_totals_require_reviewed_form_input")
		case 'F':
			if state != "closed" || n != len(rows) {
				return Batch{}, problem(n, "record_type", "end_out_of_order")
			}
			count, e := unsigned(field(s, 2, 9))
			if e != nil || (count != 0 && count != int64(len(out.Groups))) {
				return Batch{}, problem(n, "2-9", "issuer_count_mismatch")
			}
			t := field(out.Transmitter, 296, 303)
			f := field(s, 50, 57)
			if t == "" && f == "" {
				return Batch{}, problem(n, "50-57", "payee_count_required")
			}
			for _, c := range []string{t, f} {
				if c != "" {
					v, e := unsigned(c)
					if e != nil || v != int64(total) {
						return Batch{}, problem(n, "payee_count", "payee_count_mismatch")
					}
				}
			}
			state = "end"
		default:
			return Batch{}, problem(n, "record_type", "unknown_record_type")
		}
	}
	if state != "end" {
		return Batch{}, problem(len(rows), "input", "incomplete_file")
	}
	out.Payees = total
	return out, nil
}

func splitRecords(data []byte) ([]string, error) {
	rows := []string{}
	for pos := 0; pos < len(data); {
		start := pos
		for pos < len(data) && pos-start < 750 && data[pos] != '\r' && data[pos] != '\n' {
			if data[pos] < 32 || data[pos] > 126 {
				return nil, problem(len(rows)+1, "encoding", "ASCII_required")
			}
			pos++
		}
		width := pos - start
		if width != 748 && width != 750 {
			return nil, problem(len(rows)+1, "length", "expected_750_bytes_or_748_plus_CRLF")
		}
		if width == 748 {
			if pos+1 >= len(data) || data[pos] != '\r' || data[pos+1] != '\n' {
				return nil, problem(len(rows)+1, "length", "748_requires_CRLF")
			}
			rows = append(rows, string(data[start:pos])+"  ")
			pos += 2
		} else {
			rows = append(rows, string(data[start:pos]))
			if pos < len(data) && data[pos] == '\r' {
				if pos+1 >= len(data) || data[pos+1] != '\n' {
					return nil, problem(len(rows), "line_ending", "CR_requires_LF")
				}
				pos += 2
			} else if pos < len(data) && data[pos] == '\n' {
				pos++
			}
		}
	}
	if len(rows) == 0 {
		return nil, problem(0, "input", "empty_file")
	}
	return rows, nil
}

func unsigned(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("digit_required")
		}
	}
	return strconv.ParseInt(s, 10, 64)
}
func money(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	sign := int64(1)
	if s[0] == '+' || s[0] == '-' {
		if s[0] == '-' {
			sign = -1
		}
		s = s[1:]
	}
	v, e := unsigned(s)
	return sign * v, e
}
func decimal(v int64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}
