data "circleci_pipeline" "test_pipeline" {
  id = "something
  project_id = "something"
}

output "circleci_pipeline_id" {
  value = data.circleci_pipeline.test_pipeline.id
}

output "circleci_pipeline_project_id" {
  value = data.circleci_pipeline.test_pipeline.project_id
}

output "circleci_pipeline_name" {
  value = data.circleci_pipeline.test_pipeline.name
}

output "circleci_pipeline_description" {
  value = data.circleci_pipeline.test_pipeline.description
}

output "circleci_pipeline_created_at" {
  value = data.circleci_pipeline.test_pipeline.created_at
}

output "circleci_pipeline_config_source_provider" {
  value = data.circleci_pipeline.test_pipeline.config_source_provider
}

output "circleci_pipeline_config_source_file_path" {
  value = data.circleci_pipeline.test_pipeline.config_source_file_path
}

output "circleci_pipeline_config_source_repo_full_name" {
  value = data.circleci_pipeline.test_pipeline.config_source_repo_full_name
}

output "circleci_pipeline_config_source_repo_external_id" {
  value = data.circleci_pipeline.test_pipeline.config_source_repo_external_id
}

output "circleci_pipeline_checkout_source_provider" {
  value = data.circleci_pipeline.test_pipeline.checkout_source_provider
}

output "circleci_pipeline_checkout_source_repo_full_name" {
  value = data.circleci_pipeline.test_pipeline.checkout_source_repo_full_name
}

output "circleci_pipeline_checkout_source_repo_external_id" {
  value = data.circleci_pipeline.test_pipeline.checkout_source_repo_external_id
}
