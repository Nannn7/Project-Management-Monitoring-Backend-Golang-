param(
  [string]$Base = "http://localhost:8080/api/v1",
  [Parameter(Mandatory)][string]$Email,
  [string]$Password = $env:SMOKE_PASSWORD
)
. "$PSScriptRoot\lib.ps1"
$script:Base = $Base
if (-not $Password) {
  $Password = [System.Net.NetworkCredential]::new("", (Read-Host "Password" -AsSecureString)).Password
}

$login = Api POST "/auth/login" @{ email = $Email; password = $Password }
Check "login" $login.Status 200
if ($login.Status -ne 200) { exit 1 }
$script:token = $login.Data.data.accessToken
$run = Get-Date -Format "HHmmss"
$zero = "00000000-0000-0000-0000-000000000000"

# data referensi dari seed lewat /lookups
$lk = (Api GET "/lookups").Data.data
$dirs = @($lk.directorates)
$dirA = $dirs[0]; $dirB = $dirs[1]
$subA = @($lk.subdirectorates) | Where-Object { $_.directorateId -eq $dirA.id } | Select-Object -First 1
$subB = @($lk.subdirectorates) | Where-Object { $_.directorateId -eq $dirB.id } | Select-Object -First 1
$pmo  = @($lk.projectRoles) | Where-Object { $_.code -eq "PMO" } | Select-Object -First 1
$dev  = @($lk.projectRoles) | Where-Object { $_.code -eq "DEVELOPER" } | Select-Object -First 1
if (-not ($dirA -and $dirB -and $subA -and $subB -and $pmo -and $dev)) { Write-Host "STOP: data seed nggak lengkap"; exit 1 }

$name = "SMOKE Emp $run"
$ghost = "SMOKE Ghost $run"
function Body($n, $sub, $roles) { @{ name = $n; directorateId = $dirA.id; subdirectorateId = $sub; roleIds = $roles } }

# --- create ---
$c = Api POST "/employees" (Body $name $subA.id @($pmo.id))
Check "create pegawai" $c.Status 201
$id = $c.Data.data.id
Check "  roles = PMO" (@($c.Data.data.roles) -join ",") "PMO"
Check "  userId null (bukan dari sini)" ($null -eq $c.Data.data.userId) $true

$r = Api POST "/employees" @{ name = $ghost; directorateId = $dirA.id; subdirectorateId = $subA.id }
Check "create tanpa roleIds" $r.Status 422
Check "  fields.roleIds" ($null -ne $r.Err.fields.roleIds) $true
Check "create roleIds kosong" (Api POST "/employees" (Body $ghost $subA.id @())).Status 422
Check "create roleIds bukan uuid" (Api POST "/employees" (Body $ghost $subA.id @("abc"))).Status 422
$r = Api POST "/employees" (Body $ghost $subB.id @($pmo.id))
Check "create subdit milik direktorat lain" $r.Status 422
Check "  fields.subdirectorateId" ($null -ne $r.Err.fields.subdirectorateId) $true
$r = Api POST "/employees" (Body $ghost $subA.id @($zero))
Check "create role tidak ada" $r.Status 422
Check "  fields.roleIds" ($null -ne $r.Err.fields.roleIds) $true
Check "  ATOMIK: pegawai ghost tidak tersisa" (Api GET "/employees?q=$([uri]::EscapeDataString($ghost))").Data.meta.total 0
Check "create field tak dikenal (userId)" (Api POST "/employees" @{ name = $ghost; directorateId = $dirA.id; subdirectorateId = $subA.id; roleIds = @($pmo.id); userId = $zero }).Status 400

# --- list ---
$q = [uri]::EscapeDataString($name)
Check "list q + role=PMO" (Api GET "/employees?q=$q&role=PMO").Data.meta.total 1
Check "list q + role=DEVELOPER" (Api GET "/employees?q=$q&role=DEVELOPER").Data.meta.total 0
Check "list directorateId ngawur" (Api GET "/employees?directorateId=abc").Status 422
Check "list sort=-name" (Api GET "/employees?sort=-name&pageSize=5").Status 200

# --- update ---
$p = Api PATCH "/employees/$id" @{ roleIds = @($pmo.id, $dev.id, $dev.id) }
Check "patch roleIds (dedupe)" $p.Status 200
Check "  jumlah role = 2" @($p.Data.data.roles).Count 2
Check "patch roleIds []" (Api PATCH "/employees/$id" @{ roleIds = @() }).Status 422
Check "patch {}" (Api PATCH "/employees/$id" @{}).Status 400
Check "patch uuid tak ada" (Api PATCH "/employees/$zero" @{ name = "x" }).Status 404
Check "patch id bukan uuid" (Api PATCH "/employees/abc" @{ name = "x" }).Status 404
Check "patch subdit beda direktorat" (Api PATCH "/employees/$id" @{ subdirectorateId = $subB.id }).Status 422
Check "patch name" (Api PATCH "/employees/$id" @{ name = "$name EDIT" }).Data.data.name "$name EDIT"
Check "patch isActive=false" (Api PATCH "/employees/$id" @{ isActive = $false }).Status 200
Check "list isActive=true tidak memuat dia" (Api GET "/employees?isActive=true&q=$q").Data.meta.total 0

# --- delete ---
Check "delete pegawai" (Api DELETE "/employees/$id").Status 204
Check "delete lagi" (Api DELETE "/employees/$id").Status 404

Write-Host ""
if ($script:fail -gt 0) { Write-Host "$($script:fail) tes GAGAL" -ForegroundColor Red; exit 1 }
Write-Host "Semua tes lolos" -ForegroundColor Green