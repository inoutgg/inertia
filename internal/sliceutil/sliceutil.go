package sliceutil

// Map returns a new slice containing the results of applying f to each element of in.
func Map[I, O any](in []I, f func(I) O) []O {
	out := make([]O, 0, len(in))
	for _, t := range in {
		out = append(out, f(t))
	}

	return out
}

// Filter returns a new slice containing only the elements of in for which f returns true.
func Filter[I any](in []I, f func(I) bool) []I {
	out := make([]I, 0, len(in))
	for _, t := range in {
		if f(t) {
			out = append(out, t)
		}
	}

	return out
}

// Reduce returns the result of applying f to each element of in, accumulating the result.
func Reduce[I, O any](in []I, f func(I, O) O, initial O) O {
	out := initial
	for _, t := range in {
		out = f(t, out)
	}

	return out
}
