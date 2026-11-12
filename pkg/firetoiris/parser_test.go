package firetoiris

import (
	"fmt"
	"strings"
	"testing"
)

func put(b []byte, start, end int, value string) {
	copy(b[start-1:end], []byte(fmt.Sprintf("%-*s", end-start+1, value)))
}

func fixture(code string) []string {
	rows := make([]string, 5)
	for i, kind := range []byte("TABCF") {
		b := []byte(strings.Repeat(" ", 750))
		b[0] = kind
		put(b, 500, 507, fmt.Sprintf("%08d", i+1))
		switch kind {
		case 'T':
			put(b, 2, 5, "2026")
			put(b, 296, 303, "00000001")
			put(b, 304, 343, "TEST CONTACT")
			put(b, 344, 358, "2065550100")
		case 'A':
			put(b, 2, 5, "2026")
			put(b, 12, 20, "000123456")
			put(b, 26, 27, code)
			put(b, 28, 45, "1")
			put(b, 53, 92, "SYNTHETIC PAYER")
			put(b, 133, 133, "0")
			put(b, 134, 173, "123 TEST ST")
			put(b, 174, 213, "SEATTLE")
			put(b, 214, 215, "WA")
			put(b, 216, 224, "98101")
		case 'B':
			put(b, 2, 5, "2026")
			put(b, 11, 11, "1")
			put(b, 12, 20, "000654321")
			put(b, 21, 40, "ACCOUNT-01")
			for j := 0; j < 18; j++ {
				put(b, 55+12*j, 66+12*j, "000000000000")
			}
			put(b, 55, 66, "000000123456")
			put(b, 288, 327, "SYNTHETIC RECIPIENT")
			put(b, 368, 407, "456 TEST ST")
			put(b, 448, 487, "SEATTLE")
			put(b, 488, 489, "WA")
			put(b, 490, 498, "98102")
		case 'C':
			put(b, 2, 9, "00000001")
			for j := 0; j < 18; j++ {
				put(b, 16+18*j, 33+18*j, "000000000000000000")
			}
			put(b, 16, 33, "000000000000123456")
		case 'F':
			put(b, 2, 9, "00000001")
			put(b, 10, 30, strings.Repeat("0", 21))
			put(b, 50, 57, "00000001")
		}
		rows[i] = string(b)
	}
	return rows
}

func change(rows []string, row, start, end int, value string) []string {
	out := append([]string(nil), rows...)
	b := []byte(out[row])
	put(b, start, end, value)
	out[row] = string(b)
	return out
}

func TestParseFormatsAndAmounts(t *testing.T) {
	rows := fixture("NE")
	for _, sep := range []string{"", "\n", "\r\n"} {
		t.Run(fmt.Sprintf("separator-%q", sep), func(t *testing.T) {
			b, err := Parse(strings.NewReader(strings.Join(rows, sep)), 2026)
			if err != nil {
				t.Fatal(err)
			}
			if len(b.Groups) != 1 || b.Payees != 1 || b.Groups[0].Records[0].Amounts[0] != 123456 {
				t.Fatalf("bad batch: %+v", b)
			}
		})
	}
	embedded := make([]string, len(rows))
	for i, r := range rows {
		embedded[i] = r[:748] + "\r\n"
	}
	if _, err := Parse(strings.NewReader(strings.Join(embedded, "")), 2026); err != nil {
		t.Fatal(err)
	}
}

func TestParseRejectsInvalidWithoutDisclosingTIN(t *testing.T) {
	cases := map[string][]string{
		"sequence":     change(fixture("NE"), 2, 500, 507, "00000004"),
		"mixed-year":   change(fixture("NE"), 2, 2, 5, "2025"),
		"total":        change(fixture("NE"), 3, 16, 33, "000000000000123457"),
		"payer-count":  change(fixture("NE"), 4, 2, 9, "00000002"),
		"payee-count":  change(fixture("NE"), 0, 296, 303, "00000002"),
		"bad-amount":   change(fixture("NE"), 2, 55, 66, "00000X123456"),
		"no-end":       fixture("NE")[:4],
		"out-of-order": {fixture("NE")[0], fixture("NE")[2]},
	}
	for name, rows := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(strings.Join(rows, "\n")), 2026)
			if err == nil {
				t.Fatal("accepted invalid file")
			}
			if strings.Contains(err.Error(), "000654321") {
				t.Fatal("TIN leaked")
			}
		})
	}
}
