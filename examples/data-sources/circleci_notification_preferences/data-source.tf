data "circleci_notification_preferences" "example" {
  organization_id = "00000000-0000-0000-0000-000000000000"
  project_id      = "11111111-1111-1111-1111-111111111111"
}

output "notification_preferences" {
  value = {
    for p in data.circleci_notification_preferences.example.preferences :
    "${p.section_name}/${p.group_name}/${p.name}" => p.id
    if p.is_user_configurable
  }
}
