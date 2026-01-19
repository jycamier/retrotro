# API Reference

This document provides a complete reference for the Retrotro REST API.

## Authentication

All API endpoints (except auth routes) require JWT authentication:

```bash
Authorization: Bearer <token>
```

## Base URL

```
https://retrotro.example.com/api/v1
```

## Endpoints

### Authentication

#### Login (OIDC)

```bash
GET /auth/login
```

Redirects to OIDC provider for authentication.

#### Callback

```bash
GET /auth/callback?code=xxx&state=xxx
```

Handles OIDC callback and returns JWT tokens.

#### Refresh Token

```bash
POST /auth/refresh
Content-Type: application/json

{
  "refreshToken": "xxx"
}
```

#### Logout

```bash
POST /auth/logout
```

#### Get Current User

```bash
GET /api/v1/me
```

**Response:**
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "displayName": "John Doe",
  "avatarUrl": "https://...",
  "isAdmin": false
}
```

---

### Teams

#### List Teams

```bash
GET /api/v1/teams
```

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "Engineering",
    "slug": "engineering",
    "description": "Engineering team"
  }
]
```

#### Create Team

```bash
POST /api/v1/teams
Content-Type: application/json

{
  "name": "Engineering",
  "slug": "engineering",
  "description": "Engineering team"
}
```

#### Get Team

```bash
GET /api/v1/teams/{teamId}
```

#### Update Team

```bash
PUT /api/v1/teams/{teamId}
Content-Type: application/json

{
  "name": "New Name",
  "description": "Updated description"
}
```

#### Delete Team

```bash
DELETE /api/v1/teams/{teamId}
```

#### List Team Members

```bash
GET /api/v1/teams/{teamId}/members
```

**Response:**
```json
[
  {
    "userId": "uuid",
    "displayName": "John Doe",
    "email": "john@example.com",
    "role": "admin"
  }
]
```

#### Add Team Member

```bash
POST /api/v1/teams/{teamId}/members
Content-Type: application/json

{
  "email": "user@example.com",
  "role": "member"
}
```

Roles: `admin`, `member`

#### Remove Team Member

```bash
DELETE /api/v1/teams/{teamId}/members/{userId}
```

#### Update Member Role

```bash
PUT /api/v1/teams/{teamId}/members/{userId}/role
Content-Type: application/json

{
  "role": "facilitator"
}
```

---

### Templates

#### List Templates

```bash
GET /api/v1/templates
GET /api/v1/templates?teamId={teamId}
```

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "Mad/Sad/Glad",
    "description": "Classic emotional retrospective",
    "columns": [...],
    "phaseTimes": {...},
    "isBuiltIn": true
  }
]
```

#### Get Template

```bash
GET /api/v1/templates/{templateId}
```

#### Create Template

```bash
POST /api/v1/templates
Content-Type: application/json

{
  "name": "Custom Template",
  "description": "My custom template",
  "teamId": "uuid",
  "columns": [
    {
      "id": "col1",
      "name": "Column 1",
      "color": "#22c55e",
      "order": 0
    }
  ],
  "phaseTimes": {
    "brainstorm": 300
  }
}
```

---

### Retrospectives

#### List Retrospectives

```bash
GET /api/v1/retrospectives?teamId={teamId}
GET /api/v1/retrospectives?teamId={teamId}&status=active
```

Status: `draft`, `active`, `completed`

#### Create Retrospective

```bash
POST /api/v1/retrospectives
Content-Type: application/json

{
  "name": "Sprint 42 Retro",
  "teamId": "uuid",
  "templateId": "uuid",
  "maxVotesPerUser": 5,
  "maxVotesPerItem": 3,
  "anonymousVoting": false,
  "anonymousItems": false,
  "allowItemEdit": true,
  "allowVoteChange": true,
  "phaseTimerOverrides": {
    "brainstorm": 600
  },
  "scheduledAt": "2025-01-25T14:00:00Z"
}
```

#### Get Retrospective

```bash
GET /api/v1/retrospectives/{retroId}
```

**Response:**
```json
{
  "id": "uuid",
  "name": "Sprint 42 Retro",
  "teamId": "uuid",
  "templateId": "uuid",
  "facilitatorId": "uuid",
  "status": "active",
  "currentPhase": "brainstorm",
  "maxVotesPerUser": 5,
  "maxVotesPerItem": 3,
  "anonymousVoting": false,
  "anonymousItems": false,
  "allowItemEdit": true,
  "allowVoteChange": true,
  "phaseTimerOverrides": null,
  "startedAt": "2025-01-22T14:00:00Z",
  "endedAt": null
}
```

#### Update Retrospective

```bash
PUT /api/v1/retrospectives/{retroId}
Content-Type: application/json

{
  "name": "Updated Name",
  "maxVotesPerUser": 7
}
```

#### Delete Retrospective

```bash
DELETE /api/v1/retrospectives/{retroId}
```

#### Start Retrospective

```bash
POST /api/v1/retrospectives/{retroId}/start
```

#### End Retrospective

```bash
POST /api/v1/retrospectives/{retroId}/end
```

---

### Phases

#### Next Phase

```bash
POST /api/v1/retrospectives/{retroId}/phase/next
```

**Response:**
```json
{
  "phase": "vote"
}
```

#### Set Phase

```bash
POST /api/v1/retrospectives/{retroId}/phase/set
Content-Type: application/json

{
  "phase": "discuss"
}
```

Phases: `waiting`, `icebreaker`, `brainstorm`, `group`, `vote`, `discuss`, `action`, `roti`

---

### Timer

#### Start Timer

```bash
POST /api/v1/retrospectives/{retroId}/timer/start
```

#### Pause Timer

```bash
POST /api/v1/retrospectives/{retroId}/timer/pause
```

#### Resume Timer

```bash
POST /api/v1/retrospectives/{retroId}/timer/resume
```

#### Reset Timer

```bash
POST /api/v1/retrospectives/{retroId}/timer/reset
```

#### Add Time

```bash
POST /api/v1/retrospectives/{retroId}/timer/add-time
Content-Type: application/json

{
  "seconds": 60
}
```

---

### Items

#### List Items

```bash
GET /api/v1/retrospectives/{retroId}/items
```

**Response:**
```json
[
  {
    "id": "uuid",
    "retroId": "uuid",
    "columnId": "mad",
    "content": "Too many meetings",
    "authorId": "uuid",
    "position": 0,
    "groupId": null,
    "votes": 5,
    "createdAt": "2025-01-22T14:05:00Z"
  }
]
```

#### Create Item

```bash
POST /api/v1/retrospectives/{retroId}/items
Content-Type: application/json

{
  "columnId": "mad",
  "content": "Too many meetings"
}
```

#### Update Item

```bash
PUT /api/v1/retrospectives/{retroId}/items/{itemId}
Content-Type: application/json

{
  "content": "Updated content"
}
```

#### Delete Item

```bash
DELETE /api/v1/retrospectives/{retroId}/items/{itemId}
```

#### Group Items

```bash
POST /api/v1/retrospectives/{retroId}/items/{itemId}/group
Content-Type: application/json

{
  "childIds": ["uuid1", "uuid2"]
}
```

---

### Votes

#### Vote on Item

```bash
POST /api/v1/retrospectives/{retroId}/items/{itemId}/vote
```

**Response:** `204 No Content`

**Errors:**
- `400` - Vote limit reached (per user or per item)

#### Remove Vote

```bash
DELETE /api/v1/retrospectives/{retroId}/items/{itemId}/vote
```

---

### Actions

#### List Actions

```bash
GET /api/v1/retrospectives/{retroId}/actions
```

**Response:**
```json
[
  {
    "id": "uuid",
    "retroId": "uuid",
    "itemId": "uuid",
    "title": "Improve documentation",
    "description": "Add examples",
    "assigneeId": "uuid",
    "dueDate": "2025-02-01",
    "priority": 1,
    "isCompleted": false,
    "createdBy": "uuid"
  }
]
```

#### Create Action

```bash
POST /api/v1/retrospectives/{retroId}/actions
Content-Type: application/json

{
  "title": "Improve documentation",
  "description": "Add request/response examples",
  "assigneeId": "uuid",
  "dueDate": "2025-02-01",
  "priority": 1,
  "itemId": "uuid"
}
```

#### Update Action

```bash
PUT /api/v1/retrospectives/{retroId}/actions/{actionId}
Content-Type: application/json

{
  "title": "Updated title",
  "isCompleted": true
}
```

#### Delete Action

```bash
DELETE /api/v1/retrospectives/{retroId}/actions/{actionId}
```

---

### Icebreaker

#### Get Moods

```bash
GET /api/v1/retrospectives/{retroId}/icebreaker
```

**Response:**
```json
[
  {
    "userId": "uuid",
    "mood": "sunny",
    "submittedAt": "2025-01-22T14:02:00Z"
  }
]
```

Moods: `sunny`, `partly_cloudy`, `cloudy`, `rainy`, `stormy`

---

### ROTI

#### Get Results

```bash
GET /api/v1/retrospectives/{retroId}/roti
```

**Response:**
```json
{
  "average": 3.8,
  "count": 8,
  "distribution": {
    "1": 0,
    "2": 1,
    "3": 2,
    "4": 4,
    "5": 1
  },
  "revealed": true,
  "votes": [
    { "userId": "uuid", "rating": 4 }
  ]
}
```

---

### Webhooks

See [Webhooks Documentation](./webhooks.md) for complete webhook API reference.

#### List Webhooks

```bash
GET /api/v1/teams/{teamId}/webhooks
```

#### Create Webhook

```bash
POST /api/v1/teams/{teamId}/webhooks
Content-Type: application/json

{
  "name": "Slack Notifications",
  "url": "https://hooks.slack.com/...",
  "secret": "optional-secret",
  "events": ["retro.completed", "action.created"],
  "isEnabled": true
}
```

#### Get Webhook

```bash
GET /api/v1/teams/{teamId}/webhooks/{webhookId}
```

#### Update Webhook

```bash
PUT /api/v1/teams/{teamId}/webhooks/{webhookId}
Content-Type: application/json

{
  "name": "Updated Name",
  "isEnabled": false
}
```

#### Delete Webhook

```bash
DELETE /api/v1/teams/{teamId}/webhooks/{webhookId}
```

#### List Deliveries

```bash
GET /api/v1/teams/{teamId}/webhooks/{webhookId}/deliveries?limit=50
```

---

### Stats

#### Team ROTI Stats

```bash
GET /api/v1/teams/{teamId}/stats/roti
```

#### Team Mood Stats

```bash
GET /api/v1/teams/{teamId}/stats/mood
```

#### My Stats

```bash
GET /api/v1/teams/{teamId}/stats/me
```

#### User ROTI Stats

```bash
GET /api/v1/teams/{teamId}/stats/users/{userId}/roti
```

#### User Mood Stats

```bash
GET /api/v1/teams/{teamId}/stats/users/{userId}/mood
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message here"
}
```

### Common Status Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Missing or invalid token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource doesn't exist |
| 500 | Internal Server Error |

## WebSocket API

Connect to `/ws` with query parameters:

```
wss://retrotro.example.com/ws?token={jwt}&retroId={uuid}
```

See [Dynamic Facilitator](./dynamic-facilitator.md) for WebSocket message formats.

## Rate Limiting

Currently no rate limiting is enforced. This may change in future versions.

## Related Documentation

- [Webhooks](./webhooks.md) - Webhook events and payloads
- [Configuration](./configuration.md) - Retrospective configuration
- [Templates](./templates.md) - Available templates
- [Dynamic Facilitator](./dynamic-facilitator.md) - WebSocket messages
- [Terraform Provider](./terraform-provider.md) - Infrastructure as Code
