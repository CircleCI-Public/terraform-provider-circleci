---
page_title: "Setting up Forgejo projects"
subcategory: ""
description: |-
  Connect a standalone CircleCI organization to Forgejo and build its repositories.
---

# Setting up Forgejo projects

CircleCI builds Forgejo repositories in a standalone (`circleci`) organization, through a connection between the organization and the Forgejo instance. The Forgejo instance must run Forgejo 16 or later and be reachable from CircleCI; if it sits behind a proxy, `request_headers` can carry the proxy's credentials.

## 1. Create the organization

```terraform
resource "circleci_organization" "example" {
  name     = "example"
  vcs_type = "circleci"
}
```

## 2. Create the Authorized Integrations in Forgejo

CircleCI authenticates to Forgejo with OIDC tokens, which Forgejo accepts through Authorized Integrations. CircleCI checks both integrations when the connection is created, so they must exist first.

1. Create a dedicated Forgejo user without site administrator rights; CircleCI acts as this user, and refuses a site administrator. If the repositories belong to a Forgejo organization, add the user to a team in it that can read them.
2. Signed in as that user, open **Settings → Authorized Integrations** (`/user/settings/authorized-integrations`) and create two generic integrations, both with the issuer `https://oidc.circleci.com/org/<organization ID>` and access to the repositories CircleCI builds:

   | Integration | Claim rule                                                  | Scopes                                                  |
   |-------------|-------------------------------------------------------------|---------------------------------------------------------|
   | Connection  | `sub` equals `circleci/soc-integrations/forgejo/connection` | `read:user`, `read:organization`, `read:repository`     |
   | Checkout    | `sub` equals `circleci/soc-integrations/forgejo/checkout`   | `read:repository`                                       |

   CircleCI refuses an integration without the claim rule on `sub`, because it would also accept tokens issued to jobs. Forgejo fetches the issuer's discovery document when an integration is saved, so the Forgejo server must be able to reach `https://oidc.circleci.com`.
3. Copy the audience Forgejo shows for each integration.

## 3. Connect the organization

```terraform
resource "circleci_forgejo_connection" "example" {
  organization_id     = circleci_organization.example.id
  url                 = "https://forgejo.example.com"
  owner               = "example-org"
  connection_audience = var.forgejo_connection_audience
  checkout_audience   = var.forgejo_checkout_audience
}
```

## 4. Create the webhook in Forgejo

Create a webhook for `owner` that sends the events in `webhook_events` to `webhook_url`, with `webhook_secret` as its secret. CircleCI shows the secret only once, so read it from the Terraform state, or pass it to a Forgejo Terraform provider. To replace it, set `webhook_secret_rotation` to a new value and update the webhook with the new secret.

## 5. Create the project, pipeline and trigger

```terraform
resource "circleci_project" "service" {
  name            = "service"
  organization_id = circleci_organization.example.id
}

resource "circleci_pipeline" "build" {
  project_id                     = circleci_project.service.id
  name                           = "build"
  description                    = "Builds the service"
  config_source_provider         = "forgejo"
  config_source_repo_full_name   = "example-org/service"
  config_source_file_path        = ".circleci/config.yml"
  checkout_source_provider       = "forgejo"
  checkout_source_repo_full_name = "example-org/service"

  depends_on = [circleci_forgejo_connection.example]
}

resource "circleci_trigger" "prs" {
  project_id                    = circleci_project.service.id
  pipeline_id                   = circleci_pipeline.build.id
  event_source_provider         = "forgejo"
  event_preset                  = "only-build-prs"
  event_source_repo_external_id = circleci_pipeline.build.checkout_source_repo_external_id
}
```

The pipeline depends on the connection explicitly because CircleCI resolves the repository through it when the pipeline is created.

## 6. Grant access with groups

CircleCI manages roles in a standalone organization through groups. Users must already belong to the organization, and are identified by their CircleCI user ID.

```terraform
resource "circleci_group" "platform" {
  organization_id = circleci_organization.example.id
  name            = "platform"
}

resource "circleci_group_member" "alice" {
  organization_id = circleci_organization.example.id
  group_id        = circleci_group.platform.id
  user_id         = var.alice_user_id
}

resource "circleci_group_project_role" "platform_service" {
  organization_id = circleci_organization.example.id
  group_id        = circleci_group.platform.id
  project_id      = circleci_project.service.id
  role            = "project-contributor"
}
```

## Removing the connection

Destroying `circleci_forgejo_connection` also deletes every Forgejo pipeline and trigger in the organization. CircleCI changes nothing in Forgejo, so delete the webhook and both Authorized Integrations there yourself.
