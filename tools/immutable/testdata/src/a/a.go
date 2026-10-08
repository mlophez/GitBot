// Package a holds the protected aggregate used by the analyzer test, together
// with the assignments that are legal because they happen in its own methods.
package a

// Aggregate is the protected type: "a.Aggregate" is the name the test protects.
type Aggregate struct {
	Name   string
	Locked bool
	Count  int
	Nested Nested
}

// Nested is a plain field struct, not protected: assigning its fields is only
// reported when it is reached through a protected value.
type Nested struct {
	Label string
}

// Lock is a state transition: assigning fields of the receiver is allowed.
func (a Aggregate) Lock() Aggregate {
	a.Locked = true
	a.Count++
	a.Nested.Label = "locked"
	return a
}

// Rename mutates through a pointer receiver, which is still a method of the type.
func (a *Aggregate) Rename(name string) {
	a.Name = name
}

// Holder is another type of the same package holding an aggregate, used to check
// that being in the same package grants no permission.
type Holder struct {
	Item Aggregate
}

// Touch mutates the aggregate held by a pointer receiver: shared state.
func (h *Holder) Touch() {
	h.Item.Locked = true // want "field Locked of a.Aggregate is mutated outside its own methods"
}

// Build hydrates a local value, which the analyzer allows.
func Build(name string) Aggregate {
	var item Aggregate
	item.Name = name
	item.Nested.Label = name
	item.Count += 1
	return item
}

// Enrich mutates a by-value parameter: it is a copy nobody else observes.
func Enrich(item Aggregate) Aggregate {
	item.Count = 2
	return item
}

// current is a package-level aggregate: mutating it is observable.
var current Aggregate

// LockAll shows the mutations the analyzer must report.
func LockAll(items []Aggregate, byName map[string]Aggregate, pointer *Aggregate, holder *Holder) {
	current.Locked = true  // want "field Locked of a.Aggregate is mutated outside its own methods"
	items[0].Locked = true // want "field Locked of a.Aggregate is mutated outside its own methods"
	pointer.Locked = true  // want "field Locked of a.Aggregate is mutated outside its own methods"
	pointer.Count++        // want "field Count of a.Aggregate is mutated outside its own methods"
	// Reached through a pointer to the holder, so mutating a field of the nested
	// struct is still a mutation of the aggregate.
	holder.Item.Nested.Label = "x" // want "field Nested of a.Aggregate is mutated outside its own methods"

	local := &items[0]
	local.Name = "x" // want "field Name of a.Aggregate is mutated outside its own methods"

	for i := range items {
		items[i].Count = i // want "field Count of a.Aggregate is mutated outside its own methods"
	}

	// Not reported: byName holds copies that cannot be assigned at all, so only
	// the read is exercised here.
	_ = byName["k"].Locked
}

// LockInsideClosure shows that a closure declared inside a method of the type
// keeps the receiver permission, while a closure elsewhere does not.
func (a Aggregate) LockInsideClosure() Aggregate {
	apply := func() {
		a.Locked = true
	}
	apply()
	return a
}

// lockShared is a package-level function value: it has no receiver, so the
// mutation through the pointer is reported.
var lockShared = func(item *Aggregate) {
	item.Locked = true // want "field Locked of a.Aggregate is mutated outside its own methods"
}

// use keeps the unused declarations referenced.
func use() {
	lockShared(&current)
}
