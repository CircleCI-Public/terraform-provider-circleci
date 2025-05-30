schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2021
  copyright_holder = "CircleCI and HashiCorp, Inc."

  header_ignore = [
    # examples used within documentation (prose)
    "examples/**",
    ".circleci/**",

    # GitHub issue template configuration
    ".github/ISSUE_TEMPLATE/*.yml",

    # golangci-lint tooling configuration
    ".golangci.yml",

    # GoReleaser tooling configuration
    ".goreleaser.yml",
  ]
}