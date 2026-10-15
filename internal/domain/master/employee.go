package master

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
)

// Satu query untuk semua kolom + role (array_agg), bukan 1 query per pegawai.
const employeeSelect = `
	SELECT e.id::text, e.name,
	       d.id::text, d.name,
	       sd.id::text, sd.name,
	       e.user_id::text, e.is_active,
	       COALESCE(array_agg(r.name ORDER BY r.sort_order, r.name) FILTER (WHERE r.id IS NOT NULL), '{}'),
	       COALESCE(array_agg(r.id::text ORDER BY r.sort_order, r.name) FILTER (WHERE r.id IS NOT NULL), '{}')
	FROM employees e
	JOIN master_directorates d ON d.id = e.directorate_id
	JOIN master_subdirectorates sd ON sd.id = e.subdirectorate_id
	LEFT JOIN employee_roles er ON er.employee_id = e.id
	LEFT JOIN master_project_roles r ON r.id = er.project_role_id`

const employeeGroup = ` GROUP BY e.id, d.id, sd.id`

func scanEmployee(row pgx.CollectableRow) (Employee, error) {
	var e Employee
	err := row.Scan(&e.ID, &e.Name, &e.DirectorateID, &e.Directorate,
		&e.SubdirectorateID, &e.Subdirectorate, &e.UserID, &e.IsActive, &e.Roles, &e.RoleIDs)
	if e.Roles == nil {
		e.Roles = []string{}
	}
	if e.RoleIDs == nil {
		e.RoleIDs = []string{}
	}
	return e, err
}

func (s *Service) ListEmployees(ctx context.Context, q, role, directorateID, sort string, isActive *bool, page, pageSize int) ([]Employee, int, error) {
	if directorateID != "" && !uuidRe.MatchString(directorateID) {
		return nil, 0, apperr.Validation(map[string]string{"directorateId": "Format UUID tidak valid"})
	}

	where := []string{"TRUE"}
	args := []any{}
	if q != "" {
		args = append(args, "%"+likeEscaper.Replace(q)+"%")
		where = append(where, fmt.Sprintf("e.name ILIKE $%d", len(args)))
	}
	if directorateID != "" {
		args = append(args, directorateID)
		where = append(where, fmt.Sprintf("e.directorate_id = $%d", len(args)))
	}
	if isActive != nil {
		args = append(args, *isActive)
		where = append(where, fmt.Sprintf("e.is_active = $%d", len(args)))
	}
	if role != "" {
		args = append(args, strings.ToUpper(role))
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM employee_roles fer
			JOIN master_project_roles fr ON fr.id = fer.project_role_id
			WHERE fer.employee_id = e.id AND fr.code = $%d)`, len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM employees e WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}

	order := "e.name, e.id"
	if sort == "-name" {
		order = "e.name DESC, e.id"
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.Query(ctx, fmt.Sprintf(`%s WHERE %s%s ORDER BY %s LIMIT $%d OFFSET $%d`,
		employeeSelect, whereSQL, employeeGroup, order, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	emps, err := pgx.CollectRows(rows, scanEmployee)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if emps == nil {
		emps = []Employee{}
	}
	return emps, total, nil
}

func (s *Service) getEmployee(ctx context.Context, id string) (*Employee, error) {
	rows, err := s.db.Query(ctx, employeeSelect+` WHERE e.id = $1`+employeeGroup, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	emp, err := pgx.CollectOneRow(rows, scanEmployee)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("Data tidak ditemukan.")
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return &emp, nil
}
