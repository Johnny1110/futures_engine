package main

import (
	"testing"
)

func TestNewFuturesEngine(t *testing.T) {
	engine := NewFuturesEngine()

	if engine == nil {
		t.Fatal("NewFuturesEngine() returned nil")
	}

	if engine.name != "Futures Engine" {
		t.Errorf("Expected name 'Futures Engine', got '%s'", engine.name)
	}

	if engine.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", engine.version)
	}
}

func TestFuturesEngineStart(t *testing.T) {
	engine := NewFuturesEngine()

	err := engine.Start()
	if err != nil {
		t.Errorf("Start() returned an error: %v", err)
	}
}
