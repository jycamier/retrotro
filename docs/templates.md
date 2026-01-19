# Retrospective Templates

Templates define the structure and flow of a retrospective. Retrotro includes several built-in templates and supports custom templates.

## Built-in Templates

### Mad/Sad/Glad

The classic retrospective format focusing on emotions.

| Column | Description | Color |
|--------|-------------|-------|
| Mad | Things that frustrated the team | Red |
| Sad | Things that disappointed or were missed | Blue |
| Glad | Things that went well | Green |

**Best for:** Teams new to retrospectives, emotional check-ins

### Start/Stop/Continue

Action-oriented format focusing on behavior changes.

| Column | Description | Color |
|--------|-------------|-------|
| Start | Things we should begin doing | Green |
| Stop | Things we should stop doing | Red |
| Continue | Things we should keep doing | Blue |

**Best for:** Teams focused on process improvement

### 4Ls (Liked/Learned/Lacked/Longed For)

Comprehensive format covering multiple dimensions.

| Column | Description | Color |
|--------|-------------|-------|
| Liked | Things that went well | Green |
| Learned | New knowledge or skills gained | Blue |
| Lacked | Things that were missing | Orange |
| Longed For | Things we wish we had | Purple |

**Best for:** End of sprint or project retrospectives

### Lean Coffee

Structured discussion format with time-boxed topics.

| Column | Description | Color |
|--------|-------------|-------|
| Topics | Subjects to discuss | Amber |
| On Discuss | Topic currently being discussed | Blue |
| Done | Topics already discussed | Green |

**Best for:** Teams that want participant-driven agendas, open discussions

#### How Lean Coffee Works

1. **Brainstorm:** Participants add topics they want to discuss
2. **Vote:** Everyone votes on the topics they find most important
3. **Discuss:** Topics are discussed in priority order
4. **Move:** Topics move from "Topics" → "On Discuss" → "Done"
5. **Timeboxes:** Each topic gets a fixed time (e.g., 5 minutes) with optional extensions

## Template Structure

Each template is defined with columns and phase timers:

```json
{
  "id": "uuid",
  "name": "Template Name",
  "description": "Description of the template",
  "columns": [
    {
      "id": "column-1",
      "name": "Column Name",
      "description": "What goes in this column",
      "color": "#3b82f6",
      "icon": "thumbs-up",
      "order": 0
    }
  ],
  "phaseTimes": {
    "brainstorm": 300,
    "group": 180,
    "vote": 180,
    "discuss": 900,
    "action": 300
  },
  "isBuiltIn": true
}
```

### Column Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier within template |
| `name` | string | Display name |
| `description` | string | Helper text for participants |
| `color` | string | Hex color code |
| `icon` | string | Icon name (optional) |
| `order` | int | Display order (0-based) |

### Phase Timers

Default durations (in seconds) for each phase:

| Phase | Default | Description |
|-------|---------|-------------|
| `waiting` | 0 | Waiting room (no timer) |
| `icebreaker` | 120 | Mood sharing |
| `brainstorm` | 300 | Item creation |
| `group` | 180 | Item grouping |
| `vote` | 180 | Voting |
| `discuss` | 900 | Discussion |
| `action` | 300 | Action creation |
| `roti` | 120 | ROTI voting |

## Custom Templates

### Creating a Custom Template

```bash
POST /api/v1/templates
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Team Health Check",
  "description": "Monthly team health assessment",
  "teamId": "550e8400-e29b-41d4-a716-446655440000",
  "columns": [
    {
      "id": "delivery",
      "name": "Delivery",
      "description": "Are we delivering value?",
      "color": "#22c55e",
      "order": 0
    },
    {
      "id": "quality",
      "name": "Quality",
      "description": "Is our work high quality?",
      "color": "#3b82f6",
      "order": 1
    },
    {
      "id": "fun",
      "name": "Fun",
      "description": "Are we enjoying our work?",
      "color": "#f59e0b",
      "order": 2
    },
    {
      "id": "learning",
      "name": "Learning",
      "description": "Are we growing?",
      "color": "#8b5cf6",
      "order": 3
    }
  ],
  "phaseTimes": {
    "brainstorm": 600,
    "vote": 120,
    "discuss": 1200,
    "action": 300
  }
}
```

### Template Visibility

| Type | Visibility |
|------|------------|
| Built-in | All teams |
| Custom (with teamId) | Only specified team |
| Custom (no teamId) | All teams (admin only) |

## Listing Templates

```bash
GET /api/v1/templates
GET /api/v1/templates?teamId=550e8400-e29b-41d4-a716-446655440000
```

Returns built-in templates plus custom templates for the specified team.

## Using a Template

When creating a retrospective, specify the template ID:

```bash
POST /api/v1/retrospectives
Content-Type: application/json

{
  "name": "Sprint 42 Retro",
  "teamId": "team-uuid",
  "templateId": "template-uuid"
}
```

## Popular Template Variations

### DAKI (Drop/Add/Keep/Improve)

```json
{
  "name": "DAKI",
  "columns": [
    { "id": "drop", "name": "Drop", "color": "#ef4444" },
    { "id": "add", "name": "Add", "color": "#22c55e" },
    { "id": "keep", "name": "Keep", "color": "#3b82f6" },
    { "id": "improve", "name": "Improve", "color": "#f59e0b" }
  ]
}
```

### Sailboat

```json
{
  "name": "Sailboat",
  "columns": [
    { "id": "wind", "name": "Wind (Helps)", "color": "#22c55e" },
    { "id": "anchor", "name": "Anchor (Slows)", "color": "#ef4444" },
    { "id": "rocks", "name": "Rocks (Risks)", "color": "#f59e0b" },
    { "id": "island", "name": "Island (Goals)", "color": "#3b82f6" }
  ]
}
```

### Starfish

```json
{
  "name": "Starfish",
  "columns": [
    { "id": "keep", "name": "Keep Doing", "color": "#22c55e" },
    { "id": "less", "name": "Less Of", "color": "#ef4444" },
    { "id": "more", "name": "More Of", "color": "#3b82f6" },
    { "id": "stop", "name": "Stop Doing", "color": "#f97316" },
    { "id": "start", "name": "Start Doing", "color": "#8b5cf6" }
  ]
}
```

## Best Practices

1. **Match template to context** - Use emotional templates (Mad/Sad/Glad) after difficult sprints, action-oriented (Start/Stop/Continue) for process improvement
2. **Rotate templates** - Using the same template every time can become stale
3. **Customize timers** - Adjust phase durations based on team size
4. **Create team-specific templates** - Tailor to your team's vocabulary and needs

## Related Documentation

- [Configuration](./configuration.md) - Override timer settings per retrospective
- [Dynamic Facilitator](./dynamic-facilitator.md) - Change facilitator during session
