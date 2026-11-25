package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/audit"
	"github.com/yourorg/pmo-agi-be/internal/db"
	"github.com/yourorg/pmo-agi-be/internal/validation"
)

type patchKind int

const (
	pString patchKind = iota
	pUUID
	pEnum
	pBool
)

// patchField: satu-satunya tempat yang mendefinisikan field mana yang boleh di-PATCH.
// Nama kolom yang masuk ke SQL SELALU datang dari sini, tidak pernah dari input user.
type patchField struct {
	JSON       string
	Col        string
	Kind       patchKind
	Nullable   bool // null atau "" -> NULL
	AllowEmpty bool // "" disimpan sebagai '' (kolom NOT NULL default '')
	Max        int
	Enum       []string
	Label      string                     // label di tab Riwayat
	Raw        func(d *ProjectDetail) any // nilai sekarang (id untuk FK): deteksi "tidak berubah"
	Show       func(d *ProjectDetail) any // nilai tampil di audit (nama untuk FK); nil = pakai Raw
}

func (f *patchField) show(d *ProjectDetail) any {
	if f.Show != nil {
		return f.Show(d)
	}
	return f.Raw(d)
}

func yesNo(b bool) string {
	if b {
		return "Ya"
	}
	return "Tidak"
}

var patchFields = []patchField{
	{JSON: "name", Col: "name", Kind: pString, Max: 200, Label: "Nama",
		Raw: func(d *ProjectDetail) any { return d.Name }},
	{JSON: "crNumber", Col: "cr_number", Kind: pString, Nullable: true, Max: 100, Label: "Nomor CR",
		Raw: func(d *ProjectDetail) any { return d.CRNumber }},
	{JSON: "brNumber", Col: "br_number", Kind: pString, Nullable: true, Max: 100, Label: "Nomor BR",
		Raw: func(d *ProjectDetail) any { return d.BRNumber }},
	{JSON: "directorateId", Col: "directorate_id", Kind: pUUID, Label: "Direktorat",
		Raw:  func(d *ProjectDetail) any { return d.DirectorateID },
		Show: func(d *ProjectDetail) any { return d.Directorate }},
	{JSON: "subdirectorateId", Col: "subdirectorate_id", Kind: pUUID, Label: "Subdirektorat",
		Raw:  func(d *ProjectDetail) any { return d.SubdirectorateID },
		Show: func(d *ProjectDetail) any { return d.Subdirectorate }},
	{JSON: "categoryId", Col: "category_id", Kind: pUUID, Label: "Kategori",
		Raw:  func(d *ProjectDetail) any { return d.CategoryID },
		Show: func(d *ProjectDetail) any { return d.Category }},
	{JSON: "priorityId", Col: "priority_id", Kind: pUUID, Label: "Priority",
		Raw:  func(d *ProjectDetail) any { return d.PriorityID },
		Show: func(d *ProjectDetail) any { return d.Priority }},
	{JSON: "vendorId", Col: "vendor_id", Kind: pUUID, Nullable: true, Label: "Vendor",
		Raw:  func(d *ProjectDetail) any { return d.VendorID },
		Show: func(d *ProjectDetail) any { return d.Vendor }},
	{JSON: "poNumber", Col: "po_number", Kind: pString, Nullable: true, Max: 100, Label: "Nomor PO",
		Raw: func(d *ProjectDetail) any { return d.PONumber }},
	{JSON: "poStatus", Col: "po_status", Kind: pEnum, Enum: []string{"Belum Ada", "Draft", "Approved"}, Label: "Status PO",
		Raw: func(d *ProjectDetail) any { return d.POStatus }},
	{JSON: "gitRepo", Col: "git_repo", Kind: pString, AllowEmpty: true, Max: 500, Label: "Git Repo",
		Raw: func(d *ProjectDetail) any { return d.GitRepo }},
	{JSON: "hasMigration", Col: "has_migration", Kind: pBool, Label: "Ada Migrasi",
		Raw:  func(d *ProjectDetail) any { return d.HasMigration },
		Show: func(d *ProjectDetail) any { return yesNo(d.HasMigration) }},
}

func (f *patchField) decode(raw json.RawMessage) (any, string) {
	if string(raw) == "null" {
		if f.Nullable {
			return nil, ""
		}
		return nil, "Tidak boleh null"
	}
	if f.Kind == pBool {
		var b bool
		if json.Unmarshal(raw, &b) != nil {
			return nil, "Harus true atau false"
		}
		return b, ""
	}

	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nil, "Harus berupa teks"
	}
	s = strings.TrimSpace(s)
	if s == "" {
		switch {
		case f.Nullable:
			return nil, ""
		case f.AllowEmpty:
			return "", ""
		default:
			return nil, "Wajib diisi"
		}
	}
	if f.Max > 0 && utf8.RuneCountInString(s) > f.Max {
		return nil, fmt.Sprintf("Maksimal %d karakter", f.Max)
	}
	switch f.Kind {
	case pUUID:
		if !validation.IsUUID(s) {
			return nil, "Format UUID tidak valid"
		}
		return strings.ToLower(s), ""
	case pEnum:
		if !slices.Contains(f.Enum, s) {
			return nil, "Harus salah satu dari: " + strings.Join(f.Enum, ", ")
		}
	}
	return s, ""
}

// parsePatch memvalidasi body terhadap whitelist. Fungsi murni -> gampang dites.
// Hasil: map kolom -> nilai (nil = set NULL), dan map field -> pesan error.
func parsePatch(body map[string]json.RawMessage) (map[string]any, map[string]string) {
	byJSON := make(map[string]*patchField, len(patchFields))
	for i := range patchFields {
		byJSON[patchFields[i].JSON] = &patchFields[i]
	}

	vals := map[string]any{}
	errs := map[string]string{}
	for key, raw := range body {
		if key == "status" {
			errs["status"] = "Gunakan POST /projects/{id}/status untuk mengubah status."
			continue
		}
		f, ok := byJSON[key]
		if !ok {
			errs[key] = "Field tidak dikenal atau tidak boleh diubah"
			continue
		}
		v, msg := f.decode(raw)
		if msg != "" {
			errs[key] = msg
			continue
		}
		vals[f.Col] = v
	}
	return vals, errs
}

// changedFields membuang field yang nilainya sama dengan kondisi sekarang.
func changedFields(before *ProjectDetail, vals map[string]any) map[string]any {
	out := map[string]any{}
	for i := range patchFields {
		f := &patchFields[i]
		if v, ok := vals[f.Col]; ok && !audit.Equal(f.Raw(before), v) {
			out[f.Col] = v
		}
	}
	return out
}

// lockProject mengunci baris project (FOR UPDATE) sambil memeriksa scope. Dipakai PATCH, status, dan DELETE.
// Penguncian menyerialkan edit yang barengan, jadi diff audit nggak pernah dihitung dari "before" yang basi.
// 404 untuk "tidak ada" dan "dilarang" (spec §2.3).
func (s *Service) lockProject(ctx context.Context, q db.Querier, userID, projectID string) (name, status string, err error) {
	err = q.QueryRow(ctx,
		`SELECT p.name, p.status FROM projects p WHERE p.id = $2 AND `+scopePredicate+` FOR UPDATE OF p`,
		userID, projectID).Scan(&name, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", notFound()
	}
	if err != nil {
		return "", "", apperr.Internal(err)
	}
	return name, status, nil
}

// UpdateProject: PATCH /projects/{id}
func (s *Service) UpdateProject(ctx context.Context, userID, projectID string, body map[string]json.RawMessage) (*ProjectDetail, error) {
	if !validation.IsUUID(projectID) {
		return nil, notFound()
	}
	vals, errs := parsePatch(body)
	if len(errs) > 0 {
		return nil, apperr.Validation(errs)
	}
	if len(vals) == 0 {
		return nil, apperr.BadRequest("Tidak ada field yang diubah.")
	}

	var result *ProjectDetail
	err := db.WithTx(ctx, s.db, func(tx pgx.Tx) error {
		if _, _, err := s.lockProject(ctx, tx, userID, projectID); err != nil {
			return err
		}
		before, err := s.loadDetail(ctx, tx, userID, projectID)
		if err != nil {
			return err
		}

		// ganti direktorat tanpa subdit baru = subdit lama bukan milik direktorat baru (spec §6)
		if v, ok := vals["directorate_id"]; ok && v != before.DirectorateID {
			if _, has := vals["subdirectorate_id"]; !has {
				return apperr.Validation(map[string]string{"subdirectorateId": "Pilih subdit untuk direktorat yang baru"})
			}
		}

		changed := changedFields(before, vals)
		if len(changed) == 0 {
			result = before // tidak ada perubahan nyata: tanpa UPDATE, tanpa audit
			return nil
		}

		sets, args := []string{}, []any{}
		for _, col := range slices.Sorted(maps.Keys(changed)) {
			args = append(args, changed[col])
			sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
		}
		args = append(args, projectID)
		if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE projects SET %s WHERE id = $%d`,
			strings.Join(sets, ", "), len(args)), args...); err != nil {
			return err
		}

		after, err := s.loadDetail(ctx, tx, userID, projectID)
		if err != nil {
			return err
		}

		var diff audit.Diff
		for i := range patchFields {
			f := &patchFields[i]
			if _, ok := changed[f.Col]; ok {
				diff.Add(f.Label, f.show(before), f.show(after))
			}
		}
		if err := audit.Record(ctx, tx, diff.Entries(audit.Entry{
			ProjectID: projectID, EntityType: "project", EntityID: projectID,
			Label: "Project", ChangedBy: userID,
		})...); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		return nil, mapProjectWriteErr(err)
	}
	if err := s.fillPermissions(ctx, userID, result); err != nil {
		return nil, err
	}
	return result, nil
}
