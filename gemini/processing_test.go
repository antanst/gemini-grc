package gemini

import "testing"

// Make sure NULL bytes are removed
func TestEnsureValidUTF8(t *testing.T) {
	// Create a string with a null byte
	strWithNull := "Hello" + string('\x00') + "world"
	result, _ := EnsureValidUTF8([]byte(strWithNull))
	if result != "Helloworld" {
		t.Errorf("Expected string without NULL byte, got %s", result)
	}
}