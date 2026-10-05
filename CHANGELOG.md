## 0.1.0 (Unreleased)

FEATURES:

BUG FIXES:

* resource/circleci_project: updating one advanced setting no longer writes `false` for toggles the configuration leaves unset. Unknown values were encoded as false, which could disable settings such as `forks_receive_secret_env_vars`.

ENHANCEMENTS:

* resource/circleci_project: `oss` can be set so a project's builds can use free open source credits. CircleCI only applies `true` when the underlying repository is open source; otherwise the apply fails and the flag is left unchanged.
* resource/circleci_trigger: `parameters` now accepts typed values (strings, booleans, and numbers) instead of only strings, so scheduled triggers can supply boolean and numeric pipeline parameters ([#122](https://github.com/CircleCI-Public/terraform-provider-circleci/issues/122)).
* data-source/circleci_trigger: `parameters` now reports typed values (strings, booleans, and numbers).
