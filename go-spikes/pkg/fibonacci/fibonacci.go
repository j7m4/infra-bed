package fibonacci

import (
	"context"

	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/infra-bed/go-spikes/pkg/tracing"
)

// DoFibonacci calculates the nth Fibonacci number recursively
func DoFibonacci(n int) int {
	return DoFibonacciWithContext(context.Background(), n)
}

// DoFibonacciWithContext calculates the nth Fibonacci number with tracing context
func DoFibonacciWithContext(ctx context.Context, n int) int {
	ctx, span := tracing.StartSpanWithAttributes(
		ctx, 
		"fibonacci.calculate",
		tracing.FibonacciAttributes(n),
	)
	defer span.End()

	log := logger.Ctx(ctx)
	log.Debug().Int("n", n).Msg("Starting Fibonacci calculation")

	if n <= 1 {
		result := n
		tracing.SetSpanAttributes(span, tracing.FibonacciAttributes(result)...)
		log.Debug().Int("n", n).Int("result", result).Msg("Base case reached")
		return result
	}

	// For larger numbers, create child spans
	log.Debug().Int("n", n).Msg("Computing recursive Fibonacci")
	
	// Create child spans for recursive calls to avoid deep recursion overhead
	result := fibonacciRecursive(n)
	
	tracing.SetSpanAttributes(span, tracing.FibonacciAttributes(result)...)
	log.Info().Int("n", n).Int("result", result).Msg("Fibonacci calculation completed")
	
	return result
}

// fibonacciRecursive performs the actual recursive calculation without creating spans for every call
func fibonacciRecursive(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacciRecursive(n-1) + fibonacciRecursive(n-2)
}