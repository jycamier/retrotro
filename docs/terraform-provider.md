# Terraform Provider Retrotro

The Terraform provider allows you to manage Retrotro resources as Infrastructure as Code.

## Installation

### From Source

```bash
cd terraform-provider-retrotro
go build -o terraform-provider-retrotro
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/jycamier/retrotro/1.0.0/darwin_arm64
mv terraform-provider-retrotro ~/.terraform.d/plugins/registry.terraform.io/jycamier/retrotro/1.0.0/darwin_arm64/
```

### Terraform Configuration

```hcl
terraform {
  required_providers {
    retrotro = {
      source = "registry.terraform.io/jycamier/retrotro"
    }
  }
}
```

## Provider Configuration

```hcl
provider "retrotro" {
  api_url   = "https://retrotro.example.com"
  api_token = var.retrotro_token
}
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `api_url` | string | No | Retrotro API URL (default: `http://localhost:8080`) |
| `api_token` | string | Yes | JWT authentication token |

### Environment Variables

Parameters can also be set via environment variables:

- `RETROTRO_API_URL` - API URL
- `RETROTRO_API_TOKEN` - Authentication token

## Data Sources

### retrotro_team

Fetches a team by its slug.

```hcl
data "retrotro_team" "engineering" {
  slug = "engineering"
}

output "team_id" {
  value = data.retrotro_team.engineering.id
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `slug` | string | Yes | Team's URL-friendly identifier |

#### Exported Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Team's unique ID |
| `name` | string | Team name |
| `description` | string | Team description |

### retrotro_webhook

Fetches an existing webhook.

```hcl
data "retrotro_webhook" "existing" {
  id      = "550e8400-e29b-41d4-a716-446655440000"
  team_id = data.retrotro_team.engineering.id
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `id` | string | Yes | Webhook ID |
| `team_id` | string | Yes | Team ID |

#### Exported Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `name` | string | Webhook name |
| `url` | string | Destination URL |
| `events` | list(string) | Subscribed events |
| `enabled` | bool | Activation status |

## Resources

### retrotro_webhook

Manages a webhook.

```hcl
resource "retrotro_webhook" "slack_notifications" {
  team_id = data.retrotro_team.engineering.id
  name    = "Slack Notifications"
  url     = "https://hooks.slack.com/services/xxx/yyy/zzz"
  secret  = var.webhook_secret

  events = [
    "retro.completed",
    "action.created"
  ]

  enabled = true
}
```

#### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `team_id` | string | Yes | Team ID (forces replacement if changed) |
| `name` | string | Yes | Webhook name |
| `url` | string | Yes | Destination URL |
| `secret` | string | No | HMAC signing secret (sensitive) |
| `events` | list(string) | Yes | Events to subscribe to |
| `enabled` | bool | No | Enable webhook (default: `true`) |

#### Exported Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Webhook's unique ID |

#### Valid Events

- `retro.completed` - Retrospective completed
- `action.created` - Action created

## Complete Examples

### Webhooks for a Team

```hcl
terraform {
  required_providers {
    retrotro = {
      source = "registry.terraform.io/jycamier/retrotro"
    }
  }
}

provider "retrotro" {
  api_url   = var.retrotro_url
  api_token = var.retrotro_token
}

variable "retrotro_url" {
  description = "Retrotro API URL"
  type        = string
  default     = "https://retrotro.example.com"
}

variable "retrotro_token" {
  description = "Authentication token"
  type        = string
  sensitive   = true
}

variable "webhook_secret" {
  description = "Shared secret for webhooks"
  type        = string
  sensitive   = true
}

# Fetch the team
data "retrotro_team" "engineering" {
  slug = "engineering"
}

# Slack webhook - retro completion notifications
resource "retrotro_webhook" "slack_retro" {
  team_id = data.retrotro_team.engineering.id
  name    = "Slack - Retro Completed"
  url     = "https://hooks.slack.com/exemple"
  secret  = var.webhook_secret

  events = ["retro.completed"]
}

# Jira webhook - automatic ticket creation
resource "retrotro_webhook" "jira_actions" {
  team_id = data.retrotro_team.engineering.id
  name    = "Jira - Action Items"
  url     = "https://automation.atlassian.com/pro/hooks/xxxxxxxxx"
  secret  = var.webhook_secret

  events = ["action.created"]
}

# Monitoring webhook - all events
resource "retrotro_webhook" "datadog" {
  team_id = data.retrotro_team.engineering.id
  name    = "Datadog Monitoring"
  url     = "https://http-intake.logs.datadoghq.com/api/v2/logs"
  secret  = var.webhook_secret

  events = [
    "retro.completed",
    "action.created"
  ]
}

# Outputs
output "team_id" {
  value = data.retrotro_team.engineering.id
}

output "slack_webhook_id" {
  value = retrotro_webhook.slack_retro.id
}

output "jira_webhook_id" {
  value = retrotro_webhook.jira_actions.id
}
```

### Using with Terraform Cloud

```hcl
terraform {
  cloud {
    organization = "my-org"

    workspaces {
      name = "retrotro-webhooks"
    }
  }

  required_providers {
    retrotro = {
      source = "registry.terraform.io/jycamier/retrotro"
    }
  }
}

# Sensitive variables are defined in Terraform Cloud
variable "retrotro_token" {
  type      = string
  sensitive = true
}

variable "webhook_secret" {
  type      = string
  sensitive = true
}

provider "retrotro" {
  api_token = var.retrotro_token
}
```

### Reusable Module

```hcl
# modules/retrotro-webhooks/main.tf

variable "team_slug" {
  type = string
}

variable "slack_webhook_url" {
  type    = string
  default = ""
}

variable "jira_webhook_url" {
  type    = string
  default = ""
}

variable "webhook_secret" {
  type      = string
  sensitive = true
}

data "retrotro_team" "team" {
  slug = var.team_slug
}

resource "retrotro_webhook" "slack" {
  count = var.slack_webhook_url != "" ? 1 : 0

  team_id = data.retrotro_team.team.id
  name    = "Slack Notifications"
  url     = var.slack_webhook_url
  secret  = var.webhook_secret
  events  = ["retro.completed"]
}

resource "retrotro_webhook" "jira" {
  count = var.jira_webhook_url != "" ? 1 : 0

  team_id = data.retrotro_team.team.id
  name    = "Jira Integration"
  url     = var.jira_webhook_url
  secret  = var.webhook_secret
  events  = ["action.created"]
}

output "team_id" {
  value = data.retrotro_team.team.id
}
```

```hcl
# Module usage

module "engineering_webhooks" {
  source = "./modules/retrotro-webhooks"

  team_slug         = "engineering"
  slack_webhook_url = "https://hooks.slack.com/services/xxx"
  jira_webhook_url  = "https://automation.atlassian.com/xxx"
  webhook_secret    = var.webhook_secret
}

module "product_webhooks" {
  source = "./modules/retrotro-webhooks"

  team_slug         = "product"
  slack_webhook_url = "https://hooks.slack.com/services/yyy"
  webhook_secret    = var.webhook_secret
}
```

## Import

Existing webhooks can be imported:

```bash
terraform import retrotro_webhook.example TEAM_ID/WEBHOOK_ID
```

## Debugging

To enable provider debug logs:

```bash
TF_LOG=DEBUG terraform apply
```

## Development

### Build

```bash
cd terraform-provider-retrotro
go build -o terraform-provider-retrotro
```

### Tests

```bash
go test ./...
```

### Local Installation

```bash
# macOS ARM
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/jycamier/retrotro/1.0.0/darwin_arm64
cp terraform-provider-retrotro ~/.terraform.d/plugins/registry.terraform.io/jycamier/retrotro/1.0.0/darwin_arm64/

# Linux AMD64
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/jycamier/retrotro/1.0.0/linux_amd64
cp terraform-provider-retrotro ~/.terraform.d/plugins/registry.terraform.io/jycamier/retrotro/1.0.0/linux_amd64/
```

## Related Documentation

- [Webhooks](./webhooks.md) - Webhook events and payloads
- [Configuration](./configuration.md) - Retrospective configuration options
