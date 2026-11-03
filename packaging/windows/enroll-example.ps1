# This template is deliberately non-mutating. Review paths, service identity,
# privileges and policy before converting it into an installation script.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$ServiceName,
    [Parameter(Mandatory = $true)][string]$PolicyPath
)

$ErrorActionPreference = 'Stop'
if (-not [System.IO.Path]::IsPathRooted($PolicyPath)) {
    throw 'An absolute protected policy path is required.'
}

Write-Output 'Enrollment checklist; no system changes were made.'
Write-Output ('Service: ' + $ServiceName)
Write-Output '1. Use a dedicated native service account and enable its service SID.'
Write-Output '2. Protect executable, policy, identity and state ACLs against application writes.'
Write-Output '3. Run cquay doctor and plan in the actual service identity.'
Write-Output '4. Enroll exact source/disclosure grants and seed approved offline material.'
Write-Output '5. Register an SCM-capable service wrapper; cquay run is a console executable.'
Write-Output '6. Register a boot/periodic Task Scheduler action under the chosen identity.'
Write-Output '7. Prove no-login reboot, offline behavior, rotation and restoration.'
Write-Output 'DPAPI LocalMachine requires NTFS ACL isolation; it does not isolate a copied blob.'
