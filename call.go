package shimmy

// Call is the runtime envelope exposed to interceptor middleware.
//
// v0 note: generated call structs are mutable and are not safe for concurrent
// access while invoke is running.
type Call interface {
	Method() string
	Args() []any
	Results() []any
}
