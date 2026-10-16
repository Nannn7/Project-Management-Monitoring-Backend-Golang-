package project

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/db"
	"github.com/yourorg/pmo-agi-be/internal/validation"
)

// ownerSQL: entity -> project_id pemiliknya. Satu tempat, supaya 20 endpoint sub-resource
// nggak masing-masing nulis join sendiri (dan nggak ada yang lupa cek scope).
var ownerSQL = map[string]string{
	"phase":       `SELECT project_id::text FROM phases WHERE id = $1`,
	"member":      `SELECT project_id::text FROM project_members WHERE id = $1`,
	"task":        `SELECT project_id::text FROM tasks WHERE id = $1`,
	"environment": `SELECT ph.project_id::text FROM environments x JOIN phases ph ON ph.id = x.phase_id WHERE x.id = $1`,
	"document":    `SELECT ph.project_id::text FROM documents x JOIN phases ph ON ph.id = x.phase_id WHERE x.id = $1`,
	"migration":   `SELECT ph.project_id::text FROM phase_migrations x JOIN phases ph ON ph.id = x.phase_id WHERE x.id = $1`,
	"dependency":  `SELECT ph.project_id::text FROM phase_dependencies x JOIN phases ph ON ph.id = x.phase_id WHERE x.id = $1`,
	"approver": `SELECT ph.project_id::text FROM document_approvers x
	             JOIN documents d ON d.id = x.document_id
	             JOIN phases ph ON ph.id = d.phase_id WHERE x.id = $1`,
}

func notFound() error { return apperr.NotFound("Data tidak ditemukan.") }

// ensureProjectAccess: lolos kalau project ada, belum dihapus, dan user boleh akses (scope §2.4).
// Selain itu SELALU 404 (tidak membedakan "tidak ada" vs "dilarang").
func (s *Service) ensureProjectAccess(ctx context.Context, q db.Querier, userID, projectID string) error {
	if !validation.IsUUID(projectID) {
		return notFound()
	}
	var ok bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM projects p WHERE p.id = $2 AND `+scopePredicate+`)`,
		userID, projectID).Scan(&ok)
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		return notFound()
	}
	return nil
}

// accessViaEntity: untuk endpoint /phases/{id}, /documents/{id}, dst.
// Mengembalikan projectID pemiliknya (dibutuhkan audit log) setelah scope lolos.
func (s *Service) accessViaEntity(ctx context.Context, q db.Querier, userID, entity, id string) (string, error) {
	query, ok := ownerSQL[entity]
	if !ok {
		return "", apperr.Internal(fmt.Errorf("entity %q tidak terdaftar di ownerSQL", entity))
	}
	if !validation.IsUUID(id) {
		return "", notFound()
	}
	var projectID string
	if err := q.QueryRow(ctx, query, id).Scan(&projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", notFound()
		}
		return "", apperr.Internal(err)
	}
	if err := s.ensureProjectAccess(ctx, q, userID, projectID); err != nil {
		return "", err
	}
	return projectID, nil
}

var _ = pgx.ErrNoRows // (dipakai di atas; baris ini cukup dihapus kalau import sudah terpakai)
