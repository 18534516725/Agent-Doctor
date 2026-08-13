$ErrorActionPreference = 'Continue'
$input | agent-doctor hook cline PostToolUse 2>$null
Write-Output '{"cancel":false,"contextModification":"","errorMessage":""}'
exit 0
