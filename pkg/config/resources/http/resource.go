package http

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
)

const TypeHTTP string = "http"

type HTTP struct {
	types.ResourceBase `hcl:",remain"`

	Method string `hcl:"method" json:"method"`
	URL    string `hcl:"url" json:"url"`

	Headers map[string]string `hcl:"headers,optional" json:"headers,omitempty"`
	Payload string            `hcl:"payload,optional" json:"payload,omitempty"`
	Timeout string            `hcl:"timeout,optional" json:"timeout,omitempty"`

	// Output parameters
	Status int    `hcl:"status,optional" json:"status"`
	Body   string `hcl:"body,optional" json:"body"`
}

func (t *HTTP) Process() error {
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(t.Meta.ID)
		if r != nil {
			state := r.(*HTTP)
			t.Status = state.Status
			t.Body = state.Body
		}
	}

	return nil
}
