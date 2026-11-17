# The organization must have the CircleCI Slack App installed. CircleCI posts a
# welcome message into the channel on create, and on every update while enabled.
resource "circleci_project_slack_notification" "example" {
  organization_id = "00000000-0000-0000-0000-000000000000"
  project_id      = "11111111-1111-1111-1111-111111111111"

  # The Slack channel ID, not its name.
  channel = "C0123456789"
  enabled = true
}
