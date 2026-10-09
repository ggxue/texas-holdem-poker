# Build-time only: ship the recorded Mandarin voice with the Go executable.
param([string[]]$Only = @())
$ErrorActionPreference = 'Stop'
$taskVoice = New-Object -ComObject SAPI.SpVoice
$taskToken = @($taskVoice.GetVoices()) | Where-Object { $_.GetDescription() -match 'Huihui' } | Select-Object -First 1
if (-not $taskToken) { throw 'Microsoft Huihui Chinese voice is required.' }
$taskVoice.Voice = $taskToken
$taskVoice.Rate = 1
$taskVoice.Volume = 100
$taskOutput = Join-Path $PSScriptRoot '../internal/poker/web/audio'
New-Item -ItemType Directory -Force -Path $taskOutput | Out-Null
$taskPhrases = [ordered]@{
 enabled='声音已启用。'; check='过牌。'; bet='下注'; call='跟注'; fold='弃牌。'; allin='全押。'; 'allin-action'='全押'; timeout='超时，自动'; chips='筹码。'; winner='获得'
 start='开始新一局。'; deal='已发手牌，每人底注一枚。'; flop='翻牌。'; turn='转牌。'; river='河牌。'; showdown='开始摊牌。'; settlement='本局结束，开始派奖。'; thinking='机器人思考中。'
 join='进入房间。'; leave='离开房间。'; disconnect='掉线，等待重连。'; return='已回来。'; host='成为房主。'
 error='暂时无法完成操作，请查看屏幕提示。'; connection='连接中断，请重新连接。'; roomfull='房间已满，请稍后再试。'; takeover='已在其他页面打开，请使用新页面。'
 reminder='轮到你出牌。'; urgent='请尽快行动，还剩十秒。'
 'seat-0'='玩家一'; 'seat-1'='玩家二'; 'seat-2'='玩家三'; 'seat-3'='玩家四'; 'seat-4'='玩家五'; 'seat-5'='机器人'
 'n-0'='零'; 'n-1'='一'; 'n-2'='二'; 'n-3'='三'; 'n-4'='四'; 'n-5'='五'; 'n-6'='六'; 'n-7'='七'; 'n-8'='八'; 'n-9'='九'; 'n-ten'='十'; 'n-hundred'='百'; 'n-thousand'='千'; 'n-wan'='万'; 'n-yi'='亿'; 'n-zhao'='兆'; 'n-jing'='京'
}
for ($taskSeconds = 1; $taskSeconds -le 30; $taskSeconds++) {
 $taskPhrases["remaining-$taskSeconds"] = "轮到你出牌，还剩$taskSeconds 秒。"
}
$taskRecorded = 0
foreach ($taskPair in $taskPhrases.GetEnumerator()) {
 if ($Only.Count -gt 0 -and $taskPair.Key -notin $Only) { continue }
 $taskStream = New-Object -ComObject SAPI.SpFileStream
 try {
  $taskStream.Format.Type = 22
  $taskStream.Open((Join-Path $taskOutput ($taskPair.Key + '.wav')), 3, $false)
  $taskVoice.AudioOutputStream = $taskStream
  [void]$taskVoice.Speak($taskPair.Value)
  $taskRecorded++
 } finally {
  $taskStream.Close()
  [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($taskStream)
 }
}
[void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($taskVoice)
$taskManifest = [ordered]@{ voice='Microsoft Huihui'; locale='zh-CN'; rate=1; phrases=$taskPhrases }
[IO.File]::WriteAllText((Join-Path $taskOutput 'manifest.json'), ($taskManifest | ConvertTo-Json -Depth 3), [Text.UTF8Encoding]::new($false))
Write-Output "Recorded $taskRecorded Mandarin WAV samples; manifest contains $($taskPhrases.Count) phrases."
