# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

provider "trinogateway" {
  endpoint = "http://127.0.0.1:18080"

  # login = "admin"
  # password = "admin"
}


resource "trinogateway_backend" "example" {
  name         = "trino-1"
  proxy_to     = "http://localhost:8085"
  active       = false
  routing_group = "some"
  external_url = "http://trino-1:8081"
}


terraform {
  required_providers {
    trinogateway = {
      source = "hashicorp.com/paragor/trinogateway"
    }
  }
}
