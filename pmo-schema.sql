-- =============================================================================
-- PMO App — skema PostgreSQL (draft v1)
-- Cakupan: menu Dashboard, Project (list + detail), Master Data.
-- Tabel yang DIMILIKI menu lain tapi dibaca/ditulis sebagian oleh detail project
-- (tasks, meetings, audit_logs, app_users/roles) ikut ditulis di sini supaya FK-nya valid;
-- ditandai "[milik menu X]".
--
-- Konvensi:
--   * PK uuid (gen_random_uuid() bawaan PostgreSQL 13+).
--   * Nilai enum (status, stage, dst) pakai text + CHECK, bukan tipe ENUM — lebih gampang dimigrasi.
--   * Semua referensi ke master pakai FK id, BUKAN nama (di mock, project nyimpen nama string).
--   * Waktu: timestamptz. Tanggal murni (start_date, dst): date.
-- =============================================================================

create extension if not exists pg_trgm;   -- untuk search ILIKE di nama project

create or replace function set_updated_at() returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end $$;

-- =============================================================================
-- 1. MASTER DATA (13 tab di menu Master Data; Master Pegawai ada di bagian 3)
--    Kolom umum: name, sort_order, is_active (nonaktifkan = alternatif hapus kalau sudah dipakai).
-- =============================================================================

create table master_directorates (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_directorates_name on master_directorates (lower(name));

create table master_subdirectorates (
  id uuid primary key default gen_random_uuid(),
  directorate_id uuid not null references master_directorates(id) on delete restrict,
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (id, directorate_id)   -- dipakai FK komposit: memastikan subdit memang milik direktoratnya
);
create unique index ux_master_subdirectorates_name on master_subdirectorates (directorate_id, lower(name));

create table master_vendors (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_vendors_name on master_vendors (lower(name));

create table master_project_categories (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_project_categories_name on master_project_categories (lower(name));

-- sort_order = tingkat prioritas (1 = Rendah ... 3 = Tinggi). Dipakai sorter kolom Priority di list project.
create table master_project_priorities (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_project_priorities_name on master_project_priorities (lower(name));

-- sort_order = urutan hierarki (Staff < Manager < Kepala Divisi)
create table master_approval_levels (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_approval_levels_name on master_approval_levels (lower(name));

-- "code" stabil karena aturan akses (PMO/Developer) bergantung padanya; name boleh diganti label-nya.
create table master_project_roles (
  id uuid primary key default gen_random_uuid(),
  code text not null unique,                 -- PMO | DEVELOPER | USER | QA | SECURITY
  name text not null,
  is_system boolean not null default false,  -- baris sistem: tidak boleh dihapus / diganti code-nya
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_project_roles_name on master_project_roles (lower(name));

create table master_task_categories (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_task_categories_name on master_task_categories (lower(name));

create table master_meeting_categories (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_meeting_categories_name on master_meeting_categories (lower(name));

-- is_other = baris "Lainnya": pilih ini => user wajib isi custom_system_name di dependency.
create table master_dependency_systems (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  is_other boolean not null default false,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_dependency_systems_name on master_dependency_systems (lower(name));
create unique index ux_master_dependency_systems_other on master_dependency_systems (is_other) where is_other;

create table master_document_types (
  id uuid primary key default gen_random_uuid(),
  code text not null,            -- BR | CR | FSD | PO
  description text not null,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_document_types_code on master_document_types (upper(code));

create table master_test_types (
  id uuid primary key default gen_random_uuid(),
  code text not null,            -- IUT | SIT | UAT | TO | Pentest
  requires_previous boolean not null default false,
  allow_parallel boolean not null default false,
  sort_order int not null default 0,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_master_test_types_code on master_test_types (lower(code));

-- =============================================================================
-- 2. AKUN & RBAC  [milik menu Manajemen User — scaffold minimal supaya FK valid]
-- =============================================================================

create table roles (
  id uuid primary key default gen_random_uuid(),
  name text not null,            -- Administrator | PMO | Developer | QA | User / Viewer
  description text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index ux_roles_name on roles (lower(name));

create table role_permissions (
  role_id uuid not null references roles(id) on delete cascade,
  menu_key text not null,        -- dashboard | projects | tasks | meetings | ... | master-data | user-management
  can_view boolean not null default false,
  can_create boolean not null default false,
  can_edit boolean not null default false,
  can_delete boolean not null default false,
  primary key (role_id, menu_key)
);

create table app_users (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  email text not null,
  password_hash text,
  directorate_id uuid references master_directorates(id) on delete restrict,
  subdirectorate_id uuid,
  is_active boolean not null default true,
  last_login_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (subdirectorate_id, directorate_id) references master_subdirectorates(id, directorate_id)
);
create unique index ux_app_users_email on app_users (lower(email));

create table user_roles (
  user_id uuid not null references app_users(id) on delete cascade,
  role_id uuid not null references roles(id) on delete restrict,
  primary key (user_id, role_id)
);

-- =============================================================================
-- 3. MASTER PEGAWAI (tab "Master Pegawai")
--    Bukan akun login. user_id menghubungkan pegawai -> akun login (kalau punya);
--    rantai aturan akses: app_users.id <- employees.user_id <- project_members.employee_id
-- =============================================================================

create table employees (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  directorate_id uuid not null references master_directorates(id) on delete restrict,
  subdirectorate_id uuid not null,
  user_id uuid unique references app_users(id) on delete set null,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  foreign key (subdirectorate_id, directorate_id) references master_subdirectorates(id, directorate_id)
);
create index ix_employees_name_trgm on employees using gin (name gin_trgm_ops);
create index ix_employees_directorate on employees (directorate_id);

create table employee_roles (   -- "Bisa Jadi Role" (multi-select di form pegawai)
  employee_id uuid not null references employees(id) on delete cascade,
  project_role_id uuid not null references master_project_roles(id) on delete restrict,
  primary key (employee_id, project_role_id)
);

-- =============================================================================
-- 4. FILE UPLOAD (dokumen tipe "Upload")
-- =============================================================================

create table files (
  id uuid primary key default gen_random_uuid(),
  original_name text not null,
  mime_type text not null,
  size_bytes bigint not null check (size_bytes >= 0),
  storage_key text not null,         -- path di disk / object storage
  uploaded_by uuid not null references app_users(id),
  created_at timestamptz not null default now()
);

-- =============================================================================
-- 5. PROJECT
-- =============================================================================

create table projects (
  id uuid primary key default gen_random_uuid(),
  cr_number text,
  br_number text,
  name text not null,
  directorate_id uuid not null references master_directorates(id) on delete restrict,
  subdirectorate_id uuid not null,
  category_id uuid not null references master_project_categories(id) on delete restrict,
  priority_id uuid not null references master_project_priorities(id) on delete restrict,
  status text not null default 'Active' check (status in ('Active','Hold','Cancelled')),
  vendor_id uuid references master_vendors(id) on delete restrict,
  po_number text,
  po_status text not null default 'Belum Ada' check (po_status in ('Belum Ada','Draft','Approved')),
  git_repo text not null default '',
  has_migration boolean not null default false,     -- false => tab "Migrasi & Rekon" disembunyikan
  overall_progress smallint not null default 0 check (overall_progress between 0 and 100),
  created_by uuid not null references app_users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  deleted_at timestamptz,                            -- soft delete
  foreign key (subdirectorate_id, directorate_id) references master_subdirectorates(id, directorate_id)
);
create unique index ux_projects_cr on projects (lower(cr_number)) where cr_number is not null and deleted_at is null;
create unique index ux_projects_br on projects (lower(br_number)) where br_number is not null and deleted_at is null;
create index ix_projects_name_trgm on projects using gin (name gin_trgm_ops);
create index ix_projects_status on projects (status) where deleted_at is null;
create index ix_projects_created_by on projects (created_by);
create index ix_projects_directorate on projects (directorate_id);

-- Tim project. Dua jenis baris:
--   (a) terhubung ke master pegawai : employee_id terisi, name NULL (nama diambil via join -> selalu mutakhir)
--   (b) manual (Developer vendor / User): employee_id NULL, name terisi
-- directorate/subdirectorate di sini = asal orang itu di project ini (bisa beda dari master); NULL untuk vendor.
create table project_members (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  project_role_id uuid not null references master_project_roles(id) on delete restrict,
  employee_id uuid references employees(id) on delete restrict,
  name text,
  is_primary boolean not null default false,
  is_vendor boolean not null default false,
  directorate_id uuid references master_directorates(id) on delete restrict,
  subdirectorate_id uuid,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check ((employee_id is not null and name is null) or (employee_id is null and coalesce(length(btrim(name)), 0) > 0)),
  foreign key (subdirectorate_id, directorate_id) references master_subdirectorates(id, directorate_id)
);
create unique index ux_project_members_employee on project_members (project_id, project_role_id, employee_id) where employee_id is not null;
create index ix_project_members_project on project_members (project_id);
create index ix_project_members_employee on project_members (employee_id);

create table project_hold_history (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  held_at date not null default current_date,
  reason text not null,
  held_by uuid not null references app_users(id),
  resumed_at date,
  resumed_by uuid references app_users(id)
);
create unique index ux_hold_active on project_hold_history (project_id) where resumed_at is null;  -- max 1 hold aktif

create table phases (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  name text not null,                                     -- "Fase 1"
  stage text not null check (stage in ('Dokumen','Development','Testing','Live')),
  sort_order int not null,                                -- urutan fase; "fase berjalan" di dashboard = sort_order terbesar
  start_date date not null,
  target_live_date date not null,
  previous_phase_id uuid references phases(id) on delete set null,
  scope text[] not null default '{}',                     -- poin-poin scope (bullet)
  note text,
  -- Angka progress: belum ada input di UI. Lihat keputusan terbuka di API-SPEC (hitung otomatis vs input manual).
  progress_percent smallint not null default 0 check (progress_percent between 0 and 100),
  document_progress smallint not null default 0 check (document_progress between 0 and 100),
  development_progress smallint not null default 0 check (development_progress between 0 and 100),
  testing_progress smallint not null default 0 check (testing_progress between 0 and 100),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (start_date <= target_live_date),
  unique (project_id, sort_order)
);

create table documents (
  id uuid primary key default gen_random_uuid(),
  phase_id uuid not null references phases(id) on delete cascade,
  document_type_id uuid not null references master_document_types(id) on delete restrict,
  status text not null default 'Draft' check (status in ('Draft','Review','Approved','Rejected')),
  source_type text not null check (source_type in ('Link','Upload')),
  url text,
  file_id uuid references files(id) on delete restrict,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (
    (source_type = 'Link'   and url is not null and file_id is null) or
    (source_type = 'Upload' and file_id is not null and url is null)
  )
);
create index ix_documents_phase on documents (phase_id);

-- approvalLevel "2/3" di FE = jumlah 'Disetujui' / jumlah baris — DIHITUNG saat query, tidak disimpan.
create table document_approvers (
  id uuid primary key default gen_random_uuid(),
  document_id uuid not null references documents(id) on delete cascade,
  employee_id uuid not null references employees(id) on delete restrict,
  approval_level_id uuid not null references master_approval_levels(id) on delete restrict,
  status text not null default 'Menunggu' check (status in ('Menunggu','Disetujui','Ditolak')),
  approved_at date,                                       -- diisi saat status jadi 'Disetujui'
  sort_order int not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (document_id, employee_id)
);

-- Environment & IP. Satu tabel untuk 7 kategori; kolom spesifik-kategori nullable.
-- Validasi "kolom mana boleh diisi untuk kategori apa" dilakukan di backend (cerminan ENV_FIELD_VISIBILITY di FE).
-- JANGAN simpan password/secret di sini — hanya IP/host/URL/nama.
create table environments (
  id uuid primary key default gen_random_uuid(),
  phase_id uuid not null references phases(id) on delete cascade,
  category text not null check (category in ('Git Repository','Local Developer','Staging','Production','Database','SSH','User')),
  label text not null,
  value text not null,                 -- IP / host / URL / path repo
  port text,
  segment text,                        -- Local Developer / User
  is_template boolean not null default false,   -- Git Repository
  main_branch text,                    -- Git Repository
  dns text,                            -- Staging / Production
  db_name text,                        -- Database
  container_name text,                 -- SSH
  note text,
  cpu text,
  memory text,
  storage text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index ix_environments_phase on environments (phase_id, category);

create table environment_extra_specs (   -- tombol "Tambah Lainnya" di spesifikasi server
  id uuid primary key default gen_random_uuid(),
  environment_id uuid not null references environments(id) on delete cascade,
  label text not null,
  value text not null,
  sort_order int not null default 0
);

create table phase_migrations (          -- "Migrasi & Rekon" (diberi prefix agar tidak bentrok dgn tabel schema_migrations tool migrasi)
  id uuid primary key default gen_random_uuid(),
  phase_id uuid not null references phases(id) on delete cascade,
  source_system text not null,
  db_name text,
  table_name text,
  description text not null,
  status text not null default 'Belum Mulai' check (status in ('Belum Mulai','On Going','Rekon Selesai')),
  report_status text not null default 'Belum Ada' check (report_status in ('Belum Ada','Draft','Final')),
  report_url text,
  pic text not null,                    -- nama bebas (di UI input teks)
  team text[] not null default '{}',
  start_date date,
  end_date date,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (end_date is null or start_date is null or end_date >= start_date)
);
create index ix_phase_migrations_phase on phase_migrations (phase_id);

create table phase_dependencies (
  id uuid primary key default gen_random_uuid(),
  phase_id uuid not null references phases(id) on delete cascade,
  system_id uuid not null references master_dependency_systems(id) on delete restrict,
  custom_system_name text,              -- wajib kalau system = baris is_other ("Lainnya") — divalidasi di backend
  service_name text,
  description text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index ix_phase_dependencies_phase on phase_dependencies (phase_id);

create table dependency_endpoints (
  id uuid primary key default gen_random_uuid(),
  dependency_id uuid not null references phase_dependencies(id) on delete cascade,
  label text not null,
  value text not null,                  -- IP / host / URL tujuan
  note text,
  sort_order int not null default 0
);

-- Tahap pengujian per fase. Di detail project hanya DITAMPILKAN; CRUD-nya nanti dari menu Pengujian.
create table phase_test_stages (
  id uuid primary key default gen_random_uuid(),
  phase_id uuid not null references phases(id) on delete cascade,
  test_type_id uuid not null references master_test_types(id) on delete restrict,
  status text not null default 'Not Started' check (status in ('Not Started','On Going','Passed','Failed')),
  progress_percent smallint not null default 0 check (progress_percent between 0 and 100),
  pic text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index ix_phase_test_stages_phase on phase_test_stages (phase_id);

-- =============================================================================
-- 6. TASK, RAPAT, AUDIT  [milik menu Task Board / Rapat / Riwayat Perubahan]
--    Dibaca oleh tab "Task & Rapat" dan "Riwayat" di detail project; review task ditulis dari sana.
-- =============================================================================

create table tasks (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  phase_id uuid references phases(id) on delete set null,
  title text not null,
  category_id uuid not null references master_task_categories(id) on delete restrict,
  status text not null default 'To Do' check (status in ('To Do','In Progress','Done','Blocked')),
  progress_percent smallint not null default 0 check (progress_percent between 0 and 100),
  pic text,
  start_date date,
  due_date date,
  review_status text not null default 'Belum Direview' check (review_status in ('Belum Direview','Disetujui','Perlu Revisi')),
  reviewed_by text,
  reviewed_at date,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index ix_tasks_project on tasks (project_id);

create table meetings (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  title text not null,
  category_id uuid not null references master_meeting_categories(id) on delete restrict,
  scheduled_at timestamptz not null,
  status text not null default 'Scheduled' check (status in ('Scheduled','Done','Cancelled')),
  attendees text[] not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index ix_meetings_project on meetings (project_id);
create index ix_meetings_upcoming on meetings (scheduled_at) where status = 'Scheduled';

-- Append-only. Ditulis OTOMATIS oleh service layer backend tiap ada perubahan (bukan dari FE).
-- Di mock, log dikaitkan lewat NAMA project (rapuh: ganti nama = riwayat putus) -> sekarang lewat project_id.
create table audit_logs (
  id uuid primary key default gen_random_uuid(),
  project_id uuid references projects(id),
  entity_type text not null,            -- project | phase | member | document | approver | environment | migration | dependency | task | master:<resource>
  entity_id uuid,
  entity_label text not null,           -- teks yang tampil di Riwayat: "Project", "Fase 1", ...
  field text not null,
  old_value text,
  new_value text,
  changed_by uuid not null references app_users(id),
  changed_at timestamptz not null default now()
);
create index ix_audit_logs_project on audit_logs (project_id, changed_at desc);

-- =============================================================================
-- 7. TRIGGER updated_at (otomatis ke semua tabel yang punya kolom updated_at)
-- =============================================================================

do $$
declare t text;
begin
  for t in
    select table_name from information_schema.columns
    where table_schema = 'public' and column_name = 'updated_at'
  loop
    execute format(
      'create trigger trg_%1$s_updated_at before update on %1$I for each row execute function set_updated_at()', t);
  end loop;
end $$;

-- =============================================================================
-- 8. SEED (diambil dari lib/mock-data.ts)
-- =============================================================================

insert into master_directorates (name, sort_order) values
  ('Direktorat Teknologi Informasi', 1),
  ('Direktorat Operasional', 2),
  ('Direktorat Keuangan', 3),
  ('Direktorat Kepatuhan & Legal', 4),
  ('Direktorat Sumber Daya Manusia', 5);

insert into master_subdirectorates (directorate_id, name, sort_order)
select d.id, v.sub, v.ord
from (values
  ('Direktorat Teknologi Informasi', 'Subdit Development', 1),
  ('Direktorat Teknologi Informasi', 'Subdit Infrastruktur', 2),
  ('Direktorat Teknologi Informasi', 'Subdit Data & Migrasi', 3),
  ('Direktorat Teknologi Informasi', 'Subdit Quality Assurance', 4),
  ('Direktorat Operasional', 'Subdit Layanan Nasabah', 1),
  ('Direktorat Operasional', 'Subdit Operasional Cabang', 2),
  ('Direktorat Keuangan', 'Subdit Treasury & Payment', 1),
  ('Direktorat Keuangan', 'Subdit Akuntansi', 2),
  ('Direktorat Kepatuhan & Legal', 'Subdit Legal & Compliance', 1),
  ('Direktorat Kepatuhan & Legal', 'Subdit Manajemen Risiko', 2),
  ('Direktorat Sumber Daya Manusia', 'Subdit Administrasi Personalia', 1),
  ('Direktorat Sumber Daya Manusia', 'Subdit Sistem Informasi SDM', 2)
) as v(dir, sub, ord)
join master_directorates d on d.name = v.dir;

insert into master_project_roles (code, name, is_system, sort_order) values
  ('PMO', 'PMO', true, 1), ('DEVELOPER', 'Developer', true, 2), ('USER', 'User', true, 3),
  ('QA', 'QA', true, 4), ('SECURITY', 'Security', true, 5);

insert into master_approval_levels (name, sort_order) values ('Staff', 1), ('Manager', 2), ('Kepala Divisi', 3);
insert into master_project_categories (name, sort_order) values ('Support', 1), ('Regulator', 2), ('Bisnis', 3);
insert into master_project_priorities (name, sort_order) values ('Rendah', 1), ('Sedang', 2), ('Tinggi', 3);
insert into master_task_categories (name, sort_order) values
  ('Development', 1), ('Migrasi/Sinkronisasi', 2), ('Testing', 3), ('To-Do Meeting', 4);
insert into master_meeting_categories (name, sort_order) values
  ('Dokumen', 1), ('Development', 2), ('Testing', 3), ('General', 4);
insert into master_vendors (name, sort_order) values
  ('PT Solusi Digital Nusantara', 1), ('PT Mitra Teknologi Bangsa', 2), ('CV Data Sinergi', 3);
insert into master_dependency_systems (name, is_other, sort_order) values
  ('Axway', false, 1), ('Fiorano', false, 2), ('Big Data', false, 3), ('Core Banking', false, 4), ('Lainnya', true, 99);
insert into master_document_types (code, description, sort_order) values
  ('BR', 'Business Requirement', 1), ('CR', 'Change Request', 2),
  ('FSD', 'Functional Spec Document', 3), ('PO', 'Purchase Order (jika ada vendor)', 4);
insert into master_test_types (code, requires_previous, allow_parallel, sort_order) values
  ('IUT', false, false, 1), ('SIT', true, false, 2), ('UAT', true, true, 3), ('TO', true, true, 4), ('Pentest', false, true, 5);

insert into roles (name, description) values
  ('Administrator', 'Akses penuh ke seluruh menu, termasuk Master Data dan Manajemen User.'),
  ('PMO', 'Mengelola project, dokumen, task, rapat, dan laporan lintas project.'),
  ('Developer', 'Mengelola task development & migrasi, mengisi IUT, lihat project yang diikuti.'),
  ('QA', 'Mengelola test script & hasil SIT, lihat task dan timeline terkait.'),
  ('User / Viewer', 'Hanya melihat project & laporan yang relevan, tanpa hak ubah data.');
