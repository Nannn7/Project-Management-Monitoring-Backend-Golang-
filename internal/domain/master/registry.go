package master

import (
	"fmt"
	"slices"
	"strings"
)

type kind int

const (
	kString kind = iota
	kBool
	kInt
	kUUID
)

type field struct {
	JSON     string // nama di body/response (camelCase)
	Col      string // nama kolom DB
	Kind     kind
	Required bool // wajib saat POST
	Editable bool // boleh di PATCH
}

type ref struct {
	Table  string // tabel yang mereferensi
	Col    string // kolom FK-nya
	Entity string // label di respons IN_USE
}

type resourceDef struct {
	Table      string
	Label      string   // dipakai di pesan error
	Select     string   // kolom SELECT, alias camelCase
	From       string   // default: Table + " t"
	SearchCols []string // kolom untuk ?q=
	OrderBy    string
	Fields     []field
	RefField   string // field FK di body (untuk error 422 kalau referensi nggak ada)
	Refs       []ref  // sumber detail IN_USE
	// baris dengan kolom boolean ini = true dilindungi
	ProtectDeleteCol string
	ProtectEditCol   string // melindungi perubahan "name"
}

const baseSelect = `t.id::text AS id, t.sort_order AS "sortOrder", t.is_active AS "isActive",
	t.created_at AS "createdAt", t.updated_at AS "updatedAt"`

// field umum, hanya bisa diubah lewat PATCH (sortOrder diisi otomatis saat create)
var baseFields = []field{
	{JSON: "isActive", Col: "is_active", Kind: kBool, Editable: true},
	{JSON: "sortOrder", Col: "sort_order", Kind: kInt, Editable: true},
}

func nameOnly(table, label string, refs ...ref) resourceDef {
	return resourceDef{
		Table:      table,
		Label:      label,
		Select:     `t.name AS name, ` + baseSelect,
		SearchCols: []string{"name"},
		OrderBy:    "t.sort_order, t.name",
		Fields:     []field{{JSON: "name", Col: "name", Kind: kString, Required: true, Editable: true}},
		Refs:       refs,
	}
}

var registry = map[string]resourceDef{
	"directorates": nameOnly("master_directorates", "Direktorat",
		ref{"master_subdirectorates", "directorate_id", "subdirectorates"},
		ref{"employees", "directorate_id", "employees"},
		ref{"projects", "directorate_id", "projects"},
		ref{"project_members", "directorate_id", "project_members"},
		ref{"app_users", "directorate_id", "users"}),

	"vendors": nameOnly("master_vendors", "Vendor",
		ref{"projects", "vendor_id", "projects"}),
	"project-categories": nameOnly("master_project_categories", "Kategori project",
		ref{"projects", "category_id", "projects"}),
	"project-priorities": nameOnly("master_project_priorities", "Priority project",
		ref{"projects", "priority_id", "projects"}),
	"approval-levels": nameOnly("master_approval_levels", "Level approval",
		ref{"document_approvers", "approval_level_id", "approvers"}),
	"task-categories": nameOnly("master_task_categories", "Kategori task",
		ref{"tasks", "category_id", "tasks"}),
	"meeting-categories": nameOnly("master_meeting_categories", "Kategori meeting",
		ref{"meetings", "category_id", "meetings"}),

	"subdirectorates": {
		Table:      "master_subdirectorates",
		Label:      "Subdit",
		Select:     `t.name AS name, t.directorate_id::text AS "directorateId", d.name AS directorate, ` + baseSelect,
		From:       "master_subdirectorates t JOIN master_directorates d ON d.id = t.directorate_id",
		SearchCols: []string{"name"},
		OrderBy:    "d.name, t.sort_order, t.name",
		Fields: []field{
			{JSON: "directorateId", Col: "directorate_id", Kind: kUUID, Required: true}, // nggak bisa dipindah direktorat
			{JSON: "name", Col: "name", Kind: kString, Required: true, Editable: true},
		},
		RefField: "directorateId",
		Refs: []ref{
			{"employees", "subdirectorate_id", "employees"},
			{"projects", "subdirectorate_id", "projects"},
			{"project_members", "subdirectorate_id", "project_members"},
			{"app_users", "subdirectorate_id", "users"},
		},
	},

	"project-roles": {
		Table:      "master_project_roles",
		Label:      "Role project",
		Select:     `t.code AS code, t.name AS name, t.is_system AS "isSystem", ` + baseSelect,
		SearchCols: []string{"name", "code"},
		OrderBy:    "t.sort_order, t.name",
		Fields: []field{
			{JSON: "code", Col: "code", Kind: kString, Required: true}, // read-only setelah dibuat
			{JSON: "name", Col: "name", Kind: kString, Required: true, Editable: true},
		},
		ProtectDeleteCol: "is_system",
		Refs: []ref{
			{"employee_roles", "project_role_id", "employee_roles"},
			{"project_members", "project_role_id", "project_members"},
		},
	},

	"dependency-systems": {
		Table:      "master_dependency_systems",
		Label:      "Sistem dependency",
		Select:     `t.name AS name, t.is_other AS "isOther", ` + baseSelect,
		SearchCols: []string{"name"},
		OrderBy:    "t.sort_order, t.name",
		Fields:     []field{{JSON: "name", Col: "name", Kind: kString, Required: true, Editable: true}},
		// baris "Lainnya" (is_other) nggak boleh dihapus / diganti namanya
		ProtectDeleteCol: "is_other",
		ProtectEditCol:   "is_other",
		Refs:             []ref{{"phase_dependencies", "system_id", "dependencies"}},
	},

	"document-types": {
		Table:      "master_document_types",
		Label:      "Jenis dokumen",
		Select:     `t.code AS code, t.description AS description, ` + baseSelect,
		SearchCols: []string{"code", "description"},
		OrderBy:    "t.sort_order, t.code",
		Fields: []field{
			{JSON: "code", Col: "code", Kind: kString, Required: true, Editable: true},
			{JSON: "description", Col: "description", Kind: kString, Required: true, Editable: true},
		},
		Refs: []ref{{"documents", "document_type_id", "documents"}},
	},

	"test-types": {
		Table:      "master_test_types",
		Label:      "Jenis testing",
		Select:     `t.code AS code, t.requires_previous AS "requiresPrevious", t.allow_parallel AS "allowParallel", ` + baseSelect,
		SearchCols: []string{"code"},
		OrderBy:    "t.sort_order, t.code",
		Fields: []field{
			{JSON: "code", Col: "code", Kind: kString, Required: true, Editable: true},
			{JSON: "requiresPrevious", Col: "requires_previous", Kind: kBool, Editable: true},
			{JSON: "allowParallel", Col: "allow_parallel", Kind: kBool, Editable: true},
		},
		Refs: []ref{{"phase_test_stages", "test_type_id", "test_stages"}},
	},
}

func (d *resourceDef) from() string {
	if d.From != "" {
		return d.From
	}
	return d.Table + " t"
}

// orderBy: ?sort=name / ?sort=-name. Hanya kolom whitelist; selalu ada tiebreaker t.id
// supaya pagination stabil (tanpa ini, baris bisa dobel/hilang antar halaman).
func (d *resourceDef) orderBy(sort string) string {
	if sort == "" {
		return d.OrderBy + ", t.id"
	}
	dir := "ASC"
	if strings.HasPrefix(sort, "-") {
		dir, sort = "DESC", sort[1:]
	}
	if sort == "createdAt" {
		return fmt.Sprintf("t.created_at %s, t.id", dir)
	}
	for _, f := range slices.Concat(d.Fields, baseFields) {
		if f.JSON == sort {
			return fmt.Sprintf("t.%s %s, t.id", f.Col, dir)
		}
	}
	return d.OrderBy + ", t.id" // sort tak dikenal -> default
}
