package testdata

import (
	"context"
	"errors"
)

func Echo(ctx context.Context, in *Input) (*Output, error) {
	return &Output{
		S: in.S,
		D: in.D,
		I: in.I,
		B: in.B,
	}, nil
}

func Err(ctx context.Context, in *Input) (*Output, error) {
	return nil, errors.New("err")
}
