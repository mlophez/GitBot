package types

type Action int

const (
	ActionNothing Action = iota
	ActionLock
	ActionUnlock
)
