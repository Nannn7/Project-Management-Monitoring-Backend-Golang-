$script:token = $null
$script:fail = 0
$script:Base = "http://localhost:8080/api/v1"

function Api($method, $path, $body = $null) {
  $p = @{ Method = $method; Uri = "$($script:Base)$path"; ContentType = "application/json"; Headers = @{}; UseBasicParsing = $true }
  if ($script:token) { $p.Headers.Authorization = "Bearer $($script:token)" }
  if ($null -ne $body) { $p.Body = ($body | ConvertTo-Json) }
  try {
    $r = Invoke-WebRequest @p
    $data = $null
    if ($r.Content) { $data = $r.Content | ConvertFrom-Json }
    return @{ Status = [int]$r.StatusCode; Data = $data; Code = $null; Err = $null }
  }
  catch {
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