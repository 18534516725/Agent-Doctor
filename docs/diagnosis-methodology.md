# Diagnosis methodology

A diagnosis contains a code, severity, explanation, supporting evidence,
counter-evidence, provenance, precision, and limitations. It is not generated
from a single slow event.

## Evidence order

1. normalized client lifecycle event;
2. Git state captured for the same session;
3. explicitly approved validation result;
4. exact or estimated usage/cost evidence;
5. matched historical baseline.

Rules require minimum sample sizes and deterministic thresholds. A personal
baseline uses comparable tasks, median and P90 rather than a global claim.
Comparisons match project, task type, and major client/model version; fewer than
15 samples per cohort remains low confidence. The engine never automatically
declares a client or model “best.”

Controlled replay is optional. The preview exposes client, model, base SHA,
sanitized task, commands, call/cost limits, cleanup behavior, and a consent hash.
The current branch is never checked out or modified.
