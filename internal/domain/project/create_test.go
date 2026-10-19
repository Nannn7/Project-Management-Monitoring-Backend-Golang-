package project

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
)

func TestMapProjectWriteErr(t *testing.T) {
	tests := []struct{ constraint, field string }{
		{"projects_subdirectorate_id_directorate_id_fkey", "subdirectorateId"},
		{"projects_directorate_id_fkey", "directorateId"},
		{"projects_category_id_fkey", "categoryId"},
		{"projects_priority_id_fkey", "priorityId"},
		{"projects_vendor_id_fkey", "vendorId"},
	}
	for _, tt := range tests {
		err := mapProjectWriteErr(&pgconn.PgError{Code: "23503", ConstraintName: tt.constraint})
		var ae *apperr.Error
		if !errors.As(err, &ae) || ae.Kind != apperr.KindValidation {
			t.Fatalf("%s: bukan validation error: %v", tt.constraint, err)
		}
		if _, ok := ae.Fields[tt.field]; !ok {
			t.Errorf("%s: harusnya error di %q, dapat %v", tt.constraint, tt.field, ae.Fields)
		}
	}

	// FK lain (mis. created_by) = bug server, bukan salah client -> 500
	err := mapProjectWriteErr(&pgconn.PgError{Code: "23503", ConstraintName: "projects_created_by_fkey"})
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Kind != apperr.KindInternal {
		t.Errorf("FK tak dikenal harusnya Internal, dapat %v", err)
	}

	// apperr yang sudah jadi lewat tanpa diubah
	nf := apperr.NotFound("x")
	if got := mapProjectWriteErr(nf); got != error(nf) {
		t.Error("apperr harus diteruskan apa adanya")
	}
}
