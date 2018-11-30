package lambdapb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
)

// handler is the generic function type
type handler func(context.Context, []byte) ([]byte, error)

// Invoke calls the handler, and serializes the response.
// If the underlying handler returned an error, or an error occurs during serialization, error is returned.
func (h handler) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	return h(ctx, payload)
}

func errorHandler(e error) handler {
	return func(ctx context.Context, event []byte) ([]byte, error) {
		return nil, e
	}
}

func validateArguments(handler reflect.Type) error {
	if handler.NumIn() != 2 {
		return fmt.Errorf("handler takes two arguments, but handler takes %d", handler.NumIn())
	}
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	argumentType := handler.In(0)
	if !argumentType.Implements(contextType) {
		return fmt.Errorf("the first argument of handler is not Context. got %s", argumentType.Kind())
	}
	return nil
}

func validateReturns(handler reflect.Type) error {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if handler.NumOut() != 2 {
		return fmt.Errorf("handler must return two values")
	}
	if !handler.Out(1).Implements(errorType) {
		return fmt.Errorf("handler returns two values, but the second does not implement error")
	}
	return nil
}

// NewHandler creates a base lambda handler from the given handler function. The
// returned Handler performs JSON deserialization and deserialization, and
// delegates to the input handler function.  The handler function parameter must
// satisfy the rules documented by Start.  If handlerFunc is not a valid
// handler, the returned Handler simply reports the validation error.
func NewHandler(f interface{}) handler {
	if f == nil {
		return errorHandler(fmt.Errorf("f is nil"))
	}
	h := reflect.ValueOf(f)
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		return errorHandler(fmt.Errorf("f is %s not %s", t.Kind(), reflect.Func))
	}

	if err := validateArguments(t); err != nil {
		return errorHandler(err)
	}

	if err := validateReturns(t); err != nil {
		return errorHandler(err)
	}

	return handler(func(ctx context.Context, payload []byte) ([]byte, error) {
		// construct arguments
		var args []reflect.Value
		args = append(args, reflect.ValueOf(ctx))

		eventType := t.In(0)
		event := reflect.New(eventType)
		var s string
		if err := json.Unmarshal(payload, &s); err != nil {
			return nil, err
		}
		in, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		}
		proto.Unmarshal(in, event.Interface().(proto.Message))
		args = append(args, event.Elem())
		response := h.Call(args)
		if err := response[1].Interface().(error); err != nil {
			return nil, err
		}
		out, err := proto.Marshal(response[0].Interface().(proto.Message))
		if err != nil {
			return nil, err
		}
		ss := base64.StdEncoding.EncodeToString(out)
		return []byte("\"" + ss + "\""), nil
	})
}
