package target

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type HttpOptions struct {
	URL     string
	Method  string
	Headers map[string]string
}

type HttpMonitor struct {
	Opts *HttpOptions
}

func parseHttpMeta(meta map[string]string) (opts *HttpOptions, err error) {
	opts = &HttpOptions{}

	if headerStr, ok := meta["headers"]; ok {
		err = mapstructure.Decode(headerStr, opts.Headers)
	}
	if err != nil {
		return nil, err
	}
	delete(meta, "headers")
	err = mapstructure.Decode(meta, opts)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func NewHttpTarget(meta map[string]string) (*HttpMonitor, error) {
	opts, err := parseHttpMeta(meta)
	if err != nil {
		return nil, errors.Wrap(err, "parse http meta")
	}

	m := &HttpMonitor{
		Opts: opts,
	}

	return m, nil
}

func (h *HttpMonitor) Exec(ctx context.Context) error {
	panic("implement me")
}
