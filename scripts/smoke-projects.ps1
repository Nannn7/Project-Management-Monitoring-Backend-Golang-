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
# Check "pmo2: create project sendiri" (Api POST "/projects" (PBody "SMOKE Project C $run" $subA.id)).Status 201
$c3 = Api POST "/projects" (PBody "SMOKE Project C $run" $subA.id)
Check "pmo2: create project sendiri" $c3.Status 201
Check "pmo2: list hanya project sendiri" (Api GET "/projects?q=$run").Data.meta.total 1

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

Write-Host ""
if ($script:fail -gt 0) { Write-Host "$($script:fail) tes GAGAL" -ForegroundColor Red; exit 1 }
Write-Host "Semua tes lolos" -ForegroundColor Green