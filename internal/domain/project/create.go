package project

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/audit"
	"github.com/yourorg/pmo-agi-be/internal/db"
)

// "Hari ini" untuk Fase 1 dihitung di zona waktu aplikasi, bukan zona server DB
// (tanpa ini, jam 03:00 WIB tanggalnya masih kemarin kalau DB di UTC).
const appTimeZone = "Asia/Jakarta"

const insertProjectSQL = `
	INSERT INTO projects (name, directorate_id, subdirectorate_id, category_id, priority_id, vendor_id, created_by)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id::text`

const insertFirstPhaseSQL = `
	INSERT INTO phases (project_id, name, stage, sort_order, start_date, target_live_date)
	SELECT $1::uuid, 'Fase 1', 'Dokumen', 1, t.d, t.d + 90
	FROM (SELECT (now() AT TIME ZONE $2::text)::date AS d) t
	RETURNING id::text`

// Pembuat otomatis jadi PMO primary KALAU dia punya pegawai terhubung (employees.user_id)
// yang punya role PMO. Kalau tidak (mis. akun admin), tidak ada baris dan itu bukan error:
// aksesnya tetap lewat projects.created_by.
const addCreatorAsPMOSQL = `
	WITH ins AS (
		INSERT INTO project_members
		  (project_id, project_role_id, employee_id, is_primary, is_vendor, directorate_id, subdirectorate_id)
		SELECT $1::uuid, r.id, e.id, true, false, e.directorate_id, e.subdirectorate_id
		FROM employees e
		JOIN employee_roles er ON er.employee_id = e.id
		JOIN master_project_roles r ON r.id = er.project_role_id AND r.code = 'PMO'
		WHERE e.user_id = $2::uuid AND e.is_active
		LIMIT 1
		RETURNING employee_id
	)
	SELECT e.name FROM ins JOIN employees e ON e.id = ins.employee_id`

func (s *Service) CreateProject(ctx context.Context, userID string, req CreateProjectRequest) (*ProjectDetail, error) {
	var detail *ProjectDetail

	err := db.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		var projectID, phaseID string

		// 1. project dulu: kalau referensi (subdit/kategori/dst) salah, gagal paling awal
		if err := tx.QueryRow(ctx, insertProjectSQL,
			req.Name, req.DirectorateID, req.SubdirectorateID, req.CategoryID, req.PriorityID,
			req.VendorID, userID).Scan(&projectID); err != nil {
			return err
		}
		// 2. Fase 1 default (mulai hari ini, target +90 hari)
		if err := tx.QueryRow(ctx, insertFirstPhaseSQL, projectID, appTimeZone).Scan(&phaseID); err != nil {
			return err
		}
		// 3. pembuat otomatis PMO (best effort)
		var pmoName string
		pmoAdded := true
		if err := tx.QueryRow(ctx, addCreatorAsPMOSQL, projectID, userID).Scan(&pmoName); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			pmoAdded = false
		}

		// 4. audit, di transaksi yang sama
		name, phaseLabel := req.Name, "Fase 1 (Dokumen)"
		entries := []audit.Entry{
			{ProjectID: projectID, EntityType: "project", EntityID: projectID, Label: "Project",
				Field: "Dibuat", New: &name, ChangedBy: userID},
			{ProjectID: projectID, EntityType: "phase", EntityID: phaseID, Label: "Fase 1",
				Field: "Dibuat", New: &phaseLabel, ChangedBy: userID},
		}
		if pmoAdded {
			who := pmoName + " (PMO)"
			entries = append(entries, audit.Entry{ProjectID: projectID, EntityType: "member", Label: "Tim",
				Field: "Anggota ditambahkan", New: &who, ChangedBy: userID})
		}
		if err := audit.Record(ctx, tx, entries...); err != nil {
			return err
		}

		// 5. baca balik di transaksi yang sama (creator lolos scope lewat created_by)
		d, err := s.loadDetail(ctx, tx, userID, projectID)
		if err != nil {
			return err
		}
		detail = d
		return nil
	})
	if err != nil {
		return nil, mapProjectWriteErr(err)
	}

	if err := s.fillPermissions(ctx, userID, detail); err != nil {
		return nil, err
	}
	return detail, nil
}

// mapProjectWriteErr menerjemahkan pelanggaran FK ke 422 per field (spec §6 & §2.3).
// FK komposit (subdirectorate_id, directorate_id) -> "Subdit bukan milik direktorat ini".
func mapProjectWriteErr(err error) error {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return err // sudah apperr (mis. dari loadDetail): teruskan apa adanya
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation saat INSERT
		c := pgErr.ConstraintName
		switch {
		case strings.Contains(c, "subdirectorate"): // cek sebelum "directorate": nama FK komposit memuat keduanya
			return apperr.Validation(map[string]string{"subdirectorateId": "Subdit bukan milik direktorat ini"})
		case strings.Contains(c, "directorate"):
			return apperr.Validation(map[string]string{"directorateId": "Direktorat tidak ditemukan"})
		case strings.Contains(c, "category"):
			return apperr.Validation(map[string]string{"categoryId": "Kategori tidak ditemukan"})
		case strings.Contains(c, "priority"):
			return apperr.Validation(map[string]string{"priorityId": "Priority tidak ditemukan"})
		case strings.Contains(c, "vendor"):
			return apperr.Validation(map[string]string{"vendorId": "Vendor tidak ditemukan"})
		}
	}
	return apperr.Internal(err)
}
