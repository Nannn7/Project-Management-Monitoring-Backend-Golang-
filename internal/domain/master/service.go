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

// --- Private helpers ---

func collect[T any](ctx context.Context, db *pgxpool.Pool, sql string, scan func(pgx.CollectableRow) (T, error)) ([]T, error) {
	rows, err := db.Query(ctx, sql)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	items, err := pgx.CollectRows(rows, scan)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if items == nil {
		items = []T{}
	}
	return items, nil
}

// table HANYA boleh diisi konstanta dari GetLookups, bukan input user.
func (s *Service) listGeneric(ctx context.Context, table string) ([]MasterItem, error) {
	return collect(ctx, s.db, fmt.Sprintf(`
		SELECT id, name, sort_order, is_active, created_at, updated_at
		FROM %s WHERE is_active = true ORDER BY sort_order, name`, table),
		func(row pgx.CollectableRow) (MasterItem, error) {
			var it MasterItem
			err := row.Scan(&it.ID, &it.Name, &it.SortOrder, &it.IsActive, &it.CreatedAt, &it.UpdatedAt)
			return it, err
		})
}

func (s *Service) listSubdirectorates(ctx context.Context) ([]Subdirectorate, error) {
	return collect(ctx, s.db, `
		SELECT sd.id, sd.name, sd.sort_order, sd.is_active, sd.created_at, sd.updated_at, d.id, d.name
		FROM master_subdirectorates sd
		JOIN master_directorates d ON d.id = sd.directorate_id
		WHERE sd.is_active = true ORDER BY d.name, sd.sort_order, sd.name`,
		func(row pgx.CollectableRow) (Subdirectorate, error) {
			var it Subdirectorate
			err := row.Scan(&it.ID, &it.Name, &it.SortOrder, &it.IsActive, &it.CreatedAt, &it.UpdatedAt,
				&it.DirectorateID, &it.Directorate)
			return it, err
		})
}

func (s *Service) listProjectRoles(ctx context.Context) ([]ProjectRole, error) {
	return collect(ctx, s.db, `
		SELECT id, code, name, is_system, sort_order, is_active, created_at, updated_at
		FROM master_project_roles WHERE is_active = true ORDER BY sort_order`,
		func(row pgx.CollectableRow) (ProjectRole, error) {
			var it ProjectRole
			err := row.Scan(&it.ID, &it.Code, &it.Name, &it.IsSystem, &it.SortOrder, &it.IsActive, &it.CreatedAt, &it.UpdatedAt)
			return it, err
		})
}

func (s *Service) listDependencySystems(ctx context.Context) ([]DependencySystem, error) {
	return collect(ctx, s.db, `
		SELECT id, name, is_other, sort_order, is_active, created_at, updated_at
		FROM master_dependency_systems WHERE is_active = true ORDER BY sort_order, name`,
		func(row pgx.CollectableRow) (DependencySystem, error) {
			var it DependencySystem
			err := row.Scan(&it.ID, &it.Name, &it.IsOther, &it.SortOrder, &it.IsActive, &it.CreatedAt, &it.UpdatedAt)
			return it, err
		})
}

func (s *Service) listDocumentTypes(ctx context.Context) ([]DocumentType, error) {
	return collect(ctx, s.db, `
		SELECT id, code, description, sort_order, is_active, created_at, updated_at
		FROM master_document_types WHERE is_active = true ORDER BY sort_order, code`,
		func(row pgx.CollectableRow) (DocumentType, error) {
			var it DocumentType
			err := row.Scan(&it.ID, &it.Code, &it.Description, &it.SortOrder, &it.IsActive, &it.CreatedAt, &it.UpdatedAt)
			return it, err
		})
}

func (s *Service) listTestTypes(ctx context.Context) ([]TestType, error) {
	return collect(ctx, s.db, `
		SELECT id, code, requires_previous, allow_parallel, sort_order, is_active, created_at, updated_at
		FROM master_test_types WHERE is_active = true ORDER BY sort_order, code`,
		func(row pgx.CollectableRow) (TestType, error) {
			var it TestType
			err := row.Scan(&it.ID, &it.Code, &it.RequiresPrevious, &it.AllowParallel, &it.SortOrder, &it.IsActive, &it.CreatedAt, &it.UpdatedAt)
			return it, err
		})
}
