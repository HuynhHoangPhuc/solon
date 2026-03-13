"""Sample Python file for testing ast-grep and LSP integration."""

from typing import Optional


class Calculator:
    """A simple calculator for testing."""

    def __init__(self, value: float = 0.0) -> None:
        self.value = value

    def add(self, n: float) -> "Calculator":
        self.value += n
        return self

    def multiply(self, n: float) -> "Calculator":
        self.value *= n
        return self

    def result(self) -> float:
        return self.value

    def __str__(self) -> str:
        return str(self.value)


def greet(name: str) -> str:
    return f"Hello, {name}!"


def divide(a: float, b: float) -> Optional[float]:
    if b == 0:
        return None
    return a / b


if __name__ == "__main__":
    calc = Calculator(0.0)
    calc.add(10.0).multiply(3.0)
    print(f"Result: {calc}")
    print(greet("world"))
