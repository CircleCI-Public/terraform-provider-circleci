locals {
  organization_id = "00000000-0000-0000-0000-000000000000"
  project_id      = "11111111-1111-1111-1111-111111111111"
}

data "circleci_notification_preferences" "example" {
  organization_id = local.organization_id
  project_id      = local.project_id
}

locals {
  # Look preferences up by section, group and name instead of hard-coding IDs.
  wanted = toset([
    "Build/Workflow Completed/On default branch (failed, error, unauthorized)",
    "Build/Job/Job awaiting approval",
    "Deployments/Releases/Failed",
  ])
}

# Every user-configurable preference not listed here is turned off.
resource "circleci_project_notification_preferences" "example" {
  organization_id = local.organization_id
  project_id      = local.project_id

  enabled_preference_ids = [
    for p in data.circleci_notification_preferences.example.preferences : p.id
    if contains(local.wanted, "${p.section_name}/${p.group_name}/${p.name}")
  ]
}
