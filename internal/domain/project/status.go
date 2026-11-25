package project

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/audit"
	"github.com/yourorg/pmo-agi-be/internal/db"
	"github.com/yourorg/pmo-agi-be/internal/validation"
)

// Tabel transisi sebagai DATA. Sesuai jawaban FE: Cancelled bisa dilanjutkan, jadi semua boleh
// kecuali ke status yang sama. Mau mengunci Cancelled? Hapus dua baris di bawah "Cancelled".
var transitions = map[string]map[string]bool{
	"Active":    {"Hold": true, "Cancelled": true},
	"Hold":      {"Active": true, "Cancelled": true},
	"Cancelled": {"Active": true, "Hold": true},
}

// normalizeReason: Hold wajib beralasan (spec §8.1). Alasan untuk status lain boleh, dicatat di audit.
// Fungsi murni.
func normalizeReason(status string, reason *string) (string, map[string]string) {
	r := ""
	if reason != nil {
		r = strings.TrimSpace(*reason)
	}
	if status == "Hold" && r == "" {
		return "", map[string]string{"reason": "Wajib diisi saat status Hold"}
	}
	if utf8.RuneCountInString(r) > 500 {
		return "", map[string]string{"reason": "Maksimal 500 karakter"}
	}
	return r, nil
}

// ChangeStatus: POST /projects/{id}/status
func (s *Service) ChangeStatus(ctx context.Context, userID, projectID string, req StatusChangeRequest) (*StatusChangeResult, error) {
	if !validation.IsUUID(projectID) {
		return nil, notFound()
	}
	reason, errs := normalizeReason(req.Status, req.Reason)
	if len(errs) > 0 {
		return nil, apperr.Validation(errs) // validasi body dulu (422), baru state (409)
	}

	var result *StatusChangeResult
	err := db.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		_, cur, err := s.lockProject(ctx, tx, userID, projectID)
		if err != nil {
			return err
		}
		if cur == req.Status {
			return apperr.InvalidState(fmt.Sprintf("Project sudah berstatus %s.", cur))
		}
		if !transitions[cur][req.Status] {
			return apperr.InvalidState(fmt.Sprintf("Status tidak bisa diubah dari %s ke %s.", cur, req.Status))
		}

		// Keluar dari Hold (ke status APA PUN, termasuk Cancelled) menutup hold aktif.
		// Sebelum Hold baru juga dijalankan sebagai pengaman, supaya unique index ux_hold_active
		// nggak pernah nabrak walau ada data lama yang nggak konsisten.
		if cur == "Hold" || req.Status == "Hold" {
			if _, err := tx.Exec(ctx, `
				UPDATE project_hold_history
				SET resumed_at = (now() AT TIME ZONE $3::text)::date, resumed_by = $2
				WHERE project_id = $1 AND resumed_at IS NULL`,
				projectID, userID, appTimeZone); err != nil {
				return err
			}
		}
		if req.Status == "Hold" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO project_hold_history (project_id, held_at, reason, held_by)
				VALUES ($1, (now() AT TIME ZONE $4::text)::date, $2, $3)`,
				projectID, reason, userID, appTimeZone); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE projects SET status = $1 WHERE id = $2`, req.Status, projectID); err != nil {
			return err
		}

		from, to := cur, req.Status
		entries := []audit.Entry{{
			ProjectID: projectID, EntityType: "project", EntityID: projectID, Label: "Project",
			Field: "Status", Old: &from, New: &to, ChangedBy: userID,
		}}
		if reason != "" {
			r := reason
			entries = append(entries, audit.Entry{
				ProjectID: projectID, EntityType: "project", EntityID: projectID, Label: "Project",
				Field: "Alasan perubahan status", New: &r, ChangedBy: userID,
			})
		}
		if err := audit.Record(ctx, tx, entries...); err != nil {
			return err
		}

		holds, err := s.loadHoldHistory(ctx, tx, projectID)
		if err != nil {
			return err
		}
		result = &StatusChangeResult{ID: projectID, Status: req.Status, HoldHistory: holds}
		return nil
	})
	if err != nil {
		return nil, mapProjectWriteErr(err)
	}
	return result, nil
}
