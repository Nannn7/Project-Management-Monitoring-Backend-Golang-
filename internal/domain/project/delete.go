package project

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/yourorg/pmo-agi-be/internal/audit"
	"github.com/yourorg/pmo-agi-be/internal/db"
	"github.com/yourorg/pmo-agi-be/internal/validation"
)

// DeleteProject: DELETE /projects/{id}. Soft delete: data dan riwayat tetap utuh.
// Nomor CR/BR otomatis bebas dipakai lagi (unique index hanya berlaku untuk deleted_at IS NULL).
func (s *Service) DeleteProject(ctx context.Context, userID, projectID string) error {
	if !validation.IsUUID(projectID) {
		return notFound()
	}
	err := db.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		name, _, err := s.lockProject(ctx, tx, userID, projectID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE projects SET deleted_at = now() WHERE id = $1`, projectID); err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Entry{
			ProjectID: projectID, EntityType: "project", EntityID: projectID, Label: "Project",
			Field: "Dihapus", Old: &name, ChangedBy: userID,
		})
	})
	if err != nil {
		return mapProjectWriteErr(err)
	}
	return nil
}
