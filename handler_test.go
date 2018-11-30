package lambdapb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/newlix/lambdapb"
	"github.com/newlix/lambdapb/testdata"
	"github.com/stretchr/testify/assert"
)

func TestInvalidHandlers(t *testing.T) {

	testCases := []struct {
		name     string
		handler  interface{}
		expected error
	}{
		{
			name:     "nil handler",
			expected: errors.New("handler is nil"),
			handler:  nil,
		},
		{
			name:     "handler is not a function",
			expected: errors.New("handler is not function"),
			handler:  struct{}{},
		},
		{
			name:     "handler declares too many arguments",
			expected: errors.New("handler takes two arguments, but got 3"),
			handler: func(n context.Context, x string, y string) error {
				return nil
			},
		},
		{
			name:     "handler first argument is not Context",
			expected: errors.New("handler first argument should implement Context"),
			handler: func(a string, x context.Context) error {
				return nil
			},
		},
		{
			name:     "handler second argument is not proto.Message",
			expected: errors.New("handler second argument should implement proto.Message"),
			handler: func(ctx context.Context, in string) error {
				return nil
			},
		},
		{
			name:     "handler returns too many values",
			expected: errors.New("handler returns two values, but got 3"),
			handler: func(ctx context.Context, in *testdata.Input) (error, error, error) {
				return nil, nil, nil
			},
		},
		{
			name:     "handler first return value should implement proto.Message",
			expected: errors.New("handler first return value should implement proto.Message"),
			handler: func(ctx context.Context, in *testdata.Input) (error, error) {
				return nil, nil
			},
		},
		{
			name:     "handler returning a single value does not implement error",
			expected: errors.New("handler second return value should implement error"),
			handler: func(ctx context.Context, in *testdata.Input) (*testdata.Output, string) {
				return nil, ""
			},
		},
	}
	for _, testCase := range testCases {
		lambdaHandler := NewHandler(testCase.handler)
		_, err := lambdaHandler.Invoke(context.TODO(), make([]byte, 0))
		assert.Equal(t, testCase.expected, err)
	}
}

func TestNewHandlerInvoke(t *testing.T) {
	in := &testdata.Input{
		S: "s",
		D: 3.14,
		I: 1,
		B: true,
	}
	h := lambdapb.NewHandler(testdata.Echo)
	bIn, err := proto.Marshal(in)
	assert.NoError(t, err)
	base64In := base64.StdEncoding.EncodeToString(bIn)
	jsonOut, err := h.Invoke(context.Background(), []byte("\""+base64In+"\""))
	var base64Out string
	if err := json.Unmarshal(jsonOut, &base64Out); err != nil {
		t.Error(err)
	}
	bOut, err := base64.StdEncoding.DecodeString(base64Out)
	assert.NoError(t, err)
	var out testdata.Output
	if err := proto.Unmarshal(bOut, &out); err != nil {
		t.Error(err)
	}
	assert.Equal(t, in.S, out.S)
	assert.InDelta(t, in.D, out.D, 0.001)
	assert.Equal(t, in.I, out.I)
	assert.Equal(t, in.B, out.B)
}

func TestNewHandlerInvokeErr(t *testing.T) {
	in := &testdata.Input{
		S: "s",
		D: 3.14,
		I: 1,
		B: true,
	}
	h := lambdapb.NewHandler(testdata.Err)
	bIn, err := proto.Marshal(in)
	assert.NoError(t, err)
	base64In := base64.StdEncoding.EncodeToString(bIn)
	jsonOut, err := h.Invoke(context.Background(), []byte("\""+base64In+"\""))
	assert.Nil(t, jsonOut)
	assert.Error(t, err, "err")
}
