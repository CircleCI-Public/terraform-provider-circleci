terraform {
  required_providers {
    circleci = {
      source = "registry.terraform.io/circleci/circleci"
    }
  }
}

provider "circleci" {
  host = "https://circleci.com/api/v2/"
  #key  = "*****"
}
