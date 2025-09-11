package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStringPtr(t *testing.T) {
	str := "test"
	ptr := StringPtr(str)
	if *ptr != str {
		t.Errorf("StringPtr() = %v, want %v", *ptr, str)
	}
}

func TestIntPtr(t *testing.T) {
	num := 42
	ptr := IntPtr(num)
	if *ptr != num {
		t.Errorf("IntPtr() = %v, want %v", *ptr, num)
	}
}

func TestBoolPtr(t *testing.T) {
	val := true
	ptr := BoolPtr(val)
	if *ptr != val {
		t.Errorf("BoolPtr() = %v, want %v", *ptr, val)
	}
}

func TestFileExists(t *testing.T) {
	// Test with existing file
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if !FileExists(tempFile) {
		t.Errorf("FileExists() = false, want true for existing file")
	}

	// Test with non-existing file
	nonExistentFile := filepath.Join(t.TempDir(), "nonexistent.txt")
	if FileExists(nonExistentFile) {
		t.Errorf("FileExists() = true, want false for non-existing file")
	}
}

func TestDirExists(t *testing.T) {
	// Test with existing directory
	tempDir := t.TempDir()
	if !DirExists(tempDir) {
		t.Errorf("DirExists() = false, want true for existing directory")
	}

	// Test with non-existing directory
	nonExistentDir := filepath.Join(tempDir, "nonexistent")
	if DirExists(nonExistentDir) {
		t.Errorf("DirExists() = true, want false for non-existing directory")
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "newdir")

	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir() error = %v", err)
	}

	if !DirExists(newDir) {
		t.Errorf("EnsureDir() did not create directory")
	}

	// Test with existing directory
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir() error = %v for existing directory", err)
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !Contains(slice, "b") {
		t.Errorf("Contains() = false, want true for existing item")
	}

	if Contains(slice, "d") {
		t.Errorf("Contains() = true, want false for non-existing item")
	}
}

func TestMap(t *testing.T) {
	input := []int{1, 2, 3}
	expected := []string{"1", "2", "3"}

	result := Map(input, func(i int) string {
		return string(rune(i + '0'))
	})

	if len(result) != len(expected) {
		t.Errorf("Map() returned slice of length %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Map() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestFilter(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	expected := []int{2, 4}

	result := Filter(input, func(i int) bool {
		return i%2 == 0
	})

	if len(result) != len(expected) {
		t.Errorf("Filter() returned slice of length %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Filter() result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}