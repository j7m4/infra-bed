package fibonacci

// DoFibonacci calculates the nth Fibonacci number recursively
func DoFibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return DoFibonacci(n-1) + DoFibonacci(n-2)
}