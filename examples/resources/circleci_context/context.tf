resource "circleci_context" "test_context" {
  name            = "terraform_created_context_updated"
  organization_id = "3ddcf1d1-7f5f-4139-8cef-71ad0921a968"
}

output "circleci_context_id" {
  value = circleci_context.test_context.id
}

output "circleci_context_name" {
  value = circleci_context.test_context.name
}

output "circleci_context_created_at" {
  value = circleci_context.test_context.created_at
}
