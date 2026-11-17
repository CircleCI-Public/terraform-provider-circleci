# Only the settings set here are written. Every other setting keeps the value
# it has on CircleCI.
resource "circleci_organization_settings" "example" {
  organization_id = "00000000-0000-0000-0000-000000000000"

  enable_uncertified_public_orbs        = false
  enable_private_orbs                   = true
  is_context_group_restriction_required = true
}
