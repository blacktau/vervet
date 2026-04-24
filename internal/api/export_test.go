package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockExportProvider implements ExportProvider for tests.
type mockExportProvider struct {
	path string
	err  error
}

func (m *mockExportProvider) Export(_ ExportRequest) (string, error) {
	return m.path, m.err
}

func TestExportProxy_ExportResults_Success(t *testing.T) {
	provider := &mockExportProvider{path: "/tmp/results.json"}
	proxy := NewExportProxy(testLogger(), provider)

	result := proxy.ExportResults(ExportRequest{
		Format: "json",
		EJSON:  `[{"a":1}]`,
	})

	assert.True(t, result.IsSuccess)
	assert.Equal(t, "/tmp/results.json", result.Data)
	assert.Empty(t, result.ErrorCode)
}

func TestExportProxy_ExportResults_Cancelled(t *testing.T) {
	// Empty path + nil err = user cancelled the save dialog.
	provider := &mockExportProvider{path: ""}
	proxy := NewExportProxy(testLogger(), provider)

	result := proxy.ExportResults(ExportRequest{
		Format: "json",
		EJSON:  `[{"a":1}]`,
	})

	assert.True(t, result.IsSuccess)
	assert.Empty(t, result.Data)
	assert.Empty(t, result.ErrorCode)
}

func TestExportProxy_ExportResults_ProviderError(t *testing.T) {
	provider := &mockExportProvider{err: errors.New("serialize failed")}
	proxy := NewExportProxy(testLogger(), provider)

	result := proxy.ExportResults(ExportRequest{
		Format: "json",
		EJSON:  `[{"a":1}]`,
	})

	assert.False(t, result.IsSuccess)
	assert.Empty(t, result.Data)
	assert.NotEmpty(t, result.ErrorCode)
}
