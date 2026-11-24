package project

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var placeholderRe = regexp.MustCompile(`\$(\d+)`)

func maxPlaceholder(s string) int {
	max := 0
	for _, m := range placeholderRe.FindAllStringSubmatch(s, -1) {
		if n, _ := strconv.Atoi(m[1]); n > max {
			max = n
		}
	}
	return max
}

func TestBuildListDefault(t *testing.T) {
	where, args, order, errs := buildList("u1", ListParams{})
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if order != "p.created_at DESC, p.id" {
		t.Errorf("default order = %q", order)
	}
	if len(args) != 1 || args[0] != "u1" {
		t.Errorf("args = %v", args)
	}
	if !strings.Contains(where, "p.deleted_at IS NULL") {
		t.Error("scope predicate hilang dari WHERE")
	}
}

// Jaga nomor placeholder $n selalu sinkron dengan jumlah args (bug off-by-one klasik).
func TestBuildListPlaceholdersMatchArgs(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	where, args, order, errs := buildList("u1", ListParams{
		Q: "100%_x", Statuses: []string{"Active", "Hold"},
		PriorityID: id, CategoryID: id, DirectorateID: id, Sort: "-priority",
	})
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if got := maxPlaceholder(where); got != len(args) {
		t.Fatalf("placeholder tertinggi $%d, tapi args = %d", got, len(args))
	}
	if order != "pr.sort_order DESC, p.id" {
		t.Errorf("order = %q", order)
	}
	found := false
	for _, a := range args {
		if s, ok := a.(string); ok && s == `%100\%\_x%` {
			found = true
		}
	}
	if !found {
		t.Errorf("wildcard LIKE di q harus di-escape; args = %v", args)
	}
}

func TestBuildListInvalid(t *testing.T) {
	tests := []struct {
		name string
		p    ListParams
		key  string
	}{
		{"status ngawur", ListParams{Statuses: []string{"Active", "Foo"}}, "status"},
		{"priorityId bukan uuid", ListParams{PriorityID: "abc"}, "priorityId"},
		{"categoryId bukan uuid", ListParams{CategoryID: "abc"}, "categoryId"},
		{"directorateId bukan uuid", ListParams{DirectorateID: "abc"}, "directorateId"},
		{"sort ngawur", ListParams{Sort: "drop table"}, "sort"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, errs := buildList("u1", tt.p)
			if _, ok := errs[tt.key]; !ok {
				t.Fatalf("harusnya error di %q, dapat %v", tt.key, errs)
			}
		})
	}
}

func TestBuildListIgnoresEmptyStatus(t *testing.T) {
	_, args, _, errs := buildList("u1", ListParams{Statuses: []string{"", " "}})
	if len(errs) != 0 || len(args) != 1 {
		t.Fatalf("status kosong harus diabaikan: errs=%v args=%v", errs, args)
	}
}

func TestMemberRefJSON(t *testing.T) {
	f := false
	b, _ := json.Marshal(MemberRef{ID: "1", Name: "x", IsVendor: &f})
	if !strings.Contains(string(b), `"isVendor":false`) {
		t.Errorf("developers harus membawa isVendor:false, dapat %s", b)
	}
	b, _ = json.Marshal(MemberRef{ID: "1", Name: "x"})
	if strings.Contains(string(b), "isVendor") {
		t.Errorf("pmo/users tidak boleh punya isVendor, dapat %s", b)
	}
}
