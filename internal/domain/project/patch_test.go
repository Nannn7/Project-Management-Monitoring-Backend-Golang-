package project

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func sp(s string) *string { return &s }

func TestParsePatch(t *testing.T) {
	const id = "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA"
	tests := []struct {
		name    string
		body    string
		want    map[string]any // kolom -> nilai (nil = NULL); dipakai kalau wantErr == ""
		wantErr string         // key field yang harus error
	}{
		{"name di-trim", `{"name":"  Baru  "}`, map[string]any{"name": "Baru"}, ""},
		{"name kosong", `{"name":"  "}`, nil, "name"},
		{"name null", `{"name":null}`, nil, "name"},
		{"name kepanjangan", fmt.Sprintf(`{"name":%q}`, strings.Repeat("x", 201)), nil, "name"},
		{"crNumber kosong jadi NULL", `{"crNumber":""}`, map[string]any{"cr_number": nil}, ""},
		{"crNumber null jadi NULL", `{"crNumber":null}`, map[string]any{"cr_number": nil}, ""},
		{"vendorId kosong jadi NULL", `{"vendorId":""}`, map[string]any{"vendor_id": nil}, ""},
		{"vendorId bukan uuid", `{"vendorId":"abc"}`, nil, "vendorId"},
		{"uuid huruf besar di-lowercase", `{"categoryId":"` + id + `"}`, map[string]any{"category_id": strings.ToLower(id)}, ""},
		{"directorateId null", `{"directorateId":null}`, nil, "directorateId"},
		{"poStatus valid", `{"poStatus":"Draft"}`, map[string]any{"po_status": "Draft"}, ""},
		{"poStatus ngawur", `{"poStatus":"Selesai"}`, nil, "poStatus"},
		{"gitRepo kosong tetap ''", `{"gitRepo":""}`, map[string]any{"git_repo": ""}, ""},
		{"hasMigration true", `{"hasMigration":true}`, map[string]any{"has_migration": true}, ""},
		{"hasMigration tipe salah", `{"hasMigration":"ya"}`, nil, "hasMigration"},
		{"status ditolak", `{"status":"Hold"}`, nil, "status"},
		{"field tak dikenal", `{"createdBy":"x"}`, nil, "createdBy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tt.body), &body); err != nil {
				t.Fatal(err)
			}
			got, errs := parsePatch(body)
			if tt.wantErr != "" {
				if _, ok := errs[tt.wantErr]; !ok {
					t.Fatalf("harusnya error di %q, dapat errs=%v", tt.wantErr, errs)
				}
				return
			}
			if len(errs) != 0 {
				t.Fatalf("errs tak terduga: %v", errs)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
		})
	}
}

func TestParsePatchStatusMessagePointsToStatusEndpoint(t *testing.T) {
	_, errs := parsePatch(map[string]json.RawMessage{"status": json.RawMessage(`"Hold"`)})
	if !strings.Contains(errs["status"], "/status") {
		t.Errorf("pesan harus mengarahkan ke endpoint status: %q", errs["status"])
	}
}

func TestChangedFields(t *testing.T) {
	before := &ProjectDetail{
		Name: "A", CRNumber: sp("CR-1"), DirectorateID: "d1", POStatus: "Belum Ada", GitRepo: "",
	}
	vals := map[string]any{
		"name":           "A",  // sama -> dibuang
		"cr_number":      nil,  // dikosongkan -> berubah
		"vendor_id":      nil,  // sudah NULL -> dibuang
		"git_repo":       "",   // sama -> dibuang
		"has_migration":  true, // false -> true -> berubah
		"directorate_id": "d1", // sama -> dibuang
	}
	got := changedFields(before, vals)
	if len(got) != 2 {
		t.Fatalf("harusnya 2 field berubah (cr_number, has_migration), dapat %v", got)
	}
	if _, ok := got["cr_number"]; !ok {
		t.Error("cr_number harus terdeteksi berubah")
	}
	if _, ok := got["has_migration"]; !ok {
		t.Error("has_migration harus terdeteksi berubah")
	}
}

// Penjaga registry: tiap field harus lengkap, dan JSON/kolom tidak boleh dobel.
func TestPatchFieldsRegistry(t *testing.T) {
	seenJSON, seenCol := map[string]bool{}, map[string]bool{}
	for _, f := range patchFields {
		if f.JSON == "" || f.Col == "" || f.Label == "" || f.Raw == nil {
			t.Errorf("field tidak lengkap: %+v", f)
		}
		if f.Kind == pEnum && len(f.Enum) == 0 {
			t.Errorf("%s: enum kosong", f.JSON)
		}
		if seenJSON[f.JSON] || seenCol[f.Col] {
			t.Errorf("duplikat: %s / %s", f.JSON, f.Col)
		}
		seenJSON[f.JSON], seenCol[f.Col] = true, true
	}
}

func TestNormalizeReason(t *testing.T) {
	long := strings.Repeat("x", 501)
	tests := []struct {
		name    string
		status  string
		reason  *string
		want    string
		wantErr bool
	}{
		{"Hold tanpa alasan", "Hold", nil, "", true},
		{"Hold alasan spasi", "Hold", sp("   "), "", true},
		{"Hold beralasan (di-trim)", "Hold", sp("  Menunggu anggaran "), "Menunggu anggaran", false},
		{"Cancelled tanpa alasan boleh", "Cancelled", nil, "", false},
		{"Cancelled beralasan dicatat", "Cancelled", sp("Dibatalkan sponsor"), "Dibatalkan sponsor", false},
		{"Active tanpa alasan boleh", "Active", nil, "", false},
		{"alasan kepanjangan", "Cancelled", sp(long), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := normalizeReason(tt.status, tt.reason)
			if (len(errs) > 0) != tt.wantErr {
				t.Fatalf("errs=%v wantErr=%v", errs, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestTransitions(t *testing.T) {
	all := []string{"Active", "Hold", "Cancelled"}
	for _, from := range all {
		for _, to := range all {
			if got, want := transitions[from][to], from != to; got != want {
				t.Errorf("%s -> %s: allowed=%v want %v", from, to, got, want)
			}
		}
	}
}
