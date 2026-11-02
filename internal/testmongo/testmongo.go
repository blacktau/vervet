// Package testmongo says which MongoDB server the integration tests run against.
package testmongo

import "os"

// Image is the container image for the test server. VERVET_TEST_MONGO_IMAGE
// picks another version; CI uses it to test every supported one.
func Image() string {
	if image := os.Getenv("VERVET_TEST_MONGO_IMAGE"); image != "" {
		return image
	}
	return "mongo:7"
}
