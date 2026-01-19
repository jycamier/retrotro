# Webhooks

Webhooks allow you to send HTTP notifications to external services when events occur in Retrotro.

## Available Events

| Event | Description | Trigger |
|-------|-------------|---------|
| `retro.completed` | A retrospective has ended | Facilitator ends the retro |
| `action.created` | An action item was created | Participant creates an action |

## Configuration

### Create a Webhook

```bash
POST /api/v1/teams/{teamId}/webhooks
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Slack Notifications",
  "url": "https://hooks.slack.com/services/xxx/yyy/zzz",
  "secret": "my-optional-secret",
  "events": ["retro.completed", "action.created"],
  "isEnabled": true
}
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Webhook name |
| `url` | string | Yes | Destination URL |
| `secret` | string | No | Secret for HMAC-SHA256 signing |
| `events` | string[] | Yes | List of events to subscribe to |
| `isEnabled` | boolean | No | Enable/disable (default: true) |

## Payloads

### retro.completed

Sent when a retrospective is completed.

```json
{
  "event": "retro.completed",
  "timestamp": "2025-01-22T15:30:00Z",
  "retroId": "550e8400-e29b-41d4-a716-446655440000",
  "teamId": "660e8400-e29b-41d4-a716-446655440001",
  "data": {
    "name": "Sprint 42 Retro",
    "facilitatorId": "770e8400-e29b-41d4-a716-446655440002",
    "participantCount": 8,
    "itemCount": 24,
    "actionCount": 5,
    "averageRoti": 3.8,
    "moods": [
      { "userId": "uuid-1", "mood": "sunny" },
      { "userId": "uuid-2", "mood": "partly_cloudy" },
      { "userId": "uuid-3", "mood": "cloudy" },
      { "userId": "uuid-4", "mood": "rainy" },
      { "userId": "uuid-5", "mood": "stormy" }
    ],
    "rotiVotes": [
      { "userId": "uuid-1", "rating": 4 },
      { "userId": "uuid-2", "rating": 3 }
    ]
  }
}
```

#### Data Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Retrospective name |
| `facilitatorId` | uuid | Facilitator's user ID |
| `participantCount` | int | Number of participants |
| `itemCount` | int | Number of items created |
| `actionCount` | int | Number of actions created |
| `averageRoti` | float | Average ROTI rating (1-5) |
| `moods` | array | List of participant moods |
| `rotiVotes` | array | Individual ROTI votes |

#### Mood Values

- `sunny` - Very positive
- `partly_cloudy` - Somewhat positive
- `cloudy` - Neutral
- `rainy` - Somewhat negative
- `stormy` - Very negative

### action.created

Sent when an action item is created during a retrospective.

```json
{
  "event": "action.created",
  "timestamp": "2025-01-22T15:25:00Z",
  "retroId": "550e8400-e29b-41d4-a716-446655440000",
  "teamId": "660e8400-e29b-41d4-a716-446655440001",
  "data": {
    "actionId": "880e8400-e29b-41d4-a716-446655440003",
    "title": "Improve API documentation",
    "description": "Add request/response examples",
    "assigneeId": "990e8400-e29b-41d4-a716-446655440004",
    "assigneeName": "John Doe",
    "dueDate": "2025-02-01T00:00:00Z",
    "priority": 1,
    "createdBy": "aa0e8400-e29b-41d4-a716-446655440005",
    "sourceItemId": "bb0e8400-e29b-41d4-a716-446655440006"
  }
}
```

#### Data Fields

| Field | Type | Description |
|-------|------|-------------|
| `actionId` | uuid | Action ID |
| `title` | string | Action title |
| `description` | string? | Detailed description |
| `assigneeId` | uuid? | Assigned user's ID |
| `assigneeName` | string? | Assigned user's name |
| `dueDate` | datetime? | Due date |
| `priority` | int | Priority (1 = high) |
| `createdBy` | uuid | Creator's user ID |
| `sourceItemId` | uuid? | Source item ID |

## Security

### HMAC-SHA256 Signature

If a `secret` is configured, each request includes an `X-Webhook-Signature` header containing an HMAC-SHA256 signature of the payload.

```
X-Webhook-Signature: sha256=5d41402abc4b2a76b9719d911017c592
```

#### Verification in Node.js

```javascript
const crypto = require('crypto');

function verifySignature(payload, signature, secret) {
  const expected = 'sha256=' + crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');

  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}

// Usage
app.post('/webhook', (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const payload = JSON.stringify(req.body);

  if (!verifySignature(payload, signature, process.env.WEBHOOK_SECRET)) {
    return res.status(401).send('Invalid signature');
  }

  // Process webhook...
});
```

#### Verification in Python

```python
import hmac
import hashlib

def verify_signature(payload: bytes, signature: str, secret: str) -> bool:
    expected = 'sha256=' + hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(signature, expected)

# Usage with Flask
@app.route('/webhook', methods=['POST'])
def handle_webhook():
    signature = request.headers.get('X-Webhook-Signature')
    payload = request.get_data()

    if not verify_signature(payload, signature, os.environ['WEBHOOK_SECRET']):
        return 'Invalid signature', 401

    # Process webhook...
```

#### Verification in Go

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func verifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### HTTP Headers

Each webhook request includes the following headers:

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` |
| `User-Agent` | `Retrotro-Webhook/1.0` |
| `X-Webhook-Event` | Event type (e.g., `retro.completed`) |
| `X-Webhook-ID` | Webhook ID |
| `X-Webhook-Signature` | HMAC signature (if secret configured) |

## REST API

### List Webhooks

```bash
GET /api/v1/teams/{teamId}/webhooks
```

### Get Webhook

```bash
GET /api/v1/teams/{teamId}/webhooks/{webhookId}
```

### Update Webhook

```bash
PUT /api/v1/teams/{teamId}/webhooks/{webhookId}
Content-Type: application/json

{
  "name": "New name",
  "url": "https://new-url.com/webhook",
  "events": ["retro.completed"],
  "isEnabled": false
}
```

### Delete Webhook

```bash
DELETE /api/v1/teams/{teamId}/webhooks/{webhookId}
```

### Delivery History

```bash
GET /api/v1/teams/{teamId}/webhooks/{webhookId}/deliveries?limit=50
```

Returns the history of delivery attempts with:
- HTTP response status
- Response body (truncated to 1KB)
- Error message if failed
- Delivery timestamp

## Use Cases

### Slack Notification When Retro Ends

1. Create a Slack incoming webhook in your workspace
2. Configure a Retrotro webhook with `retro.completed` event
3. Use a Lambda/Cloud Function to transform the payload

```javascript
// AWS Lambda example
exports.handler = async (event) => {
  const payload = JSON.parse(event.body);

  if (payload.event === 'retro.completed') {
    const { data } = payload;

    // Detect participants with negative mood
    const unhappyParticipants = data.moods.filter(
      m => m.mood === 'rainy' || m.mood === 'stormy'
    );

    const slackMessage = {
      text: `Retro "${data.name}" completed!`,
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `*${data.name}* is complete\n` +
                  `Participants: ${data.participantCount}\n` +
                  `Items: ${data.itemCount}\n` +
                  `Actions: ${data.actionCount}\n` +
                  `Avg ROTI: ${data.averageRoti?.toFixed(1) || 'N/A'}`
          }
        }
      ]
    };

    if (unhappyParticipants.length > 0) {
      slackMessage.blocks.push({
        type: "section",
        text: {
          type: "mrkdwn",
          text: `${unhappyParticipants.length} participant(s) with negative mood`
        }
      });
    }

    await fetch(process.env.SLACK_WEBHOOK_URL, {
      method: 'POST',
      body: JSON.stringify(slackMessage)
    });
  }

  return { statusCode: 200 };
};
```

### Automatic Jira Ticket Creation

```javascript
exports.handler = async (event) => {
  const payload = JSON.parse(event.body);

  if (payload.event === 'action.created') {
    const { data } = payload;

    const jiraIssue = {
      fields: {
        project: { key: 'TEAM' },
        summary: data.title,
        description: data.description || '',
        issuetype: { name: 'Task' },
        duedate: data.dueDate?.split('T')[0],
        priority: { id: data.priority === 1 ? '2' : '3' }
      }
    };

    await fetch(`${process.env.JIRA_URL}/rest/api/2/issue`, {
      method: 'POST',
      headers: {
        'Authorization': `Basic ${process.env.JIRA_AUTH}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(jiraIssue)
    });
  }

  return { statusCode: 200 };
};
```

## Best Practices

1. **Always configure a secret** to validate webhook authenticity
2. **Respond quickly** (< 5 seconds) to avoid timeouts
3. **Process asynchronously** for long-running operations
4. **Implement idempotency** as webhooks may be replayed
5. **Log payloads** for debugging
6. **Monitor errors** via delivery history

## Related Documentation

- [Terraform Provider](./terraform-provider.md) - Manage webhooks as Infrastructure as Code
- [Configuration](./configuration.md) - Retrospective configuration options
