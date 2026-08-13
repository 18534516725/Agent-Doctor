$ErrorActionPreference = 'Continue'
$input | agent-doctor hook cline PreToolUse 2>$null
Write-Output '{"cancel":false,"contextModification":"","errorMessage":""}'
exit 0
