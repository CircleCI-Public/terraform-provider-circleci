terraform {
  required_providers {
    circleci = {
      source = "registry.terraform.io/circleci/circleci"
    }
  }
}

data "circleci_project" "test_project" {
  slug = "circleci/8e4z1Akd74woxagxnvLT5q/V29Cenkg8EaiSZARmWm8Lz"
}

output "circleci_project_id" {
  value = data.circleci_project.test_project.id
}

output "circleci_project_slug" {
  value = data.circleci_project.test_project.slug
}

output "circleci_project_name" {
  value = data.circleci_project.test_project.name
}
