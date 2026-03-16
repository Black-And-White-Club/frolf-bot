package oauth

import (
	"context"
	"testing"
)

// fakeProvider implements Provider for testing purposes.
type fakeProvider struct {
	name string
}

func (f *fakeProvider) Name() string                                            { return f.name }
func (f *fakeProvider) AuthCodeURL(state string) string                         { return "" }
func (f *fakeProvider) Exchange(_ context.Context, _ string) (*UserInfo, error) { return nil, nil }

// Verify fakeProvider satisfies the Provider interface at compile time.
var _ Provider = (*fakeProvider)(nil)

func TestRegistry_Register_StoresByProviderName(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{name: "discord"}
	r.Register(p)

	got, ok := r.Get("discord")
	if !ok {
		t.Fatal("expected provider registered under name 'discord'")
	}
	if got != p {
		t.Error("Get returned a different provider instance")
	}
}

func TestRegistry_RegisterAs_StoresUnderExplicitName(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{name: "discord"}
	r.RegisterAs("discord-activity", p)

	got, ok := r.Get("discord-activity")
	if !ok {
		t.Fatal("expected provider registered under alias 'discord-activity'")
	}
	if got != p {
		t.Error("Get returned a different provider instance")
	}
}

func TestRegistry_RegisterAs_DoesNotClobberOriginalName(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{name: "discord"}
	r.Register(p)
	r.RegisterAs("discord-activity", p)

	// Alias lookup works.
	if _, ok := r.Get("discord-activity"); !ok {
		t.Error("alias 'discord-activity' not found")
	}

	// Original name lookup also works.
	if _, ok := r.Get("discord"); !ok {
		t.Error("original name 'discord' was clobbered by RegisterAs")
	}
}

func TestRegistry_Get_ReturnsFalseForUnknownName(t *testing.T) {
	r := NewRegistry()
	if _, ok := r.Get("nonexistent"); ok {
		t.Error("expected ok=false for unknown provider name")
	}
}
