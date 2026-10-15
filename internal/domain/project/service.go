package project

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
)

// scopePredicate — predikat SQL scope project sesuai spec §2.4
// Inject ke semua query yang melibatkan tabel projects
// $1 = userID
const scopePredicate = `
	p.deleted_at IS NULL AND (
	  EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id
	          WHERE ur.user_id = $1 AND ro.name = 'Administrator')
	  OR p.created_by = $1
	  OR EXISTS (
	    SELECT 1 FROM project_members m
	    JOIN employees e ON e.id = m.employee_id
	    JOIN master_project_roles r ON r.id = m.project_role_id
	    WHERE m.project_id = p.id AND e.user_id = $1 AND r.code IN ('PMO','DEVELOPER')
	  )
	)
`

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// ListProjects — GET /projects
func (s *Service) ListProjects(ctx context.Context, userID, q, status, priorityID, categoryID, directorateID, sort string, page, pageSize int) ([]ProjectListItem, int, error) {
	// TODO: implementasi full query dengan filter + pagination
	// Menggunakan scopePredicate untuk membatasi akses
	_ = scopePredicate
	return []ProjectListItem{}, 0, nil
}

// GetProject — GET /projects/{id}
func (s *Service) GetProject(ctx context.Context, userID, projectID string) (*ProjectDetail, error) {
	// TODO: implementasi query detail project dengan semua relasi
	return nil, apperr.NotImplemented("TODO: GET /projects/{id}")
}

// CreateProject — POST /projects (dengan Fase 1 default, dalam 1 transaksi)
func (s *Service) CreateProject(ctx context.Context, userID string, req CreateProjectRequest) (*ProjectDetail, error) {

	// TODO: implementasi
	// 1. Validasi subdit milik direktorat
	// 2. INSERT INTO projects
	// 3. INSERT INTO phases (Fase 1 default)
	// 4. INSERT INTO audit_logs
	// 5. tx.Commit()

	return nil, apperr.NotImplemented("TODO: POST /projects")
}

// UpdateProject — PATCH /projects/{id}
func (s *Service) UpdateProject(ctx context.Context, userID, projectID string, req UpdateProjectRequest) (*ProjectDetail, error) {
	// TODO: implementasi
	// - Tolak field status dengan 422
	// - Validasi scope project
	// - Tulis audit_logs per field yang berubah
	return nil, apperr.NotImplemented("TODO: PATCH /projects/{id}")
}

// DeleteProject — DELETE /projects/{id} (soft delete)
func (s *Service) DeleteProject(ctx context.Context, userID, projectID string) error {
	// TODO: soft delete - set deleted_at = now()
	return apperr.NotImplemented("TODO: DELETE /projects/{id}")
}

// ChangeStatus — POST /projects/{id}/status
func (s *Service) ChangeStatus(ctx context.Context, userID, projectID string, req StatusChangeRequest) (map[string]any, error) {
	// TODO:
	// - Sama dengan status sekarang → 409 INVALID_STATE
	// - Ke Hold → reason wajib, INSERT project_hold_history
	// - Keluar dari Hold → tutup hold aktif (resumedAt, resumedBy)
	return nil, apperr.NotImplemented("TODO: POST /projects/{id}/status")
}
