package project

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/db"
	"github.com/yourorg/pmo-agi-be/internal/middleware"
	"github.com/yourorg/pmo-agi-be/internal/validation"
)

// $1 = userID, $2 = projectID (scopePredicate memakai $1 dan alias p).
const detailSQL = `
	SELECT p.id::text, p.name, p.cr_number, p.br_number,
	       d.name, p.directorate_id::text, sd.name, p.subdirectorate_id::text,
	       c.name, p.category_id::text, pr.name, p.priority_id::text,
	       p.status, v.name, p.vendor_id::text,
	       p.po_number, p.po_status, p.git_repo, p.has_migration, p.overall_progress,
	       u.id::text, u.name, p.created_at, p.updated_at
	FROM projects p
	JOIN master_directorates d ON d.id = p.directorate_id
	JOIN master_subdirectorates sd ON sd.id = p.subdirectorate_id
	JOIN master_project_categories c ON c.id = p.category_id
	JOIN master_project_priorities pr ON pr.id = p.priority_id
	LEFT JOIN master_vendors v ON v.id = p.vendor_id
	JOIN app_users u ON u.id = p.created_by
	WHERE p.id = $2 AND ` + scopePredicate

const membersSQL = `
	SELECT m.id::text, m.employee_id::text, COALESCE(e.name, m.name),
	       r.code, r.name, m.is_primary, m.is_vendor,
	       d.name, m.directorate_id::text, sd.name, m.subdirectorate_id::text
	FROM project_members m
	JOIN master_project_roles r ON r.id = m.project_role_id
	LEFT JOIN employees e ON e.id = m.employee_id
	LEFT JOIN master_directorates d ON d.id = m.directorate_id
	LEFT JOIN master_subdirectorates sd ON sd.id = m.subdirectorate_id
	WHERE m.project_id = $1
	ORDER BY r.sort_order, m.is_primary DESC, COALESCE(e.name, m.name), m.id`

const holdSQL = `
	SELECT h.id::text, to_char(h.held_at, 'YYYY-MM-DD'), h.reason, hb.name,
	       to_char(h.resumed_at, 'YYYY-MM-DD'), rb.name
	FROM project_hold_history h
	JOIN app_users hb ON hb.id = h.held_by
	LEFT JOIN app_users rb ON rb.id = h.resumed_by
	WHERE h.project_id = $1
	ORDER BY h.held_at DESC, (h.resumed_at IS NULL) DESC, h.id`

const phasesSQL = `
	SELECT id::text, name, stage, sort_order,
	       to_char(start_date, 'YYYY-MM-DD'), to_char(target_live_date, 'YYYY-MM-DD'),
	       previous_phase_id::text, scope, note,
	       progress_percent, document_progress, development_progress, testing_progress
	FROM phases WHERE project_id = $1 ORDER BY sort_order`

// loadDetail memuat satu project lengkap. Jalan di pool maupun di dalam transaksi (db.Querier).
// Sub-koleksi fase (documents, testStages, environments, migrations, dependencies)
// sementara dikembalikan [] dan diisi per slice di c5.
func (s *Service) loadDetail(ctx context.Context, q db.Querier, userID, projectID string) (*ProjectDetail, error) {
	if !validation.IsUUID(projectID) {
		return nil, notFound()
	}

	d := &ProjectDetail{}
	err := q.QueryRow(ctx, detailSQL, userID, projectID).Scan(
		&d.ID, &d.Name, &d.CRNumber, &d.BRNumber,
		&d.Directorate, &d.DirectorateID, &d.Subdirectorate, &d.SubdirectorateID,
		&d.Category, &d.CategoryID, &d.Priority, &d.PriorityID,
		&d.Status, &d.Vendor, &d.VendorID,
		&d.PONumber, &d.POStatus, &d.GitRepo, &d.HasMigration, &d.OverallProgress,
		&d.CreatedBy.ID, &d.CreatedBy.Name, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound() // tidak ada ATAU dilarang: sama persis
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}

	// --- tim ---
	type memberRow struct {
		code string
		m    Member
	}
	rows, err := q.Query(ctx, membersSQL, projectID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	members, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (memberRow, error) {
		var r memberRow
		err := row.Scan(&r.m.ID, &r.m.EmployeeID, &r.m.Name, &r.code, &r.m.Role,
			&r.m.IsPrimary, &r.m.IsVendor,
			&r.m.Directorate, &r.m.DirectorateID, &r.m.Subdirectorate, &r.m.SubdirectorateID)
		return r, err
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	d.PMO, d.Developers, d.Users = []Member{}, []Member{}, []Member{}
	for _, r := range members {
		switch r.code {
		case "PMO":
			d.PMO = append(d.PMO, r.m)
		case "DEVELOPER":
			d.Developers = append(d.Developers, r.m)
		case "USER":
			d.Users = append(d.Users, r.m)
		} // QA / SECURITY belum punya slot di kontrak FE (spec §7 cuma 3 array)
	}

	// --- riwayat hold ---
	rows, err = q.Query(ctx, holdSQL, projectID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	d.HoldHistory, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (HoldRecord, error) {
		var h HoldRecord
		err := row.Scan(&h.ID, &h.HeldAt, &h.Reason, &h.HeldBy, &h.ResumedAt, &h.ResumedBy)
		return h, err
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if d.HoldHistory == nil {
		d.HoldHistory = []HoldRecord{}
	}

	// --- fase ---
	rows, err = q.Query(ctx, phasesSQL, projectID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	d.Phases, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Phase, error) {
		var ph Phase
		err := row.Scan(&ph.ID, &ph.Name, &ph.Stage, &ph.SortOrder, &ph.StartDate, &ph.TargetLiveDate,
			&ph.PreviousPhaseID, &ph.Scope, &ph.Note,
			&ph.ProgressPercent, &ph.DocumentProgress, &ph.DevelopmentProgress, &ph.TestingProgress)
		if ph.Scope == nil {
			ph.Scope = []string{}
		}
		ph.Documents = []Document{}
		ph.TestStages = []TestStage{}
		ph.Environments = []Environment{}
		ph.Migrations = []Migration{}
		ph.Dependencies = []Dependency{}
		return ph, err
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if d.Phases == nil {
		d.Phases = []Phase{}
	}
	return d, nil
}

// fillPermissions: canEdit = punya projects.edit (scope sudah lolos karena loadDetail berhasil);
// canDelete = punya projects.delete. Dipanggil setelah loadDetail, di luar transaksi.
func (s *Service) fillPermissions(ctx context.Context, userID string, d *ProjectDetail) error {
	edit, err := middleware.HasPermission(ctx, s.db, userID, "projects", "edit")
	if err != nil {
		return apperr.Internal(err)
	}
	del, err := middleware.HasPermission(ctx, s.db, userID, "projects", "delete")
	if err != nil {
		return apperr.Internal(err)
	}
	d.Permissions = ProjectPerms{CanEdit: edit, CanDelete: del}
	return nil
}

// GetProject: GET /projects/{id}
func (s *Service) GetProject(ctx context.Context, userID, projectID string) (*ProjectDetail, error) {
	d, err := s.loadDetail(ctx, s.db, userID, projectID)
	if err != nil {
		return nil, err
	}
	if err := s.fillPermissions(ctx, userID, d); err != nil {
		return nil, err
	}
	return d, nil
}
