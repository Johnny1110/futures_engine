package common

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateUUID(t *testing.T) {
	// Test without prefix
	id1 := GenerateUUID("")
	if id1 == "" {
		t.Error("GenerateUUID() returned empty string")
	}
	
	// Validate it's a proper UUID format
	if _, err := uuid.Parse(id1); err != nil {
		t.Errorf("GenerateUUID() returned invalid UUID: %v", err)
	}

	// Test with prefix
	prefix := "test"
	id2 := GenerateUUID(prefix)
	if !strings.HasPrefix(id2, prefix+"_") {
		t.Errorf("GenerateUUID() with prefix %s should start with %s_, got %s", prefix, prefix, id2)
	}

	// Test uniqueness
	id3 := GenerateUUID("")
	if id1 == id3 {
		t.Error("GenerateUUID() should generate unique UUIDs")
	}
}

func TestGenerateShortUUID(t *testing.T) {
	// Test without prefix
	id1 := GenerateShortUUID("")
	if id1 == "" {
		t.Error("GenerateShortUUID() returned empty string")
	}
	
	// Should not contain dashes
	if strings.Contains(id1, "-") {
		t.Error("GenerateShortUUID() should not contain dashes")
	}
	
	// Should be 32 characters (UUID without dashes)
	if len(id1) != 32 {
		t.Errorf("GenerateShortUUID() should be 32 characters, got %d", len(id1))
	}

	// Test with prefix
	prefix := "short"
	id2 := GenerateShortUUID(prefix)
	if !strings.HasPrefix(id2, prefix+"_") {
		t.Errorf("GenerateShortUUID() with prefix %s should start with %s_, got %s", prefix, prefix, id2)
	}

	// Test uniqueness
	id3 := GenerateShortUUID("")
	if id1 == id3 {
		t.Error("GenerateShortUUID() should generate unique UUIDs")
	}
}

func TestGenerateOrderID(t *testing.T) {
	orderID := GenerateOrderID()
	
	if !strings.HasPrefix(orderID, "ord_") {
		t.Errorf("GenerateOrderID() should start with 'ord_', got %s", orderID)
	}
	
	// Test uniqueness
	orderID2 := GenerateOrderID()
	if orderID == orderID2 {
		t.Error("GenerateOrderID() should generate unique IDs")
	}
}

func TestGeneratePositionID(t *testing.T) {
	posID := GeneratePositionID()
	
	if !strings.HasPrefix(posID, "pos_") {
		t.Errorf("GeneratePositionID() should start with 'pos_', got %s", posID)
	}
	
	// Test uniqueness
	posID2 := GeneratePositionID()
	if posID == posID2 {
		t.Error("GeneratePositionID() should generate unique IDs")
	}
}

// Benchmark tests
func BenchmarkGenerateUUID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateUUID("test")
	}
}

func BenchmarkGenerateShortUUID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateShortUUID("test")
	}
}

func BenchmarkGenerateOrderID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateOrderID()
	}
}