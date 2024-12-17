package watermillutil

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// Marshaler is a JSON marshaler for commands and events.
var Marshaler = cqrs.JSONMarshaler{}
