# Look up a context by its ID.
data "circleci_context" "test_context" {
  id = "e51158a2-f59c-4740-9eb4-d20609baa07e"
}

# Or look up a context by its name within an organization.
data "circleci_context" "production" {
  organization_id = "c5e1ac37-0f5b-4c4e-8f2a-8a5c3f6c1d2e"
  name            = "production"
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

output "production_context_id" {
  value = data.circleci_context.production.id
}
