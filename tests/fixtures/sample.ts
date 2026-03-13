/** Sample TypeScript file for testing ast-grep and LSP integration. */

interface CalculatorOptions {
  initialValue?: number;
}

class Calculator {
  private value: number;

  constructor(options: CalculatorOptions = {}) {
    this.value = options.initialValue ?? 0;
  }

  add(n: number): this {
    this.value += n;
    return this;
  }

  multiply(n: number): this {
    this.value *= n;
    return this;
  }

  result(): number {
    return this.value;
  }

  toString(): string {
    return String(this.value);
  }
}

function greet(name: string): string {
  return `Hello, ${name}!`;
}

function divide(a: number, b: number): number | null {
  if (b === 0) return null;
  return a / b;
}

const calc = new Calculator({ initialValue: 0 });
calc.add(10).multiply(3);
console.log(`Result: ${calc}`);
console.log(greet("world"));
