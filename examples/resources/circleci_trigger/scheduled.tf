resource "circleci_trigger" "scheduled" {
  project_id                              = "00000000-0000-0000-0000-000000000000"
  pipeline_id                             = "00000000-0000-0000-0000-000000000001"
  event_source_provider                   = "schedule"
  event_name                              = "nightly-build"
  checkout_ref                            = "main"
  config_ref                              = "main"
  event_source_schedule_cron_expression   = "0 2 * * *"
  event_source_schedule_attribution_actor = "system"
  parameters = {
    run_nightly_foo = "true"
    branch          = "main"
  }
}
