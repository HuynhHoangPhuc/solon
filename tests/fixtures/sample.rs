use std::fmt;

/// A simple calculator for testing
pub struct Calculator {
    value: f64,
}

impl Calculator {
    pub fn new(value: f64) -> Self {
        Calculator { value }
    }

    pub fn add(&mut self, n: f64) -> &mut Self {
        self.value += n;
        self
    }

    pub fn multiply(&mut self, n: f64) -> &mut Self {
        self.value *= n;
        self
    }

    pub fn result(&self) -> f64 {
        self.value
    }
}

impl fmt::Display for Calculator {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.value)
    }
}

fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}

fn main() {
    let mut calc = Calculator::new(0.0);
    calc.add(10.0).multiply(3.0);
    println!("Result: {calc}");
    println!("{}", greet("world"));
}
