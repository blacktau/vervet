package connections

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "no credentials",
			uri:      "mongodb://localhost:27017",
			expected: "mongodb://localhost:27017",
		},
		{
			name:     "with credentials",
			uri:      "mongodb://user:password@localhost:27017",
			expected: "mongodb://user:***localhost:27017",
		},
		{
			name:     "with credentials and database",
			uri:      "mongodb://admin:secret123@mongodb.example.com:27017/mydb",
			expected: "mongodb://admin:***mongodb.example.com:27017/mydb",
		},
		{
			name:     "complex credentials",
			uri:      "mongodb://user:pass@word@localhost:27017",
			expected: "mongodb://user:***word@localhost:27017",
		},
		{
			name:     "at symbol in password",
			uri:      "mongodb://user:p@ssw@rd@localhost:27017",
			expected: "mongodb://user:***ssw@rd@localhost:27017",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanConnectionString(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}
