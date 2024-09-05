package goutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptional(t *testing.T) {
	t.Run("NewOptional", func(t *testing.T) {
		opt := NewOptional(42)
		assert.True(t, opt.IsPresent())
		value, err := opt.Get()
		assert.NoError(t, err)
		assert.Equal(t, 42, value)
	})

	t.Run("Empty", func(t *testing.T) {
		opt := Empty[int]()
		assert.False(t, opt.IsPresent())
	})

	t.Run("Get", func(t *testing.T) {
		opt := NewOptional("test")
		value, err := opt.Get()
		assert.NoError(t, err)
		assert.Equal(t, "test", value)

		emptyOpt := Empty[string]()
		_, err = emptyOpt.Get()
		assert.Error(t, err)
	})

	t.Run("OrElse", func(t *testing.T) {
		opt := NewOptional(10)
		assert.Equal(t, 10, opt.OrElse(20))

		emptyOpt := Empty[int]()
		assert.Equal(t, 20, emptyOpt.OrElse(20))
	})

	t.Run("IfPresent", func(t *testing.T) {
		opt := NewOptional(5)
		called := false
		opt.IfPresent(func(v int) {
			called = true
			assert.Equal(t, 5, v)
		})
		assert.True(t, called)

		emptyOpt := Empty[int]()
		emptyOpt.IfPresent(func(v int) {
			t.Fail() // This should not be called
		})
	})

	t.Run("Map", func(t *testing.T) {
		opt := NewOptional(3)
		mapped := opt.Map(func(v int) int { return v * 2 })
		assert.True(t, mapped.IsPresent())
		value, err := mapped.Get()
		assert.NoError(t, err)
		assert.Equal(t, 6, value)

		emptyOpt := Empty[int]()
		mappedEmpty := emptyOpt.Map(func(v int) int { return v * 2 })
		assert.False(t, mappedEmpty.IsPresent())
	})

	t.Run("String", func(t *testing.T) {
		opt := NewOptional("hello")
		assert.Equal(t, "Optional[hello]", opt.String())

		emptyOpt := Empty[string]()
		assert.Equal(t, "Optional.Empty", emptyOpt.String())
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		var opt Optional[int]

		err := json.Unmarshal([]byte("42"), &opt)
		assert.NoError(t, err)
		assert.True(t, opt.IsPresent())
		value, err := opt.Get()
		assert.NoError(t, err)
		assert.Equal(t, 42, value)

		err = json.Unmarshal([]byte("null"), &opt)
		assert.NoError(t, err)
		assert.False(t, opt.IsPresent())

		err = json.Unmarshal([]byte(`"invalid"`), &opt)
		assert.Error(t, err)
	})
}

func TestOptionalUnmarshalJSONInStruct(t *testing.T) {
	type TestStruct struct {
		Name    string           `json:"name"`
		Age     Optional[int]    `json:"age"`
		Address Optional[string] `json:"address"`
	}

	t.Run("Present values", func(t *testing.T) {
		jsonData := `{
			"name": "John Doe",
			"age": 30,
			"address": "123 Main St"
		}`

		var result TestStruct
		err := json.Unmarshal([]byte(jsonData), &result)

		assert.NoError(t, err)
		assert.Equal(t, "John Doe", result.Name)
		assert.True(t, result.Age.IsPresent())
		ageValue, err := result.Age.Get()
		assert.NoError(t, err)
		assert.Equal(t, 30, ageValue)
		assert.True(t, result.Address.IsPresent())
		addressValue, err := result.Address.Get()
		assert.NoError(t, err)
		assert.Equal(t, "123 Main St", addressValue)
	})

	t.Run("Missing optional values", func(t *testing.T) {
		jsonData := `{
			"name": "Jane Doe"
		}`

		var result TestStruct
		err := json.Unmarshal([]byte(jsonData), &result)

		assert.NoError(t, err)
		assert.Equal(t, "Jane Doe", result.Name)
		assert.False(t, result.Age.IsPresent())
		assert.False(t, result.Address.IsPresent())
	})

	t.Run("Null optional values", func(t *testing.T) {
		jsonData := `{
			"name": "Bob Smith",
			"age": null,
			"address": "null"
		}`

		var result TestStruct
		err := json.Unmarshal([]byte(jsonData), &result)

		assert.NoError(t, err)
		assert.Equal(t, "Bob Smith", result.Name)
		assert.False(t, result.Age.IsPresent())
		assert.True(t, result.Address.IsPresent())
	})

	t.Run("Invalid type for optional value", func(t *testing.T) {
		jsonData := `{
			"name": "Alice Johnson",
			"age": "thirty"
		}`

		var result TestStruct
		err := json.Unmarshal([]byte(jsonData), &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal")
	})
}
