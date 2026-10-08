// Package b uses the protected type from another package, which is the main case
// the rule targets: no state transition may be done from the outside.
package b

import "a"

// Store keeps an aggregate and tries to mutate it from outside its package.
type Store struct {
	Item a.Aggregate
}

// LockItem mutates shared state through a pointer receiver.
func (s *Store) LockItem() {
	s.Item.Locked = true // want "field Locked of a.Aggregate is mutated outside its own methods"
}

// LockCopy hydrates its own local copy, which is allowed, and uses the method for
// the actual transition.
func LockCopy(item a.Aggregate) a.Aggregate {
	local := item
	local.Name = "copy"
	return local.Lock()
}

// LockPointer mutates the caller's value: reported.
func LockPointer(item *a.Aggregate) {
	item.Locked = true // want "field Locked of a.Aggregate is mutated outside its own methods"
}
