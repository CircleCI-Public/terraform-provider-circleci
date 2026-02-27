resource "circleci_project" "actions" {
  name            = "example"
  organization_id = var.org_id

  # Build settings
  auto_cancel_builds = true
  setup_workflows    = true

  # GitHub integration
  set_github_status = true

  # Security
  disable_ssh                   = false
  forks_receive_secret_env_vars = false
  build_fork_prs                = false
}