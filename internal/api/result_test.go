package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResult(t *testing.T) {
	t.Run("Success returns success true", func(t *testing.T) {
		result := Success()
		assert.True(t, result.IsSuccess)
		assert.Empty(t, result.Error)
	})

	t.Run("Error returns success false", func(t *testing.T) {
		result := Error("some error")
		assert.False(t, result.IsSuccess)
		assert.Equal(t, "some error", result.Error)
	})
}
