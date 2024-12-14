// internal/watermill/marshaler.go
package watermillutil

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// You can keep the marshaler as a package level variable if you only need one
var Marshaler = cqrs.JSONMarshaler{}
