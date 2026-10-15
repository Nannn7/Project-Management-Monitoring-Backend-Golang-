package master

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/db"
)

func dedupe(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(id)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// roleIDs diganti utuh (spec §5.5). Dipanggil di dalam transaksi.
func replaceEmployeeRoles(ctx context.Context, tx pgx.Tx, employeeID string, roleIDs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM employee_roles WHERE employee_id = $1`, employeeID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO employee_roles (employee_id, project_role_id)
		SELECT $1, x::uuid FROM unnest($2::text[]) AS x`, employeeID, roleIDs)
	return err
}

// FK komposit subdit-direktorat dkk menolak data ngawur; kita terjemahkan ke 422 yang spesifik.
// Pencocokan lewat nama constraint (default Postgres: tabel_kolom_fkey).
func mapEmployeeWriteErr(err error) error {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		switch c := pgErr.ConstraintName; {
		case strings.Contains(c, "subdirectorate"):
			return apperr.Validation(map[string]string{"subdirectorateId": "Subdit bukan milik direktorat ini"})
		case strings.Contains(c, "directorate"):
			return apperr.Validation(map[string]string{"directorateId": "Direktorat tidak ditemukan"})
		case strings.Contains(c, "project_role"):
			return apperr.Validation(map[string]string{"roleIds": "Ada role yang tidak ditemukan"})
		}
	}
	return apperr.Internal(err)
}

func (s *Service) CreateEmployee(ctx context.Context, req CreateEmployeeRequest) (*Employee, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperr.Validation(map[string]string{"name": "Wajib diisi"})
	}

	var id string
	err := db.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO employees (name, directorate_id, subdirectorate_id)
			VALUES ($1, $2, $3) RETURNING id::text`,
			name, req.DirectorateID, req.SubdirectorateID).Scan(&id); err != nil {
			return err
		}
		return replaceEmployeeRoles(ctx, tx, id, dedupe(req.RoleIDs))
	})
	if err != nil {
		return nil, mapEmployeeWriteErr(err)
	}
	return s.getEmployee(ctx, id)
}

func (s *Service) UpdateEmployee(ctx context.Context, id string, req UpdateEmployeeRequest) (*Employee, error) {
	if !uuidRe.MatchString(id) {
		return nil, apperr.NotFound("Data tidak ditemukan.")
	}

	sets, args := []string{}, []any{}
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, apperr.Validation(map[string]string{"name": "Wajib diisi"})
		}
		add("name", name)
	}
	if req.DirectorateID != nil {
		add("directorate_id", *req.DirectorateID)
	}
	if req.SubdirectorateID != nil {
		add("subdirectorate_id", *req.SubdirectorateID)
	}
	if req.IsActive != nil {
		add("is_active", *req.IsActive)
	}
	if len(sets) == 0 && req.RoleIDs == nil {
		return nil, apperr.BadRequest("Tidak ada field yang diubah.")
	}

	err := db.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		// kunci baris: sekaligus cek ada/tidak, dan nyerialin edit role yang barengan
		var existing string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM employees WHERE id = $1 FOR UPDATE`, id).Scan(&existing); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound("Data tidak ditemukan.")
			}
			return err
		}
		if len(sets) > 0 {
			upd := append(slices.Clone(args), id)
			if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE employees SET %s WHERE id = $%d`,
				strings.Join(sets, ", "), len(upd)), upd...); err != nil {
				return err
			}
		}
		if req.RoleIDs != nil {
			return replaceEmployeeRoles(ctx, tx, id, dedupe(req.RoleIDs))
		}
		return nil
	})
	if err != nil {
		return nil, mapEmployeeWriteErr(err)
	}
	return s.getEmployee(ctx, id)
}

func (s *Service) DeleteEmployee(ctx context.Context, id string) error {
	if !uuidRe.MatchString(id) {
		return apperr.NotFound("Data tidak ditemukan.")
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM employees WHERE id = $1`, id)
	if err != nil {
		if isInUse(err) {
			return s.inUseErr(ctx, "Pegawai", []ref{
				{"project_members", "employee_id", "project_members"},
				{"document_approvers", "employee_id", "approvers"},
			}, id)
		}
		return apperr.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("Data tidak ditemukan.")
	}
	return nil
}
