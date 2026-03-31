# Schedule trigger — runs the pipeline every day at 01:00 UTC
resource "circleci_trigger" "nightly" {
  project_id            = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
  pipeline_id           = "fefb451c-9966-4b75-b555-d4d94d7116ef"
  event_source_provider = "schedule"
  event_name            = "Nightly build"
  cron_expression       = "0 1 * * *"
  checkout_ref          = "main"
  config_ref            = "main"
  attribution_actor     = "current"

  parameters = {
    deploy_env = "staging"
  }
}
