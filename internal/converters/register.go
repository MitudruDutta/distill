package converters

import "github.com/MitudruDutta/distill/internal/convert"

// Default returns a Registry wired with the built-in converters. Generic
// converters get a higher priority value (tried later); specific ones get 0.
func Default() *convert.Registry {
	reg := &convert.Registry{}
	reg.Register(PlainText{}, 10) // generic catch-all
	reg.Register(CSV{}, 0)        // specific
	return reg
}
