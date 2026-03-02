package scoreservice

// ptr returns a pointer to v. Used in tests to construct pointer fields inline.
func ptr[T any](v T) *T { return &v }
