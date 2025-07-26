import pytest
from main import greet, add


def test_greet():
    assert greet("World") == "Hello, World!"
    assert greet("Python") == "Hello, Python!"


def test_add():
    assert add(2, 3) == 5
    assert add(-1, 1) == 0
    assert add(0, 0) == 0