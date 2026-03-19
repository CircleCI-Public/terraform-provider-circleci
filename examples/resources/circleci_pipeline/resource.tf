resource "circleci_pipeline" "example" {
  project_id  = circleci_project.example.id
  name        = "my-pipeline"
  description = "Main CI/CD pipeline"

  config_source_provider         = "github_app"
  config_source_file_path        = ".circleci/config.yml"
  config_source_repo_external_id = "123456789"

  checkout_source_provider         = "github_app"
  checkout_source_repo_external_id = "123456789"
}
