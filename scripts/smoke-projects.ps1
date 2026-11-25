param(
    [string]$Base = "http://localhost:8080/api/v1",
    [Parameter(Mandatory)][string]$AdminEmail,
    [string]$AdminPassword = $env:SMOKE_PASSWORD,
    [string]$PmoEmail = "pmo1@ag.co.id",
    [string]$Pmo2Email = "pmo2@ag.co.id",
    [string]$DevPassword = "Dev-Only-123!"
)
. "$PSScriptRoot\lib.ps1"
$script:Base = $Base
if (-not $AdminPassword) {
    $AdminPassword = [System.Net.NetworkCredential]::new("", (Read-Host "Password admin" -AsSecureString)).Password
}

function LoginAs($email, $pw) {
    $script:token = $null
    $r = Api POST "/auth/login" @{ email = $email; password = $pw }
    if ($r.Status -ne 200) { Write-Host "STOP: login $email gagal ($($r.Status))" -ForegroundColor Red; exit 1 }
    $script:token = $r.Data.data.accessToken
}

$run = Get-Date -Format "HHmmss"
$zero = "00000000-0000-0000-0000-000000000000"

# ================= ADMIN =================
LoginAs $AdminEmail $AdminPassword
$lk = (Api GET "/lookups").Data.data
$dirA = @($lk.directorates)[0]; $dirB = @($lk.directorates)[1]
$subA = @($lk.subdirectorates) | Where-Object { $_.directorateId -eq $dirA.id } | Select-Object -First 1
$subB = @($lk.subdirectorates) | Where-Object { $_.directorateId -eq $dirB.id } | Select-Object -First 1
$cat = @($lk.projectCategories)[0]
$pri = @($lk.projectPriorities)[0]
if (-not ($dirA -and $dirB -and $subA -and $subB -and $cat -and $pri)) { Write-Host "STOP: data seed nggak lengkap"; exit 1 }
Check "priorities punya level" ($null -ne $pri.level) $true

function PBody($name, $sub) {
    @{ name = $name; directorateId = $dirA.id; subdirectorateId = $sub; categoryId = $cat.id; priorityId = $pri.id }
}

$c = Api POST "/projects" (PBody "SMOKE Project A $run" $subA.id)
Check "admin: create project" $c.Status 201
$p = $c.Data.data
$projA = $p.id
Check "  status Active" $p.status "Active"
Check "  poStatus" $p.poStatus "Belum Ada"
Check "  hasMigration false" $p.hasMigration $false
Check "  overallProgress 0" $p.overallProgress 0
Check "  gitRepo kosong" $p.gitRepo ""
Check "  createdBy terisi" ($null -ne $p.createdBy.id) $true
Check "  admin tanpa pegawai: pmo kosong" @($p.pmo).Count 0
Check "  developers/users kosong" (@($p.developers).Count + @($p.users).Count) 0
Check "  holdHistory kosong (bukan null)" ($null -ne $p.holdHistory) $true
Check "  canEdit" $p.permissions.canEdit $true
Check "  canDelete (admin)" $p.permissions.canDelete $true
Check "  1 fase" @($p.phases).Count 1
$ph = @($p.phases)[0]
Check "  fase: nama" $ph.name "Fase 1"
Check "  fase: stage" $ph.stage "Dokumen"
Check "  fase: startDate = hari ini (WIB)" $ph.startDate ([DateTime]::UtcNow.AddHours(7).ToString("yyyy-MM-dd"))
Check "  fase: sortOrder" $ph.sortOrder 1
Check "  fase: previousPhaseId null" ($null -eq $ph.previousPhaseId) $true
Check "  fase: scope kosong" @($ph.scope).Count 0
Check "  fase: target = start + 90 hari" ([int](([datetime]$ph.targetLiveDate) - ([datetime]$ph.startDate)).TotalDays) 90
Check "  fase: documents [] (bukan null)" ($null -ne $ph.documents) $true
Check "  fase: progress 0" ($ph.progressPercent + $ph.documentProgress + $ph.developmentProgress + $ph.testingProgress) 0

# validasi
Check "create tanpa name" (Api POST "/projects" @{ directorateId = $dirA.id }).Status 422
$r = Api POST "/projects" (PBody "SMOKE Project X $run" $subB.id)
Check "create subdit milik direktorat lain" $r.Status 422
Check "  fields.subdirectorateId" ($null -ne $r.Err.fields.subdirectorateId) $true
$bad = PBody "SMOKE Project X $run" $subA.id; $bad.categoryId = $zero
$r = Api POST "/projects" $bad
Check "create kategori tidak ada" $r.Status 422
Check "  fields.categoryId" ($null -ne $r.Err.fields.categoryId) $true
$bad = PBody "SMOKE Project X $run" $subA.id; $bad.categoryId = "abc"
Check "create categoryId bukan uuid" (Api POST "/projects" $bad).Status 422
Check "create field tak dikenal (status)" (Api POST "/projects" ((PBody "SMOKE Project X $run" $subA.id) + @{ status = "Hold" })).Status 400
$v = Api POST "/projects" ((PBody "SMOKE Project V $run" $subA.id) + @{ vendorId = "" })
Check "vendorId kosong diterima" $v.Status 201
Check "  vendor null" ($null -eq $v.Data.data.vendor) $true
$projV = $v.Data.data.id

# get
$g = Api GET "/projects/$projA"
Check "get detail" $g.Status 200
Check "  id sama" $g.Data.data.id $projA
Check "get uuid tidak ada" (Api GET "/projects/$zero").Status 404
Check "get id bukan uuid" (Api GET "/projects/abc").Status 404

# ================= PMO 1 =================
LoginAs $PmoEmail $DevPassword
$c2 = Api POST "/projects" (PBody "SMOKE Project B $run" $subA.id)
Check "pmo: create project" $c2.Status 201
$p2 = $c2.Data.data
Check "  pmo: pembuat otomatis jadi PMO (1 anggota)" @($p2.pmo).Count 1
$m = @($p2.pmo)[0]
$me = (Api GET "/auth/me").Data.data
Check "  role" $m.role "PMO"
Check "  isPrimary" $m.isPrimary $true
Check "  isVendor" $m.isVendor $false
Check "  nama dari pegawai" $m.name $me.name
Check "  employeeId terisi" ($null -ne $m.employeeId) $true
Check "  direktorat disalin dari pegawai" ($null -ne $m.directorateId) $true
Check "  canEdit" $p2.permissions.canEdit $true
Check "  canDelete (PMO tidak punya)" $p2.permissions.canDelete $false
Check "pmo: get project sendiri" (Api GET "/projects/$($p2.id)").Status 200
Check "pmo: get project milik admin -> 404 (scope)" (Api GET "/projects/$projA").Status 404
$l = Api GET "/projects?q=$run"
Check "pmo: list hanya project sendiri" $l.Data.meta.total 1
Check "  id = project miliknya" @($l.Data.data)[0].id $p2.id
Check "  pmo list memuat anggota PMO" @(@($l.Data.data)[0].pmo).Count 1
Check "pmo: DELETE project -> 403 (tanpa projects.delete)" (Api DELETE "/projects/$($p2.id)").Status 403
Check "pmo: master-data -> 403" (Api GET "/master/vendors").Status 403
Check "pmo: lookups -> 200" (Api GET "/lookups").Status 200

# ================= PMO 2 =================
LoginAs $Pmo2Email $DevPassword
Check "pmo2: project milik pmo1 -> 404" (Api GET "/projects/$($p2.id)").Status 404
Check "pmo2: project milik admin -> 404" (Api GET "/projects/$projA").Status 404
$c3 = Api POST "/projects" (PBody "SMOKE Project C $run" $subA.id)
Check "pmo2: create project sendiri" $c3.Status 201
Check "pmo2: list hanya project sendiri" (Api GET "/projects?q=$run").Data.meta.total 1
$projC = $c3.Data.data.id

# ================= ADMIN lagi =================
LoginAs $AdminEmail $AdminPassword
Check "admin: lihat project milik pmo1 (bypass)" (Api GET "/projects/$($p2.id)").Status 200

# ===== LIST: admin lihat semua (A, V, B, C) =====
# (jangan pakai nama variabel $base: PowerShell tidak case-sensitive, bentrok dengan param $Base)
$listQ = "q=$run&pageSize=100"
function Names($r) { @($r.Data.data) | ForEach-Object { $_.name } }

$all = Api GET "/projects?$listQ"
Check "list: admin lihat 4 project" $all.Data.meta.total 4
$it = @($all.Data.data)[0]
Check "  item: currentPhase Fase 1" $it.currentPhase.name "Fase 1"
Check "  item: currentPhase stage" $it.currentPhase.stage "Dokumen"
Check "  item: pmo array (bukan null)" ($null -ne $it.pmo) $true
Check "  item: developers array" ($null -ne $it.developers) $true
Check "  item: users array" ($null -ne $it.users) $true
Check "  item: priority terisi" ([string]::IsNullOrEmpty($it.priority)) $false
Check "  item: overallProgress 0" $it.overallProgress 0

# filter
Check "list: status Active+Hold (berulang)" (Api GET "/projects?$listQ&status=Active&status=Hold").Data.meta.total 4
Check "list: status Cancelled" (Api GET "/projects?$listQ&status=Cancelled").Data.meta.total 0
$bad = Api GET "/projects?status=Foo"
Check "list: status ngawur" $bad.Status 422
Check "  fields.status" ($null -ne $bad.Err.fields.status) $true
Check "list: directorateId" (Api GET "/projects?$listQ&directorateId=$($dirA.id)").Data.meta.total 4
Check "list: directorateId lain" (Api GET "/projects?$listQ&directorateId=$($dirB.id)").Data.meta.total 0
Check "list: directorateId bukan uuid" (Api GET "/projects?directorateId=abc").Status 422
Check "list: priorityId" (Api GET "/projects?$listQ&priorityId=$($pri.id)").Data.meta.total 4
Check "list: categoryId" (Api GET "/projects?$listQ&categoryId=$($cat.id)").Data.meta.total 4

# pencarian
$byMember = Api GET "/projects?q=$([uri]::EscapeDataString($me.name))&pageSize=100"
Check "list: cari lewat nama anggota PMO" @($byMember.Data.data | Where-Object { $_.id -eq $p2.id }).Count 1
Check "list: q=% tidak dianggap wildcard" (Api GET "/projects?q=%25&pageSize=100").Data.meta.total 0

# sort
Check "list: default = terbaru dulu (C)" (Names (Api GET "/projects?$listQ"))[0] "SMOKE Project C $run"
Check "list: sort=createdAt (A)" (Names (Api GET "/projects?$listQ&sort=createdAt"))[0] "SMOKE Project A $run"
Check "list: sort=name (A)" (Names (Api GET "/projects?$listQ&sort=name"))[0] "SMOKE Project A $run"
Check "list: sort=-name (V)" (Names (Api GET "/projects?$listQ&sort=-name"))[0] "SMOKE Project V $run"
Check "list: sort=-priority" (Api GET "/projects?$listQ&sort=-priority").Status 200
Check "list: sort=overallProgress" (Api GET "/projects?$listQ&sort=overallProgress").Status 200
Check "list: sort ngawur" (Api GET "/projects?sort=ngawur").Status 422

# pagination
$pg = Api GET "/projects?q=$run&sort=name&pageSize=2&page=2"
Check "list: halaman 2 berisi 2 item" @($pg.Data.data).Count 2
Check "  meta.total" $pg.Data.meta.total 4
Check "  meta.page" $pg.Data.meta.page 2
Check "  meta.pageSize" $pg.Data.meta.pageSize 2
Check "  item pertama halaman 2 = C" @($pg.Data.data)[0].name "SMOKE Project C $run"
$far = Api GET "/projects?q=$run&page=99"
Check "list: halaman kelewat -> data [] bukan null" (($null -ne $far.Data.data) -and (@($far.Data.data).Count -eq 0)) $true
Check "  total tetap 4" $far.Data.meta.total 4

# ===== c4: PATCH / status (project Z khusus, SETELAH tes list supaya jumlah di atas tidak terganggu) =====
$zc = Api POST "/projects" (PBody "SMOKE Project Z $run" $subA.id)
Check "c4: create project Z" $zc.Status 201
$zid = $zc.Data.data.id
$ven = @($lk.vendors)[0]
$pri2 = @($lk.projectPriorities)[1]

# --- PATCH ---
$zu = Api PATCH "/projects/$zid" @{ name = "SMOKE Project Z $run EDIT"; crNumber = "CR-$run"; brNumber = "BR-$run"; poNumber = "PO-$run"; poStatus = "Draft"; gitRepo = "git.internal.co/z-$run"; hasMigration = $true }
Check "patch: banyak field sekaligus" $zu.Status 200
Check "  name" $zu.Data.data.name "SMOKE Project Z $run EDIT"
Check "  crNumber" $zu.Data.data.crNumber "CR-$run"
Check "  poStatus" $zu.Data.data.poStatus "Draft"
Check "  hasMigration" $zu.Data.data.hasMigration $true
Check "  gitRepo" $zu.Data.data.gitRepo "git.internal.co/z-$run"
Check "  respons detail utuh (ada fase)" @($zu.Data.data.phases).Count 1
Check "patch: nilai sama -> 200" (Api PATCH "/projects/$zid" @{ name = "SMOKE Project Z $run EDIT" }).Status 200
Check "patch: {} -> 400" (Api PATCH "/projects/$zid" @{}).Status 400
$zs = Api PATCH "/projects/$zid" @{ status = "Hold" }
Check "patch: field status ditolak" $zs.Status 422
Check "  fields.status" ($null -ne $zs.Err.fields.status) $true
Check "patch: field tak dikenal" (Api PATCH "/projects/$zid" @{ createdBy = $zero }).Status 422
Check "patch: name kosong" (Api PATCH "/projects/$zid" @{ name = "  " }).Status 422
Check "patch: name null" (Api PATCH "/projects/$zid" @{ name = $null }).Status 422
Check "patch: poStatus ngawur" (Api PATCH "/projects/$zid" @{ poStatus = "Selesai" }).Status 422
Check "patch: hasMigration tipe salah" (Api PATCH "/projects/$zid" @{ hasMigration = "ya" }).Status 422
Check "patch: id bukan uuid" (Api PATCH "/projects/abc" @{ name = "x" }).Status 404
Check "patch: uuid tak ada" (Api PATCH "/projects/$zero" @{ name = "x" }).Status 404

# nullable: kosongkan
$zn = Api PATCH "/projects/$zid" @{ poNumber = $null; brNumber = "" }
Check "patch: kosongkan poNumber (null) dan brNumber ('')" $zn.Status 200
Check "  poNumber null" ($null -eq $zn.Data.data.poNumber) $true
Check "  brNumber null" ($null -eq $zn.Data.data.brNumber) $true
Check "  crNumber tetap" $zn.Data.data.crNumber "CR-$run"

# direktorat / subdit
$zr = Api PATCH "/projects/$zid" @{ directorateId = $dirB.id }
Check "patch: ganti direktorat tanpa subdit" $zr.Status 422
Check "  fields.subdirectorateId" ($null -ne $zr.Err.fields.subdirectorateId) $true
$zr = Api PATCH "/projects/$zid" @{ subdirectorateId = $subB.id }
Check "patch: subdit milik direktorat lain" $zr.Status 422
Check "  fields.subdirectorateId" ($null -ne $zr.Err.fields.subdirectorateId) $true
$zm = Api PATCH "/projects/$zid" @{ directorateId = $dirB.id; subdirectorateId = $subB.id }
Check "patch: ganti direktorat + subdit" $zm.Status 200
Check "  directorateId" $zm.Data.data.directorateId $dirB.id
Check "  subdirectorateId" $zm.Data.data.subdirectorateId $subB.id
Check "  nama direktorat ikut" $zm.Data.data.directorate $dirB.name

# vendor + priority
$zv = Api PATCH "/projects/$zid" @{ vendorId = $ven.id; priorityId = $pri2.id }
Check "patch: set vendor + priority" $zv.Status 200
Check "  vendor" $zv.Data.data.vendor $ven.name
Check "  priority" $zv.Data.data.priority $pri2.name
Check "patch: vendor tidak ada" (Api PATCH "/projects/$zid" @{ vendorId = $zero }).Status 422
Check "patch: hapus vendor (null)" ($null -eq (Api PATCH "/projects/$zid" @{ vendorId = $null }).Data.data.vendor) $true

# CR unik (tanpa beda huruf besar/kecil)
$z2 = Api POST "/projects" (PBody "SMOKE Project Z2 $run" $subA.id)
$z2id = $z2.Data.data.id
$zd = Api PATCH "/projects/$z2id" @{ crNumber = "cr-$run" }
Check "patch: CR duplikat (case-insensitive)" $zd.Status 409
Check "  code DUPLICATE" $zd.Code "DUPLICATE"
Check "  fields.crNumber" ($null -ne $zd.Err.fields.crNumber) $true

# --- STATUS ---
$zs = Api POST "/projects/$zid/status" @{ status = "Hold" }
Check "status: Hold tanpa alasan" $zs.Status 422
Check "  fields.reason" ($null -ne $zs.Err.fields.reason) $true
Check "status: Hold alasan spasi" (Api POST "/projects/$zid/status" @{ status = "Hold"; reason = "   " }).Status 422
Check "status: nilai ngawur" (Api POST "/projects/$zid/status" @{ status = "Selesai" }).Status 422
Check "status: field tak dikenal" (Api POST "/projects/$zid/status" @{ status = "Hold"; reason = "x"; foo = 1 }).Status 400
$zs = Api POST "/projects/$zid/status" @{ status = "Active" }
Check "status: sama dengan sekarang" $zs.Status 409
Check "  code INVALID_STATE" $zs.Code "INVALID_STATE"

$zh = Api POST "/projects/$zid/status" @{ status = "Hold"; reason = "Menunggu revisi anggaran $run" }
Check "status: Active -> Hold" $zh.Status 200
Check "  status" $zh.Data.data.status "Hold"
Check "  1 riwayat hold" @($zh.Data.data.holdHistory).Count 1
$hh = @($zh.Data.data.holdHistory)[0]
Check "  alasan" $hh.reason "Menunggu revisi anggaran $run"
Check "  heldAt hari ini (WIB)" $hh.heldAt ([DateTime]::UtcNow.AddHours(7).ToString("yyyy-MM-dd"))
Check "  resumedAt null (hold aktif)" ($null -eq $hh.resumedAt) $true
Check "  heldBy = nama user (bukan 'Anda')" (([string]$hh.heldBy).Length -gt 0 -and $hh.heldBy -ne "Anda") $true
Check "  GET detail: status Hold" (Api GET "/projects/$zid").Data.data.status "Hold"
Check "  list status=Hold" (Api GET "/projects?q=$run&status=Hold").Data.meta.total 1
Check "status: Hold -> Hold" (Api POST "/projects/$zid/status" @{ status = "Hold"; reason = "lagi" }).Status 409

$zc2 = Api POST "/projects/$zid/status" @{ status = "Cancelled"; reason = "Dibatalkan sponsor" }
Check "status: Hold -> Cancelled" $zc2.Status 200
$ch = @($zc2.Data.data.holdHistory)[0]
Check "  hold ditutup: resumedAt terisi" ($null -ne $ch.resumedAt) $true
Check "  hold ditutup: resumedBy terisi" ($null -ne $ch.resumedBy) $true
$za = Api POST "/projects/$zid/status" @{ status = "Active" }
Check "status: Cancelled -> Active (boleh dilanjutkan)" $za.Status 200
Check "  riwayat hold tetap 1" @($za.Data.data.holdHistory).Count 1
$zh2 = Api POST "/projects/$zid/status" @{ status = "Hold"; reason = "Hold kedua $run" }
Check "status: Active -> Hold lagi (tidak nabrak unique hold aktif)" $zh2.Status 200
Check "  2 riwayat" @($zh2.Data.data.holdHistory).Count 2
Check "  terbaru dulu" @($zh2.Data.data.holdHistory)[0].reason "Hold kedua $run"
$zb = Api POST "/projects/$zid/status" @{ status = "Active" }
Check "status: Hold -> Active (resume)" $zb.Status 200
Check "  semua hold tertutup" @($zb.Data.data.holdHistory | Where-Object { $null -eq $_.resumedAt }).Count 0
Check "status: id bukan uuid" (Api POST "/projects/abc/status" @{ status = "Hold"; reason = "x" }).Status 404
Check "status: uuid tak ada" (Api POST "/projects/$zero/status" @{ status = "Hold"; reason = "x" }).Status 404

# ===== c4: scope PMO (Z masih hidup, jadi 404 di sini benar-benar karena scope) =====
LoginAs $PmoEmail $DevPassword
Check "pmo: patch project milik admin -> 404" (Api PATCH "/projects/$zid" @{ name = "hack" }).Status 404
Check "pmo: status project milik admin -> 404" (Api POST "/projects/$zid/status" @{ status = "Hold"; reason = "x" }).Status 404
$zp = Api PATCH "/projects/$($p2.id)" @{ gitRepo = "git.internal.co/pmo-$run" }
Check "pmo: patch project sendiri" $zp.Status 200
Check "  gitRepo" $zp.Data.data.gitRepo "git.internal.co/pmo-$run"
Check "pmo: status project sendiri" (Api POST "/projects/$($p2.id)/status" @{ status = "Hold"; reason = "uji scope" }).Status 200
Check "pmo: delete project sendiri -> 403 (tanpa projects.delete)" (Api DELETE "/projects/$($p2.id)").Status 403

# ===== c4: DELETE (soft) =====
LoginAs $AdminEmail $AdminPassword
Check "delete: project Z" (Api DELETE "/projects/$zid").Status 204
Check "delete lagi -> 404" (Api DELETE "/projects/$zid").Status 404
Check "get setelah delete -> 404" (Api GET "/projects/$zid").Status 404
Check "patch setelah delete -> 404" (Api PATCH "/projects/$zid" @{ name = "x" }).Status 404
Check "status setelah delete -> 404" (Api POST "/projects/$zid/status" @{ status = "Hold"; reason = "x" }).Status 404
Check "list: Z tidak muncul (tersisa A, V, B, C, Z2)" (Api GET "/projects?q=$run&pageSize=100").Data.meta.total 5
Check "delete id bukan uuid -> 404" (Api DELETE "/projects/abc").Status 404
Check "delete uuid tak ada -> 404" (Api DELETE "/projects/$zero").Status 404
Check "CR milik project terhapus bisa dipakai lagi" (Api PATCH "/projects/$z2id" @{ crNumber = "CR-$run" }).Status 200

# ===== CLEANUP: hapus (soft) semua project SMOKE milik run ini =====
# Sekaligus menguji DELETE pada project yang punya anggota PMO dan riwayat hold (B).
foreach ($pid2 in @($projA, $projV, $p2.id, $projC, $z2id)) {
    Check "cleanup: delete $pid2" (Api DELETE "/projects/$pid2").Status 204
}
Check "cleanup: tidak ada project SMOKE run ini yang tersisa" (Api GET "/projects?q=$run").Data.meta.total 0

Write-Host ""
if ($script:fail -gt 0) { Write-Host "$($script:fail) tes GAGAL" -ForegroundColor Red; exit 1 }
Write-Host "Semua tes lolos" -ForegroundColor Green