resource "circleci_trigger" "github_server_example" {
  project_id                    = "61169e84-93ee-415d-8d65-ddf6dc0d2939"
  pipeline_id                   = "fefb451c-9966-4b75-b555-d4d94d7116ef"
  event_source_provider         = "github_server"
  event_source_repo_external_id = "952038793"
  event_preset                  = "all-pushes"
  checkout_ref                  = "main"
  config_ref                    = "main"
  disabled                      = false
}
