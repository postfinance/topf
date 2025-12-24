package config

import (
	"net/url"

	"go.yaml.in/yaml/v4"
)

// Endpoint is a wrapper around url.URL that marshals/unmarshals as a string
type Endpoint url.URL

// MarshalYAML implements yaml.Marshaler to transparently convert url.URL to a string
func (e Endpoint) MarshalYAML() (any, error) {
	return (*url.URL)(&e).String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler to transparently parse from a
// string to url.URL
func (e *Endpoint) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	parsed, err := url.Parse(s)
	if err != nil {
		return err
	}

	*e = Endpoint(*parsed)

	return nil
}

func (e Endpoint) String() string {
	return (*url.URL)(&e).String()
}
