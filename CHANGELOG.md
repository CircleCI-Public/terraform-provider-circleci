## 0.1.0 (Unreleased)

FEATURES:

ENHANCEMENTS:

* resource/circleci_project: `oss` can be set so a project's builds can use free open source credits. CircleCI only applies `true` when the underlying repository is open source.
* resource/circleci_trigger: `parameters` now accepts typed values (strings, booleans, and numbers) instead of only strings, so scheduled triggers can supply boolean and numeric pipeline parameters ([#122](https://github.com/CircleCI-Public/terraform-provider-circleci/issues/122)).
* data-source/circleci_trigger: `parameters` now reports typed values (strings, booleans, and numbers).
