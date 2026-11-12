package firetoiris

import (
	"fmt"
	"strings"
	"testing"
)

func TestEveryFIREFormConvertsRepresentativeExport(t *testing.T) {
	for _, profile := range Profiles() {
		for _, year := range []int{2025, 2026} {
			t.Run(fmt.Sprintf("%d-%s", year, profile.Form), func(t *testing.T) {
				rows := fixture(profile.Code)
				for _, i := range []int{0, 1, 2} {
					rows = change(rows, i, 2, 5, fmt.Sprint(year))
				}
				code := ""
				for _, c := range AmountCodes {
					if profile.Amounts[string(c)] != "" {
						code = string(c)
						break
					}
				}
				rows = change(rows, 1, 28, 45, code)
				for j := 0; j < 18; j++ {
					rows = change(rows, 2, 55+12*j, 66+12*j, "000000000000")
					rows = change(rows, 3, 16+18*j, 33+18*j, "000000000000000000")
				}
				index := strings.Index(AmountCodes, code)
				rows = change(rows, 2, 55+12*index, 66+12*index, "000000123456")
				rows = change(rows, 3, 16+18*index, 33+18*index, "000000000000123456")
				set := func(a, z int, v string) { rows = change(rows, 2, a, z, v) }
				date := fmt.Sprintf("%d0105", year)
				switch profile.Form {
				case "1097BTC":
					set(547, 547, "1")
					set(556, 556, "A")
					set(560, 598, "BOND-1")
					set(599, 601, "101")
				case "1098":
					set(544, 551, date)
					set(552, 552, "1")
				case "1098C":
					set(546, 546, "1")
					set(652, 659, date)
					set(660, 660, "2")
				case "1098F":
					set(544, 551, date)
				case "1098Q":
					set(546, 553, date)
					set(555, 556, "05")
				case "1098T":
					set(544, 544, "1")
					set(547, 547, "1")
				case "1099A":
					set(548, 555, date)
					set(547, 547, "1")
					set(556, 594, "SYNTHETIC PROPERTY")
				case "1099B":
					set(546, 546, "1")
					set(547, 547, "1")
					set(548, 555, date)
					set(569, 607, "SYNTHETIC SHARES")
					set(608, 615, fmt.Sprintf("%d0101", year))
					set(617, 617, "A")
				case "1099C":
					set(547, 547, "F")
					set(548, 555, date)
					set(556, 594, "SYNTHETIC DEBT")
				case "1099CAP":
					set(548, 555, date)
					set(608, 615, "00000010")
					set(616, 625, "COMMON")
				case "1099K":
					set(547, 547, "1")
					set(548, 548, "1")
					set(549, 561, "0000000000010")
					set(605, 608, "0000")
				case "1099LS":
					set(546, 553, date)
					set(663, 701, "TEST CONTACT")
				case "1099LTC":
					set(547, 547, "1")
					set(548, 556, "000111111")
					set(557, 596, "DOE JANE")
					set(597, 636, "123 TEST ST")
					set(637, 676, "SEATTLE")
					set(677, 678, "WA")
					set(679, 687, "98101")
					set(688, 688, "1")
					set(689, 696, date)
					set(697, 697, "1")
				case "1099OID":
					set(547, 585, "SYNTHETIC OBLIGATION")
				case "1099Q":
					set(548, 548, "3")
				case "1099R":
					set(545, 546, "7")
				case "1099S":
					set(548, 555, date)
					set(556, 594, "123 TEST ST SEATTLE WA 98101")
				case "1099SA":
					set(545, 545, "1")
					set(548, 548, "1")
				case "3921", "3922":
					set(547, 554, fmt.Sprintf("%d0101", year))
					set(555, 562, date)
					set(563, 570, "00000010")
					if profile.Form == "3922" {
						set(571, 578, date)
					}
				case "5498":
					set(547, 547, "1")
				case "5498SA":
					set(548, 548, "1")
				case "W2G":
					set(547, 547, "7")
					set(548, 555, date)
					set(556, 570, "SYNTHETIC-1")
				}
				batch, err := Parse(strings.NewReader(strings.Join(rows, "\n")), year)
				if err != nil {
					t.Fatal(err)
				}
				inputs, err := Convert(batch, Options{NameOrder: "last-first"})
				if err != nil {
					t.Fatal(err)
				}
				if err = CheckEnvelope(inputs[0]); err != nil {
					t.Fatal(err)
				}
				target := profile.Amounts[code]
				if year == 2025 && profile.Form == "1099S" {
					target = "GrossProceedsAmt"
				}
				if inputs[0].Records[0][target] != "1234.56" {
					t.Fatal("payment did not survive migration")
				}
			})
		}
	}
}
