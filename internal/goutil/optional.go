package goutil

import (
	"encoding/json"
	"fmt"
)

// Optional represents a value that may or may not be present.
type Optional[T any] struct {
	value T
	isSet bool
}

// NewOptional creates a new Optional with the given value.
func NewOptional[T any](value T) Optional[T] {
	return Optional[T]{
		value: value,
		isSet: true,
	}
}

// Empty returns an empty Optional.
func Empty[T any]() Optional[T] {
	return Optional[T]{}
}

// IsPresent returns true if the Optional contains a value.
func (o Optional[T]) IsPresent() bool {
	return o.isSet
}

// Get returns the value if present, or panics if not present.
func (o Optional[T]) Get() T {
	if !o.isSet {
		panic("Optional value is not present")
	}
	return o.value
}

// OrElse returns the value if present, or the given default value if not present.
func (o Optional[T]) OrElse(defaultValue T) T {
	if o.isSet {
		return o.value
	}
	return defaultValue
}

// IfPresent calls the given function with the value if present.
func (o Optional[T]) IfPresent(f func(T)) {
	if o.isSet {
		f(o.value)
	}
}

// Map applies the given function to the value if present and returns a new Optional.
func (o Optional[T]) Map(f func(T) T) Optional[T] {
	if !o.isSet {
		return Empty[T]()
	}
	return NewOptional(f(o.value))
}

// String returns a string representation of the Optional.
func (o Optional[T]) String() string {
	if !o.isSet {
		return "Optional.Empty"
	}
	return fmt.Sprintf("Optional[%v]", o.value)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	// Check if it's null first
	if string(data) == "null" {
		*o = Empty[T]()
		return nil
	}

	// If not null, try to unmarshal into the value type T
	var value T
	err := json.Unmarshal(data, &value)
	if err == nil {
		*o = NewOptional(value)
		return nil
	}

	// If it's neither a valid T nor null, return an error
	return fmt.Errorf("cannot unmarshal %s into Optional[T]", string(data))
}
