package oauth

// Registry holds registered OAuth2 providers, keyed by name.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds or replaces a provider in the registry using the provider's own Name().
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// RegisterAs adds or replaces a provider under an explicit name, overriding p.Name().
// Use this to register the same provider type with different configurations
// (e.g., "discord-activity" alongside "discord").
func (r *Registry) RegisterAs(name string, p Provider) {
	r.providers[name] = p
}

// Get returns the provider for the given name, or false if not registered.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}
