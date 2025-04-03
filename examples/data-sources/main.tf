terraform {
  required_providers {
    circleci = {
      source = "registry.terraform.io/circleci/circleci"
    }
  }
}

data "circleci_project" "test_project" {
  slug = "circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"
}

output "circleci_project_id" {
  value = data.circleci_project.test_project.id
}

output "circleci_project_slug" {
  value = data.circleci_project.test_project.slug
}

output "circleci_project_name" {
  value = data.circleci_project.test_project.name
}

output "circleci_project_auto_cancel_builds" {
  value = data.circleci_project.test_project.auto_cancel_builds
}

output "circleci_project_build_fork_prs" {
  value = data.circleci_project.test_project.build_fork_prs
}
output "circleci_project_disable_ssh" {
  value = data.circleci_project.test_project.disable_ssh
}
output "circleci_project_forks_receive_secret_env_vars" {
  value = data.circleci_project.test_project.forks_receive_secret_env_vars
}
output "circleci_project_oss" {
  value = data.circleci_project.test_project.oss
}
output "circleci_project_set_github_status" {
  value = data.circleci_project.test_project.set_github_status
}
output "circleci_project_setup_workflows" {
  value = data.circleci_project.test_project.setup_workflows
}
output "circleci_project_write_settings_requires_admin" {
  value = data.circleci_project.test_project.write_settings_requires_admin
}
output "circleci_project_pr_only_branch_overrides" {
  value = data.circleci_project.test_project.pr_only_branch_overrides
}