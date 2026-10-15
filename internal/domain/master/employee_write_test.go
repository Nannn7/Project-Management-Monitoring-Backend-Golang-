package master

import (
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
)

func TestDedupe(t *testing.T) {
	got := dedupe([]string{"AAA", "bbb", "aaa", "BBB", "ccc"})
	want := []string{"aaa", "bbb", "ccc"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMapEmployeeWriteErr(t *testing.T) {
	tests := []struct{ constraint, field string }{
		{"employees_subdirectorate_id_directorate_id_fkey", "subdirectorateId"},
		{"employees_directorate_id_fkey", "directorateId"},
		{"employee_roles_project_role_id_fkey", "roleIds"},
	}
	for _, tt := range tests {
		err := mapEmployeeWriteErr(&pgconn.PgError{Code: "23503", ConstraintName: tt.constraint})
		var ae *apperr.Error
		if !errors.As(err, &ae) || ae.Kind != apperr.KindValidation {
			t.Fatalf("%s: bukan validation error: %v", tt.constraint, err)
		}
		if _, ok := ae.Fields[tt.field]; !ok {
			t.Errorf("%s: harusnya error di %q, dapat %v", tt.constraint, tt.field, ae.Fields)
		}
	}
	// apperr yang sudah jadi (mis. NotFound dari dalam transaksi) harus lewat tanpa diubah
	nf := apperr.NotFound("x")
	if got := mapEmployeeWriteErr(nf); got != error(nf) {
		t.Errorf("apperr harus diteruskan apa adanya")
	}
}
