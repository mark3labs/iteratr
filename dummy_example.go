package main

import "fmt"

// DummyGreeting returns a friendly greeting message with emojis
func DummyGreeting(name string) string {
	if name == "" {
		name = "World"
	}
	return fmt.Sprintf("Hello, %s! Welcome to iteratr!", name)
}

// DummyAdd adds two integers and returns the result
func DummyAdd(a, b int) int {
	return a + b
}

// DummyMultiply multiplies two integers and returns the result
func DummyMultiply(a, b int) int {
	return a * b
}
