package master

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestParse(t *testing.T) {
	vendors := registry["vendors"]
	roles := registry["project-roles"]

	tests := []struct {
		name     string
		def      resourceDef
		body     string
		create   bool
		wantErrs []string // key field yang harus error; kosong = harus valid
	}{
		{"create valid", vendors, `{"name":"PT Baru"}`, true, nil},
		{"create tanpa name", vendors, `{}`, true, []string{"name"}},
		{"create name spasi doang", vendors, `{"name":"   "}`, true, []string{"name"}},
		{"create isActive ditolak", vendors, `{"name":"X","isActive":false}`, true, []string{"isActive"}},
		{"patch isActive ok", vendors, `{"isActive":false}`, false, nil},
		{"patch tipe salah", vendors, `{"sortOrder":"dua"}`, false, []string{"sortOrder"}},
		{"patch null", vendors, `{"name":null}`, false, []string{"name"}},
		{"patch code project-roles ditolak", roles, `{"code":"X"}`, false, []string{"code"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tt.body), &body); err != nil {
				t.Fatal(err)
			}
			_, errs := tt.def.parse(body, tt.create)
			if len(errs) != len(tt.wantErrs) {
				t.Fatalf("errs=%v want keys=%v", errs, tt.wantErrs)
			}
			for _, k := range tt.wantErrs {
				if _, ok := errs[k]; !ok {
					t.Fatalf("expected error on %q, got %v", k, errs)
				}
			}
		})
	}
}

func TestIsInUse(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"23001", true},  // ON DELETE RESTRICT (schema kita)
		{"23503", true},  // NO ACTION
		{"23505", false}, // unique violation bukan "in use"
		{"42P01", false},
	}
	for _, tt := range tests {
		err := fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: tt.code})
		if got := isInUse(err); got != tt.want {
			t.Errorf("code %s: got %v want %v", tt.code, got, tt.want)
		}
	}
	if isInUse(errors.New("bukan pg error")) {
		t.Error("error non-pg nggak boleh dianggap in-use")
	}
}

func TestRoleCode(t *testing.T) {
	tests := map[string]string{
		"QA Lead":          "QA_LEAD",
		"  scrum  master ": "SCRUM_MASTER",
		"Dev-Ops/SRE":      "DEV_OPS_SRE",
		"Ünit 2":           "NIT_2",
		"!!!":              "",
	}
	for in, want := range tests {
		if got := roleCode(in); got != want {
			t.Errorf("roleCode(%q) = %q, want %q", in, got, want)
		}
	}
}
