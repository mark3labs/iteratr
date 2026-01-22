# State Snapshots

Performance optimization: snapshot state to KV, replay only newer events.

## Overview

Snapshot session state to JetStream KV on iteration end. LoadState reads snapshot first, replays only events after snapshot sequence. Eliminates O(n) event replay for every state load.

## User Story

**As a** developer running long multi-iteration sessions  
**I want** faster state loading  
**So that** sessions with hundreds of events don't slow down iteration startup

## Requirements

### Functional

1. **Snapshot Write**
   - Snapshot state to KV bucket on iteration end
   - Include sequence number of last applied event
   - Key: session name, Value: serialized state + metadata

2. **Snapshot Read**
   - LoadState checks for snapshot first
   - If found, deserialize and replay only events after snapshot sequence
   - Fall back to full replay if no snapshot exists

3. **KV Bucket**
   - Bucket name: `iteratr_snapshots`
   - Created on orchestrator startup if not exists

### Non-Functional

1. Backward compatible - sessions work without snapshots
2. Snapshot write overhead < 10ms
3. No data loss if snapshot is stale (events always authoritative)

## Technical Implementation

### KV Snapshot Schema

```
Bucket: iteratr_snapshots
Key: {session}
Value: {
    "state": <State>,
    "after_sequence": 12345,
    "created_at": "2025-01-22T..."
}
```

### Snapshot Struct

```go
type Snapshot struct {
    State         *State    `json:"state"`
    AfterSequence uint64    `json:"after_sequence"`
    CreatedAt     time.Time `json:"created_at"`
}
```

### Optimized LoadState

```go
func (s *Store) LoadState(ctx context.Context, session string) (*State, error) {
    // Try snapshot first
    snapshot, err := s.kv.Get(ctx, session)
    if err == nil {
        state := decodeSnapshot(snapshot)
        // Replay only events after snapshot sequence
        return s.replayEventsAfter(ctx, session, state, snapshot.AfterSequence)
    }
    // Fall back to full replay
    return s.loadStateFull(ctx, session)
}
```

### New Methods

```go
// nats/store.go
func (s *Store) CreateSnapshotBucket(ctx context.Context) error
func (s *Store) WriteSnapshot(ctx context.Context, session string, state *State, afterSeq uint64) error
func (s *Store) ReadSnapshot(ctx context.Context, session string) (*Snapshot, error)
func (s *Store) replayEventsAfter(ctx context.Context, session string, state *State, afterSeq uint64) (*State, error)
```

### Integration Point

```go
// orchestrator.go - after IterationComplete
func (o *Orchestrator) endIteration(ctx context.Context) error {
    // ... existing iteration end logic ...
    
    // Write snapshot for next iteration
    seq := latestEvent.Sequence
    return o.store.WriteSnapshot(ctx, o.session, state, seq)
}
```

## Tasks

### 1. KV Store Setup
- [ ] Add `CreateSnapshotBucket` function in nats/store.go
- [ ] Create bucket `iteratr_snapshots` with appropriate config
- [ ] Call bucket creation in orchestrator startup (idempotent)

### 2. Snapshot Struct
- [ ] Add `Snapshot` struct in nats/store.go or session.go
- [ ] Fields: State, AfterSequence (uint64), CreatedAt

### 3. Snapshot Write
- [ ] Add `WriteSnapshot` method to Store
- [ ] Serialize snapshot to JSON
- [ ] Store in KV with session as key
- [ ] Return error if KV unavailable

### 4. Snapshot Read
- [ ] Add `ReadSnapshot` method to Store
- [ ] Deserialize JSON to Snapshot struct
- [ ] Return nil, error if not found (not a failure case)

### 5. Partial Event Replay
- [ ] Add `replayEventsAfter` helper method
- [ ] Accept state, afterSequence params
- [ ] Query stream for events with seq > afterSequence
- [ ] Apply events to provided state

### 6. Optimized LoadState
- [ ] Modify `LoadState` to check snapshot first via ReadSnapshot
- [ ] If snapshot exists, call replayEventsAfter
- [ ] If no snapshot, fall back to existing full replay
- [ ] Ensure backward compat (old sessions without snapshots still work)

### 7. Snapshot on Iteration End
- [ ] Identify integration point in orchestrator iteration end flow
- [ ] Call `WriteSnapshot` after `IterationComplete` succeeds
- [ ] Pass current sequence number from latest event
- [ ] Handle write errors gracefully (log, don't fail iteration)

### 8. Tests
- [ ] Test WriteSnapshot stores state correctly
- [ ] Test ReadSnapshot retrieves state
- [ ] Test ReadSnapshot returns error when no snapshot
- [ ] Test LoadState uses snapshot when available
- [ ] Test LoadState replays only events after snapshot
- [ ] Test LoadState falls back when no snapshot
- [ ] Test snapshot survives orchestrator restart

## Out of Scope

- Snapshot pruning/cleanup (manual deletion if needed)
- Snapshot compression (premature optimization)
- Multiple snapshots per session (only latest needed)
- Snapshot on every task edit (iteration end sufficient)

## Open Questions

1. Should snapshots be written on every task edit, or only iteration end? (Current: iteration end only)
2. TTL for snapshots? (Current: none, persist indefinitely)
3. Should snapshot failures be surfaced to user or silently logged?

## Dependencies

- NATS JetStream KV (already available)
- Existing State struct and event replay logic
