package audit

import (
	"context"
	"fmt"
	"reflect"

	"github.com/yourorg/pmo-agi-be/internal/db"
)

type Entry struct {
	ProjectID  string  // "" = bukan milik project (mis. perubahan master)
	EntityType string  // project | phase | member | document | approver | environment | migration | dependency | task
	EntityID   string  // "" kalau nggak relevan
	Label      string  // teks di tab Riwayat: "Project", "Fase 1", ...
	Field      string  // label yang tampil: "Status", "Nama", ...
	Old, New   *string // nil = kosong/NULL
	ChangedBy  string  // app_users.id
}

// Record menulis satu baris per entry. Panggil dengan tx yang sama dengan perubahan datanya.
func Record(ctx context.Context, q db.Querier, entries ...Entry) error {
	for _, e := range entries {
		_, err := q.Exec(ctx, `
			INSERT INTO audit_logs
			  (project_id, entity_type, entity_id, entity_label, field, old_value, new_value, changed_by, changed_at)
			VALUES (NULLIF($1,'')::uuid, $2, NULLIF($3,'')::uuid, $4, $5, $6, $7, $8, clock_timestamp())`,
			e.ProjectID, e.EntityType, e.EntityID, e.Label, e.Field, e.Old, e.New, e.ChangedBy)
		if err != nil {
			return fmt.Errorf("audit: %w", err)
		}
	}
	return nil
}

// Diff mengumpulkan perubahan field. Add hanya mencatat kalau nilainya benar-benar beda.
type Diff struct{ changes []Entry }

func (d *Diff) Add(field string, oldV, newV any) {
	o, n := str(oldV), str(newV)
	if (o == nil && n == nil) || (o != nil && n != nil && *o == *n) {
		return
	}
	d.changes = append(d.changes, Entry{Field: field, Old: o, New: n})
}

func (d *Diff) Empty() bool { return len(d.changes) == 0 }

// Entries melengkapi tiap perubahan dengan identitas entity (project, user, dst).
func (d *Diff) Entries(base Entry) []Entry {
	out := make([]Entry, len(d.changes))
	for i, c := range d.changes {
		e := base
		e.Field, e.Old, e.New = c.Field, c.Old, c.New
		out[i] = e
	}
	return out
}

// str menormalkan nilai apa pun (string, *string, bool, int, ...) ke *string; nil -> nil.
// Tanggal/waktu: format dulu di pemanggil, jangan andalkan fmt.Sprint(time.Time).
func str(v any) *string {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	s := fmt.Sprint(rv.Interface())
	return &s
}
