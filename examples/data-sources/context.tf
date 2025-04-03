data "circleci_context" "test_context" {
  id = "e51158a2-f59c-4740-9eb4-d20609baa07e"
}

output "circleci_context_id" {
  value = data.circleci_context.test_context.id
}

output "circleci_context_name" {
  value = data.circleci_context.test_context.name
}

output "circleci_context_created_at" {
  value = data.circleci_context.test_context.created_at
}

output "circleci_context_restrictions" {
  value = data.circleci_context.test_context.restrictions
}
