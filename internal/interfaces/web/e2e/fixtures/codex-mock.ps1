param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$Args
)

function Resolve-MockThreadId {
  param(
    [string]$Prompt
  )

  $hash = [Math]::Abs($Prompt.GetHashCode())
  return "mock-thread-$hash"
}

function Resolve-MockReply {
  param(
    [string]$Prompt
  )

  $trimmed = ""
  if (-not [string]::IsNullOrWhiteSpace($Prompt)) {
    $trimmed = $Prompt.Trim()
  }
  if ($trimmed -match '^(?i)reply with exactly:\s*(.+)$') {
    return $Matches[1].Trim()
  }
  if ($trimmed -match '^(?i)respond with exactly:\s*(.+)$') {
    return $Matches[1].Trim()
  }
  if ($trimmed -match '^(?i)output\s+(\d+)\s+lines$') {
    $count = [Math]::Min([Math]::Max([int]$Matches[1], 1), 200)
    return ((1..$count) | ForEach-Object { "line $_" }) -join "`n"
  }
  if ($trimmed -eq "") {
    return "mock-empty"
  }
  return "mock: $trimmed"
}

function Write-JsonLine {
  param(
    [hashtable]$Value
  )

  [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
  $Value | ConvertTo-Json -Compress -Depth 8
}

$forwarded = @($Args | Where-Object { $_ -ne "" })
if ($forwarded.Length -lt 2 -or $forwarded[0] -ne "exec") {
  Write-Error "unsupported mock invocation"
  exit 2
}

$prompt = ""
$threadID = ""
$isResume = $false

if ($forwarded.Length -ge 3 -and $forwarded[1] -eq "resume") {
  $isResume = $true
  $resumeArgs = @($forwarded[2..($forwarded.Length - 1)])
  $threadIndex = 0
  while ($threadIndex -lt $resumeArgs.Length -and $resumeArgs[$threadIndex].StartsWith("-")) {
    $option = $resumeArgs[$threadIndex]
    if ($option -in @("--json", "--skip-git-repo-check")) {
      $threadIndex += 1
      continue
    }
    break
  }
  if ($threadIndex -ge $resumeArgs.Length) {
    Write-Error "missing thread id"
    exit 2
  }
  $threadID = $resumeArgs[$threadIndex]
  $promptIndex = $resumeArgs.Length - 1
  $prompt = $resumeArgs[$promptIndex]
} else {
  $prompt = $forwarded[$forwarded.Length - 1]
  $threadID = Resolve-MockThreadId -Prompt $prompt
}

$reply = Resolve-MockReply -Prompt $prompt
$workingDir = (Get-Location).Path
$commandOutput = @(
  "WorkingDirectory"
  "---------------"
  $workingDir
) -join "`n"

Write-JsonLine @{ type = "thread.started"; thread_id = $threadID }
Write-JsonLine @{ type = "turn.started" }
Write-JsonLine @{
  type = "item.completed"
  item = @{
    id = "item_reasoning"
    type = "reasoning"
    text = if ($isResume) { "Resume prior Codex CLI context." } else { "Inspect workspace and prepare response." }
  }
}
Write-JsonLine @{
  type = "item.started"
  item = @{
    id = "item_command"
    type = "command_execution"
    command = "pwd"
  }
}
Write-JsonLine @{
  type = "item.completed"
  item = @{
    id = "item_command"
    type = "command_execution"
    command = "pwd"
    aggregated_output = $commandOutput
    status = "completed"
    exit_code = 0
  }
}
Write-JsonLine @{
  type = "item.completed"
  item = @{
    id = "item_message"
    type = "agent_message"
    text = $reply
  }
}
Write-JsonLine @{
  type = "turn.completed"
  usage = @{
    input_tokens = 1
    cached_input_tokens = 0
    output_tokens = 1
  }
}
exit 0
