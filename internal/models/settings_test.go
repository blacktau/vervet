package models_test

import (
	"testing"
	"vervet/internal/models"

	"github.com/stretchr/testify/assert"
)

func Test_LoggingSettings_Normalize(t *testing.T) {
	t.Run("empty level becomes info", func(t *testing.T) {
		s := models.LoggingSettings{Level: "", MaxSizeMB: 5, MaxBackups: 3}
		s.Normalize()
		assert.Equal(t, "info", s.Level)
	})

	t.Run("unknown level becomes info", func(t *testing.T) {
		s := models.LoggingSettings{Level: "verbose", MaxSizeMB: 5, MaxBackups: 3}
		s.Normalize()
		assert.Equal(t, "info", s.Level)
	})

	t.Run("recognised levels are preserved", func(t *testing.T) {
		for _, lvl := range []string{"debug", "info", "warn", "warning", "error"} {
			s := models.LoggingSettings{Level: lvl, MaxSizeMB: 5, MaxBackups: 3}
			s.Normalize()
			assert.Equal(t, lvl, s.Level, "level %q should be preserved", lvl)
		}
	})

	t.Run("MaxSizeMB below 1 clamps to 1", func(t *testing.T) {
		s := models.LoggingSettings{Level: "info", MaxSizeMB: 0, MaxBackups: 3}
		s.Normalize()
		assert.Equal(t, 1, s.MaxSizeMB)

		s = models.LoggingSettings{Level: "info", MaxSizeMB: -5, MaxBackups: 3}
		s.Normalize()
		assert.Equal(t, 1, s.MaxSizeMB)
	})

	t.Run("MaxSizeMB at or above 1 is preserved", func(t *testing.T) {
		s := models.LoggingSettings{Level: "info", MaxSizeMB: 1, MaxBackups: 3}
		s.Normalize()
		assert.Equal(t, 1, s.MaxSizeMB)

		s = models.LoggingSettings{Level: "info", MaxSizeMB: 100, MaxBackups: 3}
		s.Normalize()
		assert.Equal(t, 100, s.MaxSizeMB)
	})

	t.Run("MaxBackups below zero clamps to zero", func(t *testing.T) {
		s := models.LoggingSettings{Level: "info", MaxSizeMB: 5, MaxBackups: -1}
		s.Normalize()
		assert.Equal(t, 0, s.MaxBackups)
	})

	t.Run("MaxBackups zero is preserved", func(t *testing.T) {
		s := models.LoggingSettings{Level: "info", MaxSizeMB: 5, MaxBackups: 0}
		s.Normalize()
		assert.Equal(t, 0, s.MaxBackups)
	})

	t.Run("MaxBackups positive is preserved", func(t *testing.T) {
		s := models.LoggingSettings{Level: "info", MaxSizeMB: 5, MaxBackups: 7}
		s.Normalize()
		assert.Equal(t, 7, s.MaxBackups)
	})
}
