#!/usr/bin/env python3
"""Sample Python file for testing."""

def greet(name: str) -> str:
    """Return a greeting message."""
    return f"Hello, {name}!"

def add(a: int, b: int) -> int:
    """Add two numbers."""
    return a + b

if __name__ == "__main__":
    print(greet("World"))