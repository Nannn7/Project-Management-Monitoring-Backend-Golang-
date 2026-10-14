param(
  [string]$Base = "http://localhost:8080/api/v1",
  [Parameter(Mandatory)][string]$Email,
  [string]$Password = $env:SMOKE_PASSWORD
)

if (-not $Password) {
  $Password = [System.Net.NetworkCredential]::new("", (Read-Host "Password" -AsSecureString)).Password
}

function Api($method, $path, $body, $token) {
  $p = @{ Method = $method; Uri = "$Base$path"; ContentType = "application/json"; Headers = @{} }
  if ($token) { $p.Headers.Authorization = "Bearer $token" }
  if ($body) { $p.Body = ($body | ConvertTo-Json) }
  try { return @{ Status = 200; Data = (Invoke-RestMethod @p) } }
  catch { return @{ Status = [int]$_.Exception.Response.StatusCode; Data = $null } }
}

function Check($name, $got, $want) {
  if ($got -eq $want) { Write-Host "PASS  $name ($got)" -ForegroundColor Green }
  else { Write-Host "FAIL  $name (got $got, want $want)" -ForegroundColor Red }
}

$login = Api POST "/auth/login" @{ email = $Email; password = $Password }
Check "login akun asli" $login.Status 200
if ($login.Status -ne 200) { Write-Host "STOP: login gagal, sisa tes nggak valid."; exit 1 }
$access = $login.Data.data.accessToken
$refresh = $login.Data.data.refreshToken

Check "login password salah" (Api POST "/auth/login" @{ email = $Email; password = "salah-banget-123" }).Status 401
Check "me tanpa token"             (Api GET "/auth/me").Status 401
Check "me token sampah"            (Api GET "/auth/me" $null "sampah").Status 401
Check "me pakai access (kontrol)"  (Api GET "/auth/me" $null $access).Status 200
Check "me pakai refresh"           (Api GET "/auth/me" $null $refresh).Status 401
Check "refresh pakai access"       (Api POST "/auth/refresh" @{ refreshToken = $access }).Status 401
Check "refresh sampah"             (Api POST "/auth/refresh" @{ refreshToken = "sampah" }).Status 401
$r = Api POST "/auth/refresh" @{ refreshToken = $refresh }
Check "refresh sah"                $r.Status 200
Check "me pakai access baru"       (Api GET "/auth/me" $null $r.Data.data.accessToken).Status 200
Check "login email tidak ada" (Api POST "/auth/login" @{ email = "nggakada@x.com"; password = "salah-banget-123" }).Status 401