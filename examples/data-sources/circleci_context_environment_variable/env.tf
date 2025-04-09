data "circleci_context_environment_variable" "test_env" {
  context_id = "e51158a2-f59c-4740-9eb4-d20609baa07e"
  name       = "test_sdk"
}

output "circleci_context_environment_variable_created_at" {
  value = data.circleci_context_environment_variable.test_env.created_at
}

output "circleci_context_environment_variable_updated_at" {
  value = data.circleci_context_environment_variable.test_env.updated_at
}

output "circleci_context_environment_variable_name" {
  value = data.circleci_context_environment_variable.test_env.name
}

output "circleci_context_environment_variable_context_id" {
  value = data.circleci_context_environment_variable.test_env.context_id
}
