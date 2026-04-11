package event

import "gitbot/internal/types"

// Type aliases — the canonical definitions live in internal/types.

type EventType = types.EventType
type Event = types.Event

const (
	EventTypeUnknown   = types.EventTypeUnknown
	EventTypeOpened    = types.EventTypeOpened
	EventTypeUpdated   = types.EventTypeUpdated
	EventTypeDeclined  = types.EventTypeDeclined
	EventTypeMerged    = types.EventTypeMerged
	EventTypeCommented = types.EventTypeCommented
)
