package lambdapb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
)

// Handler is the generic function type
type Handler func(context.Context, []byte) ([]byte, error)

// Invoke calls the handler, and serializes the response.
// If the underlying handler returned an error, or an error occurs during serialization, error is returned.
func (h Handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	return h(ctx, payload)
}

func errorHandler(e error) Handler {
	return func(ctx context.Context, event []byte) ([]byte, error) {
		return nil, e
	}
}

func validateArguments(handler reflect.Type) error {
	if handler.NumIn() != 2 {
		return fmt.Errorf("handler takes two arguments, but got %d", handler.NumIn())
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !handler.In(0).Implements(contextType) {
		return fmt.Errorf("handler first argument should implement Context")
	}
	messageType := reflect.TypeOf((*proto.Message)(nil)).Elem()
	if !handler.In(1).Implements(messageType) {
		return fmt.Errorf("handler second argument should implement proto.Message")
	}
	return nil
}

func validateReturns(handler reflect.Type) error {
	if handler.NumOut() != 2 {
		return fmt.Errorf("handler returns two values, but got %d", handler.NumOut())
	}
	messageType := reflect.TypeOf((*proto.Message)(nil)).Elem()
	if !handler.Out(0).Implements(messageType) {
		return fmt.Errorf("handler first return value should implement proto.Message")
	}
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !handler.Out(1).Implements(errorType) {
		return fmt.Errorf("handler second return value should implement error")
	}
	return nil
}

// NewHandler creates a base lambda handler from the given handler function. The
// returned Handler performs JSON deserialization and deserialization, and
// delegates to the input handler function.  The handler function parameter must
// satisfy the rules documented by Start.  If handlerFunc is not a valid
// handler, the returned Handler simply reports the validation error.
func NewHandler(handler interface{}) Handler {
	if handler == nil {
		return errorHandler(errors.New("handler is nil"))
	}
	h := reflect.ValueOf(handler)
	t := reflect.TypeOf(handler)
	if t.Kind() != reflect.Func {
		return errorHandler(errors.New("handler is not function"))
	}
	if err := validateArguments(t); err != nil {
		return errorHandler(err)
	}

	if err := validateReturns(t); err != nil {
		return errorHandler(err)
	}

	return func(ctx context.Context, payload []byte) ([]byte, error) {
		// construct arguments
		msgType := t.In(1).Elem()
		msg := reflect.New(msgType)
		var s string
		if err := json.Unmarshal(payload, &s); err != nil {
			return nil, err
		}
		in, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		}
		if err := proto.Unmarshal(in, msg.Interface().(proto.Message)); err != nil {
			return nil, err
		}
		args := []reflect.Value{reflect.ValueOf(ctx), msg}
		response := h.Call(args)
		if err, ok := response[1].Interface().(error); ok {
			return nil, err
		}
		if out, ok := response[0].Interface().(proto.Message); ok {
			b, err := proto.Marshal(out)
			if err != nil {
				return nil, err
			}
			s := base64.StdEncoding.EncodeToString(b)
			return []byte("\"" + s + "\""), nil
		}
		return nil, errors.New("missing output") //todo
	}
}
