## 0.1.0 (Unreleased)

FEATURES:

BUG FIXES:

* resource/circleci_project: updating one advanced setting no longer writes `false` for toggles the configuration leaves unset. Unknown values were encoded as false, which could disable settings such as `forks_receive_secret_env_vars`.
* resource/circleci_project: creating a project no longer writes `false` for `auto_cancel_builds`, `setup_workflows`, and `write_settings_requires_admin` when the configuration leaves them out. CircleCI's defaults apply to them instead. `forks_receive_secret_env_vars` is still created as `false`, because CircleCI defaults it to `true` and a new project must not hand a forked pull request its secrets.
* resource/circleci_project: setting `oss` succeeds when CircleCI's v1.1 settings API returns an empty JSON string. The provider reads the flag back instead of failing the apply.
* resource/circleci_project: `oss` is only written when it has to change, so `oss = false` applies to a project whatever the visibility of its repository.

ENHANCEMENTS:

* resource/circleci_project: `oss` can be set so a project's builds can use free open source credits. CircleCI only applies `true` when the underlying repository is open source; otherwise the apply fails and the flag is left unchanged. `false` applies to any project ([#30](https://github.com/CircleCI-Public/terraform-provider-circleci/issues/30)).
* resource/circleci_trigger: `parameters` now accepts typed values (strings, booleans, and numbers) instead of only strings, so scheduled triggers can supply boolean and numeric pipeline parameters ([#122](https://github.com/CircleCI-Public/terraform-provider-circleci/issues/122)).
* data-source/circleci_trigger: `parameters` now reports typed values (strings, booleans, and numbers).
