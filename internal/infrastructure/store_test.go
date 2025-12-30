package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStore(t *testing.T) {
	store, err := NewStore("test.yaml", nil)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	configDir, err := getConfigDirectory()
	if err != nil {
		t.Fatalf("could not get config directory: %v", err)
	}
	expectedPath := filepath.Join(configDir, "test.yaml")
	if store.(*cfgStore).ConfigPath != expectedPath {
		t.Errorf("expected config path %s, got %s", expectedPath, store.(*cfgStore).ConfigPath)
	}
}

func TestCfgStore_Read(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_config.yaml")

	store := &cfgStore{ConfigPath: testFile}
	data, err := store.Read()
	if !os.IsNotExist(err) {
		t.Errorf("expected 'file does not exist' error, got %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty data, got %s", data)
	}

	expectedData := []byte("hello: world")
	err = os.WriteFile(testFile, expectedData, 0600)
	if err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	data, err = store.Read()
	if err != nil {
		t.Errorf("unexpected error reading file: %v", err)
	}
	if string(data) != string(expectedData) {
		t.Errorf("expected data '%s', got '%s'", expectedData, data)
	}
}

func TestCfgStore_Save(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_config.yaml")
	store := &cfgStore{ConfigPath: testFile}

	saveData := []byte("foo: bar")
	err := store.Save(saveData)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	readData, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read back saved file: %v", err)
	}

	if string(readData) != string(saveData) {
		t.Errorf("expected saved data '%s', got '%s'", saveData, readData)
	}
}
