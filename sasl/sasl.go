package sasl

// PlainAuthenticator is the interface implemented by types that can handle
// authenticating users. It should be able to also handle authenticating a user
// for a separate identity than their username.
type PlainAuthenticator interface {
	Authenticate(identity, username, password string) error
}

type FakePlain struct{}

func (fp FakePlain) Authenticate(_, _, _ string) error { return nil }
