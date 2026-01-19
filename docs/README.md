# Retrotro Documentation

Welcome to the Retrotro documentation. This guide covers all the features and configuration options available in Retrotro.

## Table of Contents

### Features

- [Webhooks](./webhooks.md) - Send HTTP notifications to external services when events occur
- [Configuration](./configuration.md) - Advanced configuration options for retrospectives
- [Templates](./templates.md) - Available templates including Lean Coffee
- [Dynamic Facilitator](./dynamic-facilitator.md) - Change facilitator during waiting room phase

### Authentication

- [OIDC Authentication](./oidc-authentication.md) - OpenID Connect with Just-In-Time provisioning

### Infrastructure

- [Terraform Provider](./terraform-provider.md) - Manage Retrotro resources as Infrastructure as Code

### API Reference

- [REST API](./api-reference.md) - Complete API documentation

## Quick Links

| Topic | Description |
|-------|-------------|
| [OIDC Authentication](./oidc-authentication.md) | SSO with Keycloak, Azure AD, Okta |
| [Webhooks](./webhooks.md) | Configure webhooks for Slack, Jira, and other integrations |
| [Multiple Votes](./configuration.md#multiple-votes-per-item) | Allow multiple votes per item (default: 3) |
| [Lean Coffee Template](./templates.md#lean-coffee) | Structured discussion format |
| [Terraform Provider](./terraform-provider.md) | IaC for webhook management |

## Getting Started

### Basic Retrospective Flow

1. Create a retrospective with a template
2. Start the session and wait for participants
3. Icebreaker phase - participants share their mood
4. Brainstorm phase - create items in columns
5. Group phase - organize similar items
6. Vote phase - prioritize items
7. Discuss phase - talk about top items
8. Action phase - create action items
9. ROTI phase - rate the retrospective

### Key Configuration Options

```json
{
  "maxVotesPerUser": 5,
  "maxVotesPerItem": 3,
  "anonymousVoting": true,
  "anonymousItems": false
}
```

See [Configuration](./configuration.md) for all options.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend                              │
│                    (React + TypeScript)                      │
└─────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │                   │
              HTTP REST            WebSocket
                    │                   │
┌─────────────────────────────────────────────────────────────┐
│                        Backend                               │
│                      (Go + Chi)                              │
├─────────────────────────────────────────────────────────────┤
│  Handlers  │  Services  │  Repository  │  WebSocket Hub     │
└─────────────────────────────────────────────────────────────┘
          │                   │
    ┌─────┴─────┐             │
    │           │             │
   OIDC    PostgreSQL    Webhooks
   (IdP)                 (HTTP)
```

## Support

For issues and feature requests, please open an issue on GitHub.
