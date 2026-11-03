// Package source defines explicit read-only provider integration.
package source

import (
 "context"
 "github.com/maldous/CircularQuay/internal/model"
)

// Reader resolves one explicitly declared binding.
type Reader interface { Read(context.Context, model.Source, model.Binding) (model.Value,error) }

// ReadFunc adapts a function to Reader.
type ReadFunc func(context.Context,model.Source,model.Binding)(model.Value,error)
func(f ReadFunc) Read(c context.Context,s model.Source,b model.Binding)(model.Value,error){return f(c,s,b)}
