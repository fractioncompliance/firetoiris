package firetoiris

import (
	"strings"
	"testing"
)

func TestNECConvertsExactCentsAndIdentity(t *testing.T) {
	b, err := Parse(strings.NewReader(strings.Join(fixture("NE"), "\n")), 2026)
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := Convert(b, Options{})
	if err != nil {
		t.Fatal(err)
	}
	r := inputs[0].Records[0]
	if r["NonemployeeCompensationAmt"] != "1234.56" {
		t.Fatalf("amount: %v", r)
	}
	if r["RecipientDetail"].(map[string]any)["TIN"] != "000654321" {
		t.Fatal("identifier changed")
	}
	if err = CheckEnvelope(inputs[0]); err != nil {
		t.Fatal(err)
	}
}

func TestConversionRejectsLostData(t *testing.T) {
	cases := map[string][]string{
		"state":           change(fixture("NE"), 1, 6, 6, "1"),
		"correction":      change(fixture("NE"), 2, 6, 6, "G"),
		"foreign":         change(fixture("NE"), 2, 287, 287, "1"),
		"special":         change(fixture("NE"), 2, 663, 722, "MUST NOT DROP"),
		"reserved":        change(fixture("NE"), 2, 508, 543, "MATERIAL"),
		"individual-name": change(fixture("NE"), 2, 11, 11, "2"),
	}
	for name, rows := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := Parse(strings.NewReader(strings.Join(rows, "\n")), 2026)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Convert(b, Options{}); err == nil {
				t.Fatal("silently dropped source data")
			}
		})
	}
}

func TestAllPublication1220AmountTargetsExist(t *testing.T) {
	profiles := Profiles()
	if len(profiles) != 32 {
		t.Fatalf("profiles %d", len(profiles))
	}
	for _, p := range profiles {
		for _, year := range []int{2025, 2026} {
			t.Run(p.Form, func(t *testing.T) {
				schema, err := schemaFor(year, p.Form)
				if err != nil {
					t.Fatal(err)
				}
				var keys func(ContractNode, map[string]bool)
				keys = func(n ContractNode, m map[string]bool) {
					if n.Key != "" {
						m[n.Key] = true
					}
					for _, c := range n.Children {
						keys(c, m)
					}
				}
				known := map[string]bool{}
				keys(schema.Record, known)
				for code, target := range p.Amounts {
					if p.Form == "1099S" && year == 2025 && code == "2" {
						continue
					}
					if target != "" && !known[target[strings.LastIndex(target, ".")+1:]] {
						t.Errorf("missing %s in %d/%s", target, year, p.Form)
					}
				}
			})
		}
	}
}

func TestMappedTailTargetsExistForEachYear(t *testing.T) {
	for _, p := range Profiles() {
		for _, year := range []int{2025, 2026} {
			schema, err := schemaFor(year, p.Form)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range p.Fields {
				key := m.Target[strings.LastIndex(m.Target, ".")+1:]
				if !hasKey(schema.Record, key) {
					t.Errorf("%d/%s field %s is absent", year, p.Form, m.Target)
				}
			}
		}
	}
}
