package main

import (
	Operations "Day1/math"
	"fmt"
)

func main() {
	var a, b int

	fmt.Print("Enter first number: ")
	fmt.Scanln(&a)

	fmt.Print("Enter second number: ")
	fmt.Scanln(&b)

	sum := Operations.Add(a, b)
	product := Operations.Multiply(a, b)

	// Print results
	fmt.Printf("Sum: %d\n", sum)
	fmt.Printf("Product: %d\n", product)
}
