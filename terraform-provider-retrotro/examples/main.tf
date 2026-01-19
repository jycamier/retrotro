terraform {
  required_providers {
    retrotro = {
      source = "registry.terraform.io/jycamier/retrotro"
    }
  }
}

# Configure the provider
provider "retrotro" {
  api_url   = "https://retrotro.example.com"
  api_token = var.retrotro_token
}

variable "retrotro_token" {
  description = "API token for Retrotro authentication"
  type        = string
  sensitive   = true
}

variable "webhook_secret" {
  description = "Secret for webhook payload signing"
  type        = string
  sensitive   = true
  default     = ""
}

# Data source - fetch a team by slug
data "retrotro_team" "engineering" {
  slug = "engineering"
}

# Resource - create a webhook for Slack notifications
resource "retrotro_webhook" "slack_retro_completed" {
  team_id = data.retrotro_team.engineering.id
  name    = "Slack - Retro Completed"
  url     = "https://hooks.slack.com/services/xxx/yyy/zzz"
  secret  = var.webhook_secret

  events = [
    "retro.completed"
  ]

  enabled = true
}

# Resource - create a webhook for Jira integration
resource "retrotro_webhook" "jira_actions" {
  team_id = data.retrotro_team.engineering.id
  name    = "Jira - Action Items"
  url     = "https://automation.atlassian.com/pro/hooks/xxx"
  secret  = var.webhook_secret

  events = [
    "action.created"
  ]

  enabled = true
}

# Output the webhook IDs
output "slack_webhook_id" {
  description = "ID of the Slack webhook"
  value       = retrotro_webhook.slack_retro_completed.id
}

output "jira_webhook_id" {
  description = "ID of the Jira webhook"
  value       = retrotro_webhook.jira_actions.id
}
