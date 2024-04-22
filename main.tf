terraform {
  required_providers {
    go = {
      source  = "terraform.local/local/go"
      version = "0.0.1"
    }
  }
}

provider "go" {
  go = file("./fixtures/lib.go")
}

output "test" {
  value = provider::go::hello("papaya")
}
