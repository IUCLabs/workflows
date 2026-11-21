terraform {
  required_version = ">= 1.6"
}

variable "greeting" {
  description = "Set by the workflow's env input; plan fails if it was not exported."
  type        = string

  validation {
    condition     = var.greeting == "from-env"
    error_message = "TF_VAR_greeting was not exported from the env input."
  }
}

resource "terraform_data" "fixture" {
  input = var.greeting
}

output "greeting" {
  description = "Echo of the greeting input."
  value       = terraform_data.fixture.output
}
