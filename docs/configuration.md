# Retrospective Configuration

Each retrospective can be configured with advanced options to adapt to the team's needs.

## Configuration Options

### Voting

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `maxVotesPerUser` | int | 5 | Total votes per user |
| `maxVotesPerItem` | int | 3 | Max votes on a single item |
| `anonymousVoting` | bool | false | Hide who voted |
| `allowVoteChange` | bool | true | Allow removing votes |

### Items

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `anonymousItems` | bool | false | Hide item authors |
| `allowItemEdit` | bool | true | Allow editing after creation |

### Timers

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `phaseTimerOverrides` | object | null | Override phase durations |

## Creating a Retrospective

### REST API

```bash
POST /api/v1/retrospectives
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Sprint 42 Retro",
  "teamId": "550e8400-e29b-41d4-a716-446655440000",
  "templateId": "660e8400-e29b-41d4-a716-446655440001",
  "maxVotesPerUser": 5,
  "maxVotesPerItem": 3,
  "anonymousVoting": false,
  "anonymousItems": false,
  "allowItemEdit": true,
  "allowVoteChange": true,
  "phaseTimerOverrides": {
    "brainstorm": 600,
    "vote": 180,
    "discuss": 1200
  },
  "scheduledAt": "2025-01-25T14:00:00Z"
}
```

## Multiple Votes Per Item

By default, each participant can vote up to 3 times on the same item. This allows better expression of relative importance of topics.

### Behavior

1. A user can vote multiple times on the same item (up to `maxVotesPerItem`)
2. Total votes cannot exceed `maxVotesPerUser`
3. Each vote can be removed individually (if `allowVoteChange` is true)

### Example

With `maxVotesPerUser: 5` and `maxVotesPerItem: 3`:

| Item | User's Votes | Total Used |
|------|--------------|------------|
| "Improve tests" | 3 | 3 |
| "API documentation" | 2 | 5 |
| "Auth refactoring" | 0 | 5 (max reached) |

### Display

The vote counter on each item shows the current user's vote count:

```
[Item] Improve tests
       ★★★ (3 votes) - Total: 12 votes
```

## Anonymity

### Anonymous Items (`anonymousItems: true`)

- Item authors are hidden from other participants
- Facilitator can still see authors
- Useful for sensitive topics

### Anonymous Voting (`anonymousVoting: true`)

- No one can see who voted for what
- Only totals are displayed
- Recommended to avoid social bias

## Custom Timers

Default durations are defined in the template but can be overridden per retrospective.

### Available Phases

| Phase | Default Duration | Description |
|-------|-----------------|-------------|
| `waiting` | 0 | Waiting room |
| `icebreaker` | 120s | Participant mood check |
| `brainstorm` | 300s | Item creation |
| `group` | 180s | Item grouping |
| `vote` | 180s | Voting phase |
| `discuss` | 900s | Item discussion |
| `action` | 300s | Action creation |
| `roti` | 120s | ROTI voting |

### Configuration Example

```json
{
  "phaseTimerOverrides": {
    "brainstorm": 600,
    "discuss": 1800,
    "action": 600
  }
}
```

Phases not specified use template values.

## Modification After Creation

### Modifiable Options

Some options can be modified after creation:

```bash
PUT /api/v1/retrospectives/{retroId}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Sprint 42 Retro (updated)",
  "maxVotesPerUser": 7,
  "maxVotesPerItem": 4,
  "phaseTimerOverrides": {
    "discuss": 1200
  }
}
```

### Restrictions

- `teamId` and `templateId` cannot be changed
- Voting options should not be changed once voting phase has started
- Facilitator can only be changed during `waiting` phase (see [Dynamic Facilitator](./dynamic-facilitator.md))

## Best Practices

### For Small Teams (< 5 people)

```json
{
  "maxVotesPerUser": 3,
  "maxVotesPerItem": 2,
  "anonymousVoting": false,
  "anonymousItems": false,
  "phaseTimerOverrides": {
    "brainstorm": 180,
    "discuss": 600
  }
}
```

### For Large Teams (> 10 people)

```json
{
  "maxVotesPerUser": 7,
  "maxVotesPerItem": 3,
  "anonymousVoting": true,
  "anonymousItems": true,
  "phaseTimerOverrides": {
    "brainstorm": 600,
    "vote": 300,
    "discuss": 1800
  }
}
```

### For Sensitive Topics

```json
{
  "anonymousVoting": true,
  "anonymousItems": true,
  "allowItemEdit": false,
  "allowVoteChange": false
}
```

## Validation

### Validation Rules

| Option | Validation |
|--------|------------|
| `maxVotesPerUser` | >= 1 |
| `maxVotesPerItem` | >= 1, <= maxVotesPerUser |
| `phaseTimerOverrides` | >= 0 for each phase |

### Common Errors

**400 Bad Request** - Invalid configuration

```json
{
  "error": "maxVotesPerItem cannot exceed maxVotesPerUser"
}
```

```json
{
  "error": "phase timer must be positive"
}
```

## Related Documentation

- [Templates](./templates.md) - Available retrospective templates
- [Dynamic Facilitator](./dynamic-facilitator.md) - Changing facilitator during session
- [Webhooks](./webhooks.md) - Event notifications
