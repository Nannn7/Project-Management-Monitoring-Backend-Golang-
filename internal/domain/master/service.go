package master

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func lookup(resource string) (*resourceDef, error) {
	def, ok := registry[resource]
	if !ok {
		return nil, apperr.NotFound("Resource tidak dikenal.")
	}
	return &def, nil
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// GetLookups — GET /lookups, semua dropdown sekaligus (is_active = true)
func (s *Service) GetLookups(ctx context.Context) (*LookupsResponse, error) {
	resp := &LookupsResponse{}
	var err error

	resp.Directorates, err = s.listGeneric(ctx, "master_directorates")
	if err != nil {
		return nil, err
	}
	resp.Subdirectorates, err = s.listSubdirectorates(ctx)
	if err != nil {
		return nil, err
	}
	resp.Vendors, err = s.listGeneric(ctx, "master_vendors")
	if err != nil {
		return nil, err
	}
	resp.ProjectCategories, err = s.listGeneric(ctx, "master_project_categories")
	if err != nil {
		return nil, err
	}
	resp.ProjectPriorities, err = s.listGeneric(ctx, "master_project_priorities")
	if err != nil {
		return nil, err
	}
	resp.ApprovalLevels, err = s.listGeneric(ctx, "master_approval_levels")
	if err != nil {
		return nil, err
	}
	resp.ProjectRoles, err = s.listProjectRoles(ctx)
	if err != nil {
		return nil, err
	}
	resp.TaskCategories, err = s.listGeneric(ctx, "master_task_categories")
	if err != nil {
		return nil, err
	}
	resp.MeetingCategories, err = s.listGeneric(ctx, "master_meeting_categories")
	if err != nil {
		return nil, err
	}
	resp.DependencySystems, err = s.listDependencySystems(ctx)
	if err != nil {
		return nil, err
	}
	resp.DocumentTypes, err = s.listDocumentTypes(ctx)
	if err != nil {
		return nil, err
	}
	resp.TestTypes, err = s.listTestTypes(ctx)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ListResource — GET /master/{resource}
func (s *Service) ListResource(ctx context.Context, resource, q, sort string, page, pageSize int, isActive *bool) ([]map[string]any, int, error) {
	def, err := lookup(resource)
	if err != nil {
		return nil, 0, err
	}

	where := []string{"TRUE"}
	args := []any{}

	if q != "" {
		args = append(args, "%"+likeEscaper.Replace(q)+"%")
		conds := make([]string, len(def.SearchCols))
		for i, c := range def.SearchCols {
			conds[i] = fmt.Sprintf("t.%s ILIKE $%d", c, len(args))
		}
		where = append(where, "("+strings.Join(conds, " OR ")+")")
	}
	if isActive != nil {
		args = append(args, *isActive)
		where = append(where, fmt.Sprintf("t.is_active = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s`, def.from(), whereSQL), args...,
	).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.Query(ctx, fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		def.Select, def.from(), whereSQL, def.orderBy(sort), len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToMap) // nutup rows + cek Scan + rows.Err sekaligus
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if items == nil {
		items = []map[string]any{}
	}
	return items, total, nil
}

func (s *Service) getOne(ctx context.Context, def *resourceDef, id string) (map[string]any, error) {
	rows, err := s.db.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM %s WHERE t.id = $1`, def.Select, def.from()), id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	item, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("Data tidak ditemukan.")
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return item, nil
}

// ListEmployees — GET /employees
func (s *Service) ListEmployees(ctx context.Context, q, role, directorateID string, isActive *bool, page, pageSize int) ([]Employee, int, error) {
	whereClause := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if q != "" {
		whereClause += fmt.Sprintf(" AND e.name ILIKE $%d", argIdx)
		args = append(args, "%"+q+"%")
		argIdx++
	}
	if directorateID != "" {
		whereClause += fmt.Sprintf(" AND e.directorate_id = $%d", argIdx)
		args = append(args, directorateID)
		argIdx++
	}
	if isActive != nil {
		whereClause += fmt.Sprintf(" AND e.is_active = $%d", argIdx)
		args = append(args, *isActive)
		argIdx++
	}
	if role != "" {
		whereClause += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM employee_roles er
			JOIN master_project_roles r ON r.id = er.project_role_id
			WHERE er.employee_id = e.id AND r.code = $%d
		)`, argIdx)
		args = append(args, role)
		argIdx++
	}

	var total int
	s.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM employees e %s
	`, whereClause), args...).Scan(&total)

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	rows, err := s.db.Query(ctx, fmt.Sprintf(`
		SELECT e.id, e.name,
		       d.id, d.name,
		       sd.id, sd.name,
		       e.user_id, e.is_active
		FROM employees e
		JOIN master_directorates d ON d.id = e.directorate_id
		JOIN master_subdirectorates sd ON sd.id = e.subdirectorate_id
		%s
		ORDER BY e.name
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var emp Employee
		rows.Scan(
			&emp.ID, &emp.Name,
			&emp.DirectorateID, &emp.Directorate,
			&emp.SubdirectorateID, &emp.Subdirectorate,
			&emp.UserID, &emp.IsActive,
		)
		// Ambil roles
		emp.Roles, emp.RoleIDs = s.getEmployeeRoles(ctx, emp.ID)
		employees = append(employees, emp)
	}
	if employees == nil {
		employees = []Employee{}
	}
	return employees, total, nil
}

// --- Private helpers ---

func (s *Service) listGeneric(ctx context.Context, table string) ([]MasterItem, error) {
	rows, err := s.db.Query(ctx, fmt.Sprintf(`
		SELECT id, name, sort_order, is_active, created_at, updated_at
		FROM %s WHERE is_active = true ORDER BY sort_order, name
	`, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MasterItem
	for rows.Next() {
		var item MasterItem
		rows.Scan(&item.ID, &item.Name, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
		items = append(items, item)
	}
	if items == nil {
		items = []MasterItem{}
	}
	return items, nil
}

func (s *Service) listSubdirectorates(ctx context.Context) ([]Subdirectorate, error) {
	rows, err := s.db.Query(ctx, `
		SELECT sd.id, sd.name, sd.sort_order, sd.is_active, sd.created_at, sd.updated_at,
		       d.id, d.name
		FROM master_subdirectorates sd
		JOIN master_directorates d ON d.id = sd.directorate_id
		WHERE sd.is_active = true ORDER BY d.name, sd.sort_order, sd.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Subdirectorate
	for rows.Next() {
		var item Subdirectorate
		rows.Scan(&item.ID, &item.Name, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
			&item.DirectorateID, &item.Directorate)
		items = append(items, item)
	}
	if items == nil {
		items = []Subdirectorate{}
	}
	return items, nil
}

func (s *Service) listProjectRoles(ctx context.Context) ([]ProjectRole, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, code, name, is_system, sort_order, is_active, created_at, updated_at
		FROM master_project_roles WHERE is_active = true ORDER BY sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ProjectRole
	for rows.Next() {
		var item ProjectRole
		rows.Scan(&item.ID, &item.Code, &item.Name, &item.IsSystem, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
		items = append(items, item)
	}
	if items == nil {
		items = []ProjectRole{}
	}
	return items, nil
}

func (s *Service) listDependencySystems(ctx context.Context) ([]DependencySystem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, is_other, sort_order, is_active, created_at, updated_at
		FROM master_dependency_systems WHERE is_active = true ORDER BY sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DependencySystem
	for rows.Next() {
		var item DependencySystem
		rows.Scan(&item.ID, &item.Name, &item.IsOther, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
		items = append(items, item)
	}
	if items == nil {
		items = []DependencySystem{}
	}
	return items, nil
}

func (s *Service) listDocumentTypes(ctx context.Context) ([]DocumentType, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, code, description, sort_order, is_active, created_at, updated_at
		FROM master_document_types WHERE is_active = true ORDER BY sort_order, code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DocumentType
	for rows.Next() {
		var item DocumentType
		rows.Scan(&item.ID, &item.Code, &item.Description, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
		items = append(items, item)
	}
	if items == nil {
		items = []DocumentType{}
	}
	return items, nil
}

func (s *Service) listTestTypes(ctx context.Context) ([]TestType, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, code, requires_previous, allow_parallel, sort_order, is_active, created_at, updated_at
		FROM master_test_types WHERE is_active = true ORDER BY sort_order, code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TestType
	for rows.Next() {
		var item TestType
		rows.Scan(&item.ID, &item.Code, &item.RequiresPrevious, &item.AllowParallel, &item.SortOrder, &item.IsActive, &item.CreatedAt, &item.UpdatedAt)
		items = append(items, item)
	}
	if items == nil {
		items = []TestType{}
	}
	return items, nil
}

func (s *Service) getEmployeeRoles(ctx context.Context, empID string) ([]string, []string) {
	rows, _ := s.db.Query(ctx, `
		SELECT r.name, r.id FROM employee_roles er
		JOIN master_project_roles r ON r.id = er.project_role_id
		WHERE er.employee_id = $1
	`, empID)
	defer rows.Close()
	var names, ids []string
	for rows.Next() {
		var name, id string
		rows.Scan(&name, &id)
		names = append(names, name)
		ids = append(ids, id)
	}
	if names == nil {
		names = []string{}
		ids = []string{}
	}
	return names, ids
}
