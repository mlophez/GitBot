package types

type HookType int

const (
	HookUnknown HookType = iota
	HookOpened
	HookUpdated
	HookDeclined
	HookMerged
	HookCommented
)

