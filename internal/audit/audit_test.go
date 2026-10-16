package audit

import "testing"

func sp(s string) *string { return &s }

func TestDiff(t *testing.T) {
	var nilStr *string

	tests := []struct {
		name         string
		old, new     any
		wantRecorded bool
		wantOld      string // "" + wantOldNil
		wantNew      string
		wantOldNil   bool
		wantNewNil   bool
	}{
		{"sama persis", "a", "a", false, "", "", false, false},
		{"beda string", "a", "b", true, "a", "b", false, false},
		{"nil ke nil", nilStr, nilStr, false, "", "", false, false},
		{"nil ke nilai", nilStr, sp("CR-1"), true, "", "CR-1", true, false},
		{"nilai ke nil", sp("CR-1"), nilStr, true, "CR-1", "", false, true},
		{"pointer vs value sama isi", sp("x"), "x", false, "", "", false, false},
		{"bool berubah", false, true, true, "false", "true", false, false},
		{"int berubah", 1, 2, true, "1", "2", false, false},
		{"untyped nil ke nilai", nil, "x", true, "", "x", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Diff
			d.Add("Field", tt.old, tt.new)
			if got := !d.Empty(); got != tt.wantRecorded {
				t.Fatalf("recorded=%v want %v", got, tt.wantRecorded)
			}
			if !tt.wantRecorded {
				return
			}
			e := d.Entries(Entry{ProjectID: "p1", Label: "Project"})[0]
			if e.ProjectID != "p1" || e.Label != "Project" || e.Field != "Field" {
				t.Fatalf("base/field tidak terisi: %+v", e)
			}
			if tt.wantOldNil != (e.Old == nil) || (e.Old != nil && *e.Old != tt.wantOld) {
				t.Errorf("old = %v", e.Old)
			}
			if tt.wantNewNil != (e.New == nil) || (e.New != nil && *e.New != tt.wantNew) {
				t.Errorf("new = %v", e.New)
			}
		})
	}
}
