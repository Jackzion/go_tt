# run.ps1 —— 启动 3 节点集群并发测试：验证 singleflight 防缓存击穿
#
# 用法：
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\run.ps1
#
# 效果：向 API(9999) 并发发起 10 个 ?key=Tom 请求，
#       日志中 [8003] Pick peer 与 [8001] GET /_geecache 各只出现 1 次，
#       说明 10 个并发请求被 singleflight 合并为 1 次远程取数。

$ErrorActionPreference = 'Stop'
$server = Join-Path $PSScriptRoot 'server.exe'

function Cleanup {
    Get-Process -Name 'server' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Remove-Item $server -ErrorAction SilentlyContinue
    Write-Host ''
    Write-Host '>>> cleanup done'
}

try {
    # ---------- 1) 编译 ----------
    Write-Host '>>> building server ...'
    go build -o $server ./geecache/main
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }

    # 清理上次的日志
    Remove-Item .\node*.log, .\curl*.out, .\curl*.err -ErrorAction SilentlyContinue

    # ---------- 2) 启动 3 个缓存节点（8003 同时挂 API 服务器）----------
    foreach ($port in 8001, 8002, 8003) {
        $nodeArgs = @("-port=$port")
        if ($port -eq 8003) { $nodeArgs += '-api=1' }
        Start-Process -FilePath $server -ArgumentList $nodeArgs `
            -RedirectStandardOutput ".\node$port.out.log" `
            -RedirectStandardError  ".\node$port.err.log" | Out-Null
    }
    Start-Sleep -Seconds 2
    Write-Host '>>> 3 cache nodes started (8001, 8002, 8003+api)'

    # ---------- 3) 并发 10 个请求 ----------
    Write-Host '>>> start test: 10 concurrent requests -> http://localhost:9999/api?key=Tom'
    $procs = foreach ($i in 1..10) {
        Start-Process -FilePath 'curl.exe' `
            -ArgumentList '"http://localhost:9999/api?key=Tom"' `
            -NoNewWindow -PassThru `
            -RedirectStandardOutput ".\curl$i.out" `
            -RedirectStandardError  ".\curl$i.err"
    }
    $procs | Wait-Process

    Write-Host '>>> responses:'
    foreach ($i in 1..10) {
        $r = Get-Content ".\curl$i.out" -Raw -ErrorAction SilentlyContinue
        if ($r) { Write-Host ('  curl#{0,-2} -> {1}' -f $i, $r.Trim()) }
    }

    # ---------- 4) 停止节点后读取日志（避免 Windows 文件共享冲突）----------
    Get-Process -Name 'server' -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep -Milliseconds 300

    Write-Host ''
    Write-Host '>>> [8003 = API 节点] Pick peer 日志（应只有 1 条，说明 10 个并发被合并）:'
    Select-String -Path '.\node8003.err.log' -Pattern 'Pick peer' -ErrorAction SilentlyContinue |
        ForEach-Object { '  ' + $_.Line }

    Write-Host '>>> [8001 = Tom 归属节点] 转发请求与回源日志（GET 与 SlowDB 各应只有 1 条）:'
    Select-String -Path '.\node8001.err.log' -Pattern 'GET /_geecache|SlowDB' -ErrorAction SilentlyContinue |
        ForEach-Object { '  ' + $_.Line }
}
finally {
    Cleanup
}
