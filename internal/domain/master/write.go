package master

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503" // INSERT/UPDATE: referensi nggak ada
	pgRestrictViolation   = "23001" // DELETE ditolak oleh ON DELETE RESTRICT
)

func isInUse(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		(pgErr.Code == pgRestrictViolation || pgErr.Code == pgForeignKeyViolation)
}

func decodeValue(k kind, raw json.RawMessage) (any, string) {
	if string(raw) == "null" {
		return nil, "Tidak boleh null"
	}
	switch k {
	case kString:
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return nil, "Harus berupa teks"
		}
		if s = strings.TrimSpace(s); s == "" {
			return nil, "Wajib diisi"
		}
		return s, ""
	case kBool:
		var b bool
		if json.Unmarshal(raw, &b) != nil {
			return nil, "Harus true atau false"
		}
		return b, ""
	case kInt:
		var n int
		if json.Unmarshal(raw, &n) != nil {
			return nil, "Harus berupa angka"
		}
		return n, ""
	case kUUID:
		var s string
		if json.Unmarshal(raw, &s) != nil || !uuidRe.MatchString(s) {
			return nil, "Format UUID tidak valid"
		}
		return s, ""
	}
	return nil, "Tipe tidak didukung"
}

// parse memvalidasi body terhadap whitelist field. Pure function -> gampang di-unit-test.
func (d *resourceDef) parse(body map[string]json.RawMessage, create bool) (map[string]any, map[string]string) {
	allowed := map[string]field{}
	for _, f := range d.Fields {
		if create || f.Editable {
			allowed[f.JSON] = f
		}
	}
	if !create {
		for _, f := range baseFields {
			allowed[f.JSON] = f
		}
	}

	vals := map[string]any{} // key = nama kolom
	errs := map[string]string{}
	for key, raw := range body {
		f, ok := allowed[key]
		if !ok {
			errs[key] = "Field tidak dikenal atau tidak boleh diubah"
			continue
		}
		v, msg := decodeValue(f.Kind, raw)
		if msg != "" {
			errs[key] = msg
			continue
		}
		vals[f.Col] = v
	}
	if create {
		for _, f := range d.Fields {
			if _, has := vals[f.Col]; f.Required && !has {
				if _, bad := errs[f.JSON]; !bad {
					errs[f.JSON] = "Wajib diisi"
				}
			}
		}
	}
	return vals, errs
}

func (d *resourceDef) mapWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation (index lower(name) / upper(code))
			return apperr.Duplicate(d.Label + " dengan nama/kode tersebut sudah ada.")
		case "23503": // foreign_key_violation saat INSERT = referensi nggak ada
			if d.RefField != "" {
				return apperr.Validation(map[string]string{d.RefField: "Data referensi tidak ditemukan"})
			}
		}
	}
	return apperr.Internal(err)
}

func (s *Service) CreateResource(ctx context.Context, resource string, body map[string]json.RawMessage) (map[string]any, error) {
	def, err := lookup(resource)
	if err != nil {
		return nil, err
	}
	vals, errs := def.parse(body, true)
	if len(errs) > 0 {
		return nil, apperr.Validation(errs)
	}

	cols, ph, args := []string{}, []string{}, []any{}
	for _, col := range slices.Sorted(maps.Keys(vals)) {
		args = append(args, vals[col])
		cols = append(cols, col)
		ph = append(ph, fmt.Sprintf("$%d", len(args)))
	}
	// sortOrder otomatis di paling akhir (UI belum punya kolom urutan)
	cols = append(cols, "sort_order")
	ph = append(ph, fmt.Sprintf("(SELECT COALESCE(MAX(sort_order), 0) + 1 FROM %s)", def.Table))

	var id string
	err = s.db.QueryRow(ctx, fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s) RETURNING id::text`,
		def.Table, strings.Join(cols, ", "), strings.Join(ph, ", ")), args...).Scan(&id)
	if err != nil {
		return nil, def.mapWriteErr(err)
	}
	return s.getOne(ctx, def, id)
}

func (s *Service) isProtected(ctx context.Context, def *resourceDef, col, id string) (bool, error) {
	var protected bool
	err := s.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, col, def.Table), id).Scan(&protected)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, apperr.NotFound("Data tidak ditemukan.")
	}
	if err != nil {
		return false, apperr.Internal(err)
	}
	return protected, nil
}

func (s *Service) UpdateResource(ctx context.Context, resource, id string, body map[string]json.RawMessage) (map[string]any, error) {
	def, err := lookup(resource)
	if err != nil {
		return nil, err
	}
	if !uuidRe.MatchString(id) { // tanpa ini, id ngawur = error 22P02 dari Postgres = 500
		return nil, apperr.NotFound("Data tidak ditemukan.")
	}
	vals, errs := def.parse(body, false)
	if len(errs) > 0 {
		return nil, apperr.Validation(errs)
	}
	if len(vals) == 0 {
		return nil, apperr.BadRequest("Tidak ada field yang diubah.")
	}

	if _, renaming := vals["name"]; renaming && def.ProtectEditCol != "" {
		prot, err := s.isProtected(ctx, def, def.ProtectEditCol, id)
		if err != nil {
			return nil, err
		}
		if prot {
			return nil, apperr.InvalidState("Data sistem tidak boleh diubah namanya.")
		}
	}

	sets, args := []string{}, []any{}
	for _, col := range slices.Sorted(maps.Keys(vals)) {
		args = append(args, vals[col])
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	args = append(args, id)
	tag, err := s.db.Exec(ctx, fmt.Sprintf(
		`UPDATE %s SET %s WHERE id = $%d`, def.Table, strings.Join(sets, ", "), len(args)), args...)
	if err != nil {
		return nil, def.mapWriteErr(err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("Data tidak ditemukan.")
	}
	return s.getOne(ctx, def, id)
}

func (s *Service) DeleteResource(ctx context.Context, resource, id string) error {
	def, err := lookup(resource)
	if err != nil {
		return err
	}
	if !uuidRe.MatchString(id) {
		return apperr.NotFound("Data tidak ditemukan.")
	}
	if def.ProtectDeleteCol != "" {
		prot, err := s.isProtected(ctx, def, def.ProtectDeleteCol, id)
		if err != nil {
			return err
		}
		if prot {
			return apperr.InvalidState("Data sistem tidak boleh dihapus.")
		}
	}

	tag, err := s.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, def.Table), id)
	if err != nil {
		if isInUse(err) {
			return s.inUseError(ctx, def, id)
		}
		return apperr.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("Data tidak ditemukan.")
	}
	return nil
}

// inUseError menghitung siapa yang masih memakai, buat detail IN_USE di spec §5.2.
func (s *Service) inUseError(ctx context.Context, def *resourceDef, id string) error {
	return s.inUseErr(ctx, def.Label, def.Refs, id)
}

func (s *Service) inUseErr(ctx context.Context, label string, refs []ref, id string) error {
	found := []InUseReference{}
	parts := []string{}
	for _, r := range refs {
		var n int
		if err := s.db.QueryRow(ctx,
			fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, r.Table, r.Col), id).Scan(&n); err != nil {
			return apperr.Internal(err)
		}
		if n > 0 {
			found = append(found, InUseReference{Entity: r.Entity, Count: n})
			parts = append(parts, fmt.Sprintf("%d %s", n, r.Entity))
		}
	}
	msg := label + " masih dipakai data lain."
	if len(parts) > 0 {
		msg = fmt.Sprintf("%s masih dipakai oleh %s.", label, strings.Join(parts, ", "))
	}
	return apperr.InUse(msg, InUseDetails{
		References: found,
		Suggestion: "Nonaktifkan saja (isActive=false).",
	})
}
