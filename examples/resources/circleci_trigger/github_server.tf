resource "circleci_trigger" "github_server_example" {
  project_id                    = "20209578-aa1c-4b4c-9ca5-f6e38a47cf73"
  pipeline_id                   = "9c7c4e85-5022-41d0-a6b0-705cfa856485"
  event_source_provider         = "github_server"
  event_source_repo_external_id = "2259"
  event_preset                  = "all-pushes"
  checkout_ref                  = "main"
  config_ref                    = "main"
  disabled                      = false
}
