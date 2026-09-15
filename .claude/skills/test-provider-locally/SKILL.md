---
name: test-provider-locally
description: >
  Run this provider's real code against the live CircleCI API using a locally
  built binary and terraform dev_overrides, to reproduce a bug or confirm a fix
  end to end. Use when asked to "test this provider locally", "verify against
  the real API", "reproduce this with terraform", or when a provider bug depends
  on what the API actually returns and unit tests cannot settle it.
---

# Test the provider locally against the live API

Unit tests prove the provider's logic. This proves its behaviour against the
real service. Reach for it when the bug depends on what the API actually
returns — response shapes are the usual culprit behind
`Provider produced inconsistent result after apply`.

This creates and destroys **real resources**. Steps 1 and 2 are mandatory
before any mutation.

## 1. Resolve the target

Take the first source that provides all of `project`, `pipeline`, `repo`:

1. Skill arguments: `project=<uuid> pipeline=<uuid> repo=<external-id>`
2. Environment, or a gitignored `.env` / `.env.local`:
   `CIRCLE_TOKEN`, `TFP_E2E_PROJECT_ID`, `TFP_E2E_PIPELINE_ID`,
   `TFP_E2E_REPO_EXTERNAL_ID`
3. Otherwise **ask**. Never guess a target, and never reuse an ID found in the
   acceptance tests — those point at shared fixtures.

```sh
API=https://circleci.com/api/v2
AUTH=(-H "Circle-Token: $CIRCLE_TOKEN" -H 'Content-Type: application/json')
```

## 2. Preflight, then confirm

Run both checks and show the user the result before mutating anything.

```sh
# a. Does the pipeline still exist? A deleted pipeline returns a 404 that
#    looks like a provider bug but is not one.
curl -sS "${AUTH[@]}" "$API/projects/$PROJECT_ID/pipeline-definitions" \
  | jq -e --arg id "$PIPELINE_ID" \
      '.items[] | select(.id==$id) | {id, name, created_at, checkout_source, config_source}' \
  || { echo "pipeline $PIPELINE_ID not found - stop and re-confirm the target"; exit 1; }

# b. Is it empty? Pre-existing triggers make leftovers impossible to attribute.
curl -sS "${AUTH[@]}" "$API/projects/$PROJECT_ID/pipeline-definitions/$PIPELINE_ID/triggers" \
  | jq '{existing: (.items | length), ids: [.items[].id]}'
```

Stop and get explicit confirmation to continue, because:

- A pipeline named `temp` with zero triggers is a throwaway. One named
  `pr-review-acceptance` is live infrastructure.
- An **enabled** `github_app` trigger fires real pipeline runs on the repo it
  points at. Prefer `disabled = true` for the resource's whole life and drive
  updates by changing another field, unless the bug needs the toggle itself.
- If the existing count is not zero, get acknowledgement or a different target.

Echo every resource ID the moment it is created, so an interrupted run leaves a
traceable trail.

## 3. Set up the dev override

`~/.terraformrc` is **user-global**. Back it up before writing.

```sh
[ -f ~/.terraformrc ] && cp ~/.terraformrc "$HOME/.terraformrc.bak.$(date +%s)"

BIN=$(mktemp -d)
go build -o "$BIN/terraform-provider-circleci" .

cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "CircleCI-Public/circleci" = "${BIN}"
  }
  direct {}
}
EOF
```

- Do **not** run `terraform init`. dev_overrides bypasses installation, and
  init will fail.
- The `source` in `required_providers` must match the override key exactly.
- A plan printing `provider development overrides are set` confirms the wiring.

## 4. Write the config

Use a temp directory so nothing lands in the repo.

```sh
WORK=$(mktemp -d)
cat > "$WORK/main.tf" <<TF
terraform {
  required_providers {
    circleci = {
      source = "CircleCI-Public/circleci"
    }
  }
}

provider "circleci" {
  host = "https://circleci.com/api/v2"
  # key is read from CIRCLE_TOKEN
}

resource "circleci_trigger" "probe" {
  project_id                    = "$PROJECT_ID"
  pipeline_id                   = "$PIPELINE_ID"
  event_source_provider         = "github_app"
  event_source_repo_external_id = "$REPO_EXTERNAL_ID"
  event_preset                  = "all-pushes"
  disabled                      = true
}
TF
terraform -chdir="$WORK" plan   # wiring check, creates nothing
```

## 5. Reproduce, then verify

Build the **unfixed** code first. A fix you never saw fail proves nothing.

| Step | Expect |
|------|--------|
| unfixed binary, `apply` | succeeds |
| change one field, `apply` | the reported failure |
| apply the fix, rebuild, `apply` | clean apply |
| `terraform plan` again | `No changes` — no perpetual drift |
| `jq` the state | attribute matches the config |

```sh
jq '.resources[0].instances[0].attributes' "$WORK/terraform.tfstate"
```

For consistency bugs, `null` versus `""` is the whole question — read the state
file, not just the plan output.

## 6. Clean up, and say so

```sh
terraform -chdir="$WORK" destroy -auto-approve

# confirm nothing is left behind
curl -sS "${AUTH[@]}" "$API/projects/$PROJECT_ID/pipeline-definitions/$PIPELINE_ID/triggers" \
  | jq '{remaining: (.items | length)}'

# restore the user's terraform config
if ls "$HOME"/.terraformrc.bak.* >/dev/null 2>&1; then
  mv "$(ls -t "$HOME"/.terraformrc.bak.* | head -1)" ~/.terraformrc
else
  rm -f ~/.terraformrc
fi
rm -rf "$WORK" "$BIN"
```

Report the remaining count explicitly. Silence reads as unverified.

## 7. Turn the finding into a permanent test

Live verification proves it once; a test stops the regression. Add one under
`internal/provider/` driving the lifecycle against an `httptest` fake that
mimics the API shape you just observed. `provider_runner_host_test.go` shows
the minimal form; `trigger_resource_fake_test.go`, where present, shows a full
create-and-update lifecycle.

- A regression test **must fail without the fix**. Check that explicitly.
- `resource.Test` skips unless `TF_ACC=1`, so these run in the
  `build-and-test-terraform-*` matrix jobs but not the fast `test` job. A fake
  host needs no API token.
- Match the package's existing idioms: `statecheck`, `knownvalue`,
  `plancheck`, `ExpectError`. `internal/provider` does not use `gotest.tools`.

## Gotchas

- **The API already applied your change.** After a failed apply the request may
  have succeeded server-side, so re-applying the same value is a no-op. Flip the
  field back to force a genuine update.
- **A 404 is probably not your bug.** Test pipelines get deleted. Re-run the
  step 2 preflight before investigating further.
- **`Provider produced inconsistent result after apply` names the attribute and
  both values.** That message is the finding, not noise. Terraform's advice to
  "report this to the provider's issue tracker" means you are already there.
- **Optional vs Optional+Computed matters.** For `Optional` without `Computed`,
  state must equal config exactly, so writing `""` over a null plan value fails.
  Check the schema before concluding the API is at fault.
- **Acceptance tests share fixtures.** Hardcoded project, context and pipeline
  IDs are reused across the terraform version matrix, which runs those jobs
  concurrently. A failure in an unrelated resource is often that race, not your
  change. Diagnose before rerunning.
