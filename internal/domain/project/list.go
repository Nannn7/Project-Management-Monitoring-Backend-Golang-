package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
	"github.com/yourorg/pmo-agi-be/internal/validation"
)

type ListParams struct {
	Q             string
	Statuses      []string
	PriorityID    string
	CategoryID    string
	DirectorateID string
	Sort          string
	Page          int
	PageSize      int
}

var statusSet = map[string]bool{"Active": true, "Hold": true, "Cancelled": true}

// Whitelist sort: nama di API -> kolom SQL. priority diurut lewat level (= sort_order master).
var sortCols = map[string]string{
	"name":            "p.name",
	"status":          "p.status",
	"priority":        "pr.sort_order",
	"overallProgress": "p.overall_progress",
	"createdAt":       "p.created_at",
}

const listFrom = `
	FROM projects p
	JOIN master_directorates d ON d.id = p.directorate_id
	JOIN master_subdirectorates sd ON sd.id = p.subdirectorate_id
	JOIN master_project_categories c ON c.id = p.category_id
	JOIN master_project_priorities pr ON pr.id = p.priority_id
	LEFT JOIN master_vendors v ON v.id = p.vendor_id`

const listSelect = `SELECT p.id::text, p.name, p.cr_number, p.br_number,
	d.name, p.directorate_id::text, sd.name, p.subdirectorate_id::text,
	c.name, p.category_id::text, pr.name, p.priority_id::text,
	p.status, v.name, p.vendor_id::text, p.overall_progress`

// Anggota untuk SEMUA project di halaman, satu query. ::text[]::uuid[] dipakai karena pgx v5
// tidak bisa meng-encode []string langsung ke uuid[] (format biner); text[] aman, lalu di-cast di SQL
// (kolom project_id tetap bisa memakai index).
const listMembersSQL = `
	SELECT m.project_id::text, m.id::text, COALESCE(e.name, m.name), r.code, m.is_vendor
	FROM project_members m
	JOIN master_project_roles r ON r.id = m.project_role_id AND r.code IN ('PMO','DEVELOPER','USER')
	LEFT JOIN employees e ON e.id = m.employee_id
	WHERE m.project_id = ANY($1::text[]::uuid[])
	ORDER BY r.sort_order, m.is_primary DESC, COALESCE(e.name, m.name), m.id`

const listPhasesSQL = `
	SELECT DISTINCT ON (project_id) project_id::text, id::text, name, stage
	FROM phases
	WHERE project_id = ANY($1::text[]::uuid[])
	ORDER BY project_id, sort_order DESC`

// buildList menyusun WHERE/ORDER dari parameter. Fungsi murni: tidak menyentuh DB, jadi gampang dites.
// $1 selalu userID (dipakai scopePredicate); filter lain menyusul $2, $3, ... lewat add().
func buildList(userID string, p ListParams) (where string, args []any, order string, errs map[string]string) {
	errs = map[string]string{}
	args = []any{userID}
	conds := []string{"(" + scopePredicate + ")"}
	add := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	// status boleh berulang: ?status=Active&status=Hold
	var statuses []string
	for _, s := range p.Statuses {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		if !statusSet[s] {
			errs["status"] = "Harus Active, Hold, atau Cancelled"
			statuses = nil
			break
		}
		statuses = append(statuses, s)
	}
	if len(statuses) > 0 {
		conds = append(conds, "p.status = ANY("+add(statuses)+"::text[])")
	}

	for _, f := range []struct{ key, col, val string }{
		{"priorityId", "p.priority_id", p.PriorityID},
		{"categoryId", "p.category_id", p.CategoryID},
		{"directorateId", "p.directorate_id", p.DirectorateID},
	} {
		if f.val == "" {
			continue
		}
		if !validation.IsUUID(f.val) {
			errs[f.key] = "Format UUID tidak valid"
			continue
		}
		conds = append(conds, f.col+" = "+add(f.val))
	}

	// q: nama, direktorat, subdit, CR, BR, dan nama anggota PMO/Developer/User (sesuai spec §6).
	// Anggota dibatasi ke 3 role yang memang tampil di list, supaya nggak ada hasil yang "tidak kelihatan kenapa cocok".
	if q := strings.TrimSpace(p.Q); q != "" {
		ph := add(validation.LikePattern(q))
		conds = append(conds, fmt.Sprintf(`(
			p.name ILIKE %[1]s OR d.name ILIKE %[1]s OR sd.name ILIKE %[1]s
			OR p.cr_number ILIKE %[1]s OR p.br_number ILIKE %[1]s
			OR EXISTS (
				SELECT 1 FROM project_members m
				JOIN master_project_roles mr ON mr.id = m.project_role_id AND mr.code IN ('PMO','DEVELOPER','USER')
				LEFT JOIN employees me ON me.id = m.employee_id
				WHERE m.project_id = p.id AND COALESCE(me.name, m.name) ILIKE %[1]s
			))`, ph))
	}

	sort := strings.TrimSpace(p.Sort)
	if sort == "" {
		sort = "-createdAt"
	}
	dir := "ASC"
	if strings.HasPrefix(sort, "-") {
		dir, sort = "DESC", sort[1:]
	}
	if col, ok := sortCols[sort]; ok {
		order = fmt.Sprintf("%s %s, p.id", col, dir) // p.id = tiebreaker supaya pagination stabil
	} else {
		errs["sort"] = "Tidak dikenal. Pilihan: name, status, priority, overallProgress, createdAt"
	}

	return strings.Join(conds, " AND "), args, order, errs
}

func scanListItem(row pgx.CollectableRow) (ProjectListItem, error) {
	var it ProjectListItem
	err := row.Scan(&it.ID, &it.Name, &it.CRNumber, &it.BRNumber,
		&it.Directorate, &it.DirectorateID, &it.Subdirectorate, &it.SubdirectorateID,
		&it.Category, &it.CategoryID, &it.Priority, &it.PriorityID,
		&it.Status, &it.Vendor, &it.VendorID, &it.OverallProgress)
	return it, err
}

// ListProjects: GET /projects
func (s *Service) ListProjects(ctx context.Context, userID string, p ListParams) ([]ProjectListItem, int, error) {
	where, args, order, errs := buildList(userID, p)
	if len(errs) > 0 {
		return nil, 0, apperr.Validation(errs)
	}

	var total int
	if err := s.db.QueryRow(ctx, "SELECT count(*) "+listFrom+" WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}

	args = append(args, p.PageSize, (p.Page-1)*p.PageSize)
	rows, err := s.db.Query(ctx, fmt.Sprintf(`%s %s WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		listSelect, listFrom, where, order, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	items, err := pgx.CollectRows(rows, scanListItem)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if len(items) == 0 {
		return []ProjectListItem{}, total, nil // halaman kelewat: data [] (bukan null), total tetap
	}

	if err := s.attachListExtras(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// attachListExtras mengisi pmo/developers/users dan currentPhase untuk semua item sekaligus.
func (s *Service) attachListExtras(ctx context.Context, items []ProjectListItem) error {
	ids := make([]string, len(items))
	pos := make(map[string]int, len(items))
	for i := range items {
		ids[i] = items[i].ID
		pos[items[i].ID] = i
		// [] bukan nil: FE melakukan .map()/.join() pada array ini
		items[i].PMO, items[i].Developers, items[i].Users = []MemberRef{}, []MemberRef{}, []MemberRef{}
	}

	type memberRow struct {
		projectID, id, name, code string
		vendor                    bool
	}
	rows, err := s.db.Query(ctx, listMembersSQL, ids)
	if err != nil {
		return apperr.Internal(err)
	}
	members, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (memberRow, error) {
		var m memberRow
		err := row.Scan(&m.projectID, &m.id, &m.name, &m.code, &m.vendor)
		return m, err
	})
	if err != nil {
		return apperr.Internal(err)
	}
	for _, m := range members {
		i, ok := pos[m.projectID]
		if !ok {
			continue
		}
		ref := MemberRef{ID: m.id, Name: m.name}
		switch m.code {
		case "PMO":
			items[i].PMO = append(items[i].PMO, ref)
		case "DEVELOPER":
			v := m.vendor
			ref.IsVendor = &v // developers selalu membawa isVendor, termasuk false
			items[i].Developers = append(items[i].Developers, ref)
		case "USER":
			items[i].Users = append(items[i].Users, ref)
		}
	}

	rows, err = s.db.Query(ctx, listPhasesSQL, ids)
	if err != nil {
		return apperr.Internal(err)
	}
	type phaseRow struct {
		projectID string
		ref       PhaseRef
	}
	phases, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (phaseRow, error) {
		var ph phaseRow
		err := row.Scan(&ph.projectID, &ph.ref.ID, &ph.ref.Name, &ph.ref.Stage)
		return ph, err
	})
	if err != nil {
		return apperr.Internal(err)
	}
	for _, ph := range phases {
		if i, ok := pos[ph.projectID]; ok {
			ref := ph.ref
			items[i].CurrentPhase = &ref
		}
	}
	return nil
}
