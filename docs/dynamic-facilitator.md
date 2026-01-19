# Dynamic Facilitator

The dynamic facilitator feature allows changing the retrospective facilitator during the waiting room phase, before the retrospective officially starts.

## Overview

By default, the person who creates a retrospective becomes its facilitator. However, teams often want flexibility to:

- Let someone else run the retro if the creator can't attend
- Rotate facilitator duties among team members
- Have a team lead delegate facilitation to a team member

## How It Works

### During Waiting Phase

While participants are gathering in the waiting room, the facilitator can be changed:

1. **Team admins** can claim the facilitator role
2. **Current facilitator** can transfer the role to any connected member

### After Retro Starts

Once the retrospective moves past the waiting phase (icebreaker begins), the facilitator is **locked** and cannot be changed.

## WebSocket Messages

### Claiming Facilitator Role

A team admin can claim the role:

```json
// Client → Server
{
  "type": "facilitator_claim"
}

// Server → All Clients
{
  "type": "facilitator_changed",
  "payload": {
    "facilitatorId": "user-uuid",
    "facilitatorName": "John Doe"
  }
}
```

### Transferring Facilitator Role

The current facilitator can transfer to another member:

```json
// Client → Server
{
  "type": "facilitator_transfer",
  "payload": {
    "userId": "target-user-uuid"
  }
}

// Server → All Clients
{
  "type": "facilitator_changed",
  "payload": {
    "facilitatorId": "target-user-uuid",
    "facilitatorName": "Jane Doe"
  }
}
```

## Permissions

### Who Can Claim Facilitator?

| Role | Can Claim? |
|------|------------|
| Team Admin | Yes |
| Team Member | No |

### Who Can Transfer?

Only the **current facilitator** can transfer the role.

### Who Can Receive?

Any **connected member** can receive the facilitator role, regardless of their team role.

## Error Cases

### Retro Already Started

```json
{
  "type": "error",
  "payload": {
    "message": "Cannot change facilitator after retro has started"
  }
}
```

### Insufficient Permissions

```json
{
  "type": "error",
  "payload": {
    "message": "Only admins can claim the facilitator role"
  }
}
```

### Target User Not Connected

```json
{
  "type": "error",
  "payload": {
    "message": "Target user is not connected to this retrospective"
  }
}
```

### Not Current Facilitator

```json
{
  "type": "error",
  "payload": {
    "message": "Only the current facilitator can transfer the role"
  }
}
```

## Frontend Implementation

### Waiting Room UI

```tsx
function WaitingRoom({ retro, currentUser, connectedUsers }) {
  const isFacilitator = retro.facilitatorId === currentUser.id;
  const canClaimFacilitator =
    !isFacilitator &&
    currentUser.teamRole === 'admin';

  return (
    <div>
      <h2>Waiting Room</h2>

      <p>
        Facilitator: {retro.facilitatorName}
        {isFacilitator && " (You)"}
      </p>

      {/* Claim button for admins */}
      {canClaimFacilitator && (
        <button onClick={() => ws.send({ type: 'facilitator_claim' })}>
          Become Facilitator
        </button>
      )}

      {/* Transfer dropdown for current facilitator */}
      {isFacilitator && (
        <select
          onChange={(e) => ws.send({
            type: 'facilitator_transfer',
            payload: { userId: e.target.value }
          })}
        >
          <option value="">Transfer facilitator to...</option>
          {connectedUsers
            .filter(u => u.id !== currentUser.id)
            .map(u => (
              <option key={u.id} value={u.id}>{u.displayName}</option>
            ))
          }
        </select>
      )}

      {/* Connected participants list */}
      <h3>Participants ({connectedUsers.length})</h3>
      <ul>
        {connectedUsers.map(u => (
          <li key={u.id}>
            {u.displayName}
            {u.id === retro.facilitatorId && " ⭐"}
          </li>
        ))}
      </ul>

      {/* Only facilitator sees start button */}
      {isFacilitator && (
        <button onClick={() => ws.send({ type: 'start_retro' })}>
          Start Retrospective
        </button>
      )}
    </div>
  );
}
```

### Handling facilitator_changed Event

```tsx
useEffect(() => {
  ws.on('facilitator_changed', (payload) => {
    setRetro(prev => ({
      ...prev,
      facilitatorId: payload.facilitatorId,
      facilitatorName: payload.facilitatorName
    }));

    // Show notification
    if (payload.facilitatorId === currentUser.id) {
      toast.success("You are now the facilitator!");
    } else {
      toast.info(`${payload.facilitatorName} is now the facilitator`);
    }
  });
}, [ws, currentUser]);
```

## Use Cases

### Scenario 1: Creator Can't Attend

1. Alice creates a retro scheduled for tomorrow
2. Alice gets sick and can't attend
3. Bob (team admin) joins the waiting room
4. Bob clicks "Become Facilitator"
5. Bob runs the retro

### Scenario 2: Rotating Facilitation

1. Charlie (admin) creates a retro
2. The team wants Diana to facilitate today for practice
3. Diana joins the waiting room
4. Charlie transfers facilitator role to Diana
5. Diana runs the retro

### Scenario 3: Last-Minute Change

1. Eve creates and joins the waiting room
2. Eve realizes she needs to leave for an urgent call
3. Frank (team admin) is already connected
4. Eve transfers the role to Frank before leaving
5. Frank runs the retro

## Facilitator Responsibilities

The facilitator has exclusive abilities:

| Action | Facilitator Only |
|--------|-----------------|
| Start retrospective | Yes |
| Change phases | Yes |
| Control timer | Yes |
| End retrospective | Yes |
| Reveal ROTI results | Yes |
| Create/edit items | No (all participants) |
| Vote | No (all participants) |
| Create actions | No (all participants) |

## Related Documentation

- [Configuration](./configuration.md) - Retrospective configuration options
- [Templates](./templates.md) - Available retrospective templates
- [Webhooks](./webhooks.md) - Event notifications
