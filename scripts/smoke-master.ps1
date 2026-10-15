param(
  [string]$Base = "http://localhost:8080/api/v1",
  [Parameter(Mandatory)][string]$Email,
  [string]$Password = $env:SMOKE_PASSWORD
)
if (-not $Password) {
  $Password = [System.Net.NetworkCredential]::new("", (Read-Host "Password" -AsSecureString)).Password
}

$script:token = $null
$script:fail = 0

function Api($method, $path, $body = $null) {
  $p = @{ Method = $method; Uri = "$Base$path"; ContentType = "application/json"; Headers = @{}; UseBasicParsing = $true }
  if ($script:token) { $p.Headers.Authorization = "Bearer $($script:token)" }
  if ($null -ne $body) { $p.Body = ($body | ConvertTo-Json) }
  try {
    $r = Invoke-WebRequest @p
    $data = $null
    if ($r.Content) { $data = $r.Content | ConvertFrom-Json }
    return @{ Status = [int]$r.StatusCode; Data = $data; Code = $null; Err = $null }
  } catch {
    $msg = $_.ErrorDetails.Message
    if (-not $msg -and $_.Exception.Response) {
      try { $msg = (New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())).ReadToEnd() } catch {}
    }
    $err = $null
    if ($msg) { try { $err = ($msg | ConvertFrom-Json).error } catch {} }
    $code = $null
    if ($err) { $code = $err.code }
    return @{ Status = [int]$_.Exception.Response.StatusCode; Data = $null; Code = $code; Err = $err }
  }
}

function Check($name, $got, $want) {
  if ($got -eq $want) { Write-Host "PASS  $name ($got)" -ForegroundColor Green }
  else { Write-Host "FAIL  $name (got $got, want $want)" -ForegroundColor Red; $script:fail++ }
}

# --- login ---
$login = Api POST "/auth/login" @{ email = $Email; password = $Password }
Check "login" $login.Status 200
if ($login.Status -ne 200) { Write-Host "STOP: login gagal."; exit 1 }
$script:token = $login.Data.data.accessToken
$run = Get-Date -Format "HHmmss"
$zero = "00000000-0000-0000-0000-000000000000"

# --- auth guard ---
$saved = $script:token; $script:token = $null
Check "tanpa token -> 401" (Api GET "/master/vendors").Status 401
$script:token = $saved

# --- read: 12 resource semua harus 200 (document-types & test-types sebelumnya error SQL) ---
foreach ($res in "directorates","subdirectorates","vendors","project-categories","project-priorities","approval-levels","project-roles","task-categories","meeting-categories","dependency-systems","document-types","test-types") {
  Check "list $res" (Api GET "/master/${res}?pageSize=5").Status 200
}
Check "resource ngawur -> 404" (Api GET "/master/ngawur").Status 404
Check "list + sort + q + isActive" (Api GET "/master/vendors?sort=-name&q=a&isActive=true").Status 200

# --- CRUD vendor ---
$name = "SMOKE Vendor $run"
$c = Api POST "/master/vendors" @{ name = $name }
Check "create vendor" $c.Status 201
$id = $c.Data.data.id
Check "sortOrder terisi otomatis" ([int]$c.Data.data.sortOrder -gt 0) $true
Check "create duplikat persis" (Api POST "/master/vendors" @{ name = $name }).Status 409
$dup = Api POST "/master/vendors" @{ name = $name.ToLower() }
Check "create duplikat beda huruf" $dup.Status 409
Check "  code = DUPLICATE" $dup.Code "DUPLICATE"
Check "create tanpa name" (Api POST "/master/vendors" @{}).Status 422
Check "create field ngawur" (Api POST "/master/vendors" @{ name = "x $run"; foo = "bar" }).Status 422

$u = Api PATCH "/master/vendors/$id" @{ name = "$name EDIT" }
Check "update name" $u.Status 200
Check "  name berubah" $u.Data.data.name "$name EDIT"
Check "nonaktifkan" (Api PATCH "/master/vendors/$id" @{ isActive = $false }).Status 200
$q = [uri]::EscapeDataString("SMOKE Vendor $run")
Check "filter isActive=false ketemu" (Api GET "/master/vendors?isActive=false&q=$q").Data.meta.total 1
Check "patch kosong -> 400" (Api PATCH "/master/vendors/$id" @{}).Status 400
Check "patch id bukan uuid -> 404" (Api PATCH "/master/vendors/abc" @{ name = "x" }).Status 404
Check "patch uuid tak ada -> 404" (Api PATCH "/master/vendors/$zero" @{ name = "x" }).Status 404
Check "patch tipe salah -> 422" (Api PATCH "/master/vendors/$id" @{ sortOrder = "dua" }).Status 422
Check "delete vendor" (Api DELETE "/master/vendors/$id").Status 204
Check "delete lagi -> 404" (Api DELETE "/master/vendors/$id").Status 404

# --- IN_USE (bikin direktorat + subdit sendiri, jadi nggak butuh data project) ---
$dir = Api POST "/master/directorates" @{ name = "SMOKE Dir $run" }
Check "create direktorat" $dir.Status 201
$dirId = $dir.Data.data.id
$sub = Api POST "/master/subdirectorates" @{ name = "SMOKE Sub $run"; directorateId = $dirId }
Check "create subdit" $sub.Status 201
$subId = $sub.Data.data.id
Check "  respons subdit ada nama direktorat" $sub.Data.data.directorate "SMOKE Dir $run"
Check "subdit direktorat tak ada -> 422" (Api POST "/master/subdirectorates" @{ name = "y $run"; directorateId = $zero }).Status 422
$inUse = Api DELETE "/master/directorates/$dirId"
Check "hapus direktorat yg punya subdit" $inUse.Status 409
Check "  code = IN_USE" $inUse.Code "IN_USE"
$ref = @($inUse.Err.details.references)[0]
Check "  detail: entity" $ref.entity "subdirectorates"
Check "  detail: count" $ref.count 1
Check "hapus subdit" (Api DELETE "/master/subdirectorates/$subId").Status 204
Check "hapus direktorat (sekarang kosong)" (Api DELETE "/master/directorates/$dirId").Status 204

# --- baris sistem ---
$pmo = @((Api GET "/master/project-roles?q=PMO&pageSize=50").Data.data) | Where-Object { $_.code -eq "PMO" } | Select-Object -First 1
if ($pmo) {
  $x = Api DELETE "/master/project-roles/$($pmo.id)"
  Check "hapus role sistem PMO" $x.Status 409
  Check "  code = INVALID_STATE" $x.Code "INVALID_STATE"
  Check "ubah code role -> 422" (Api PATCH "/master/project-roles/$($pmo.id)" @{ code = "XXX" }).Status 422
} else { Write-Host "SKIP  role PMO nggak ketemu" -ForegroundColor Yellow }

$other = @((Api GET "/master/dependency-systems?q=Lainnya").Data.data) | Where-Object { $_.isOther } | Select-Object -First 1
if ($other) {
  Check "hapus 'Lainnya'" (Api DELETE "/master/dependency-systems/$($other.id)").Status 409
  Check "rename 'Lainnya'" (Api PATCH "/master/dependency-systems/$($other.id)" @{ name = "Other" }).Status 409
} else { Write-Host "SKIP  'Lainnya' nggak ketemu" -ForegroundColor Yellow }

# --- resource berbasis code ---
$dt = Api POST "/master/document-types" @{ code = "SM$run"; description = "Smoke test" }
Check "create document-type" $dt.Status 201
Check "  nggak ada kolom name" ($null -eq $dt.Data.data.name) $true
Check "delete document-type" (Api DELETE "/master/document-types/$($dt.Data.data.id)").Status 204
$tt = Api POST "/master/test-types" @{ code = "T$run"; requiresPrevious = $true }
Check "create test-type" $tt.Status 201
Check "  requiresPrevious" $tt.Data.data.requiresPrevious $true
Check "delete test-type" (Api DELETE "/master/test-types/$($tt.Data.data.id)").Status 204

# --- admin bypass (b0): /employees dulu 'terbuka buat semua', sekarang butuh permission ---
Check "GET /employees (admin)" (Api GET "/employees").Status 200

Write-Host ""
if ($script:fail -gt 0) { Write-Host "$($script:fail) tes GAGAL" -ForegroundColor Red; exit 1 }
Write-Host "Semua tes lolos" -ForegroundColor Green