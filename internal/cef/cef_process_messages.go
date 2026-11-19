package cef

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/url"
)

const (
	processMessageEvaluateJavaScriptRequest  = "pm.evaluateJavaScript.Request"
	processMessageEvaluateJavaScriptResponse = "pm.evaluateJavaScript.Response"

	processMessageRegisterFunctionRequest = "pm.registerFunctionRequest"

	// every gob payload travels as the binary argument at index 0
	processMessagePayloadArgumentIndex = 0
)

// decodeBinaryPayload validates the message payload and gob-decodes it into
// target. A hostile or malformed message must not be able to crash the
// process.
func decodeBinaryPayload(message ProcessMessage, index int, target any) error {
	args := message.argumentList()
	if args.ptr() == nil {
		return errors.New("process message has no argument list")
	}
	defer args.Release()

	if args.Size() <= index {
		return fmt.Errorf("process message has no argument at index %d", index)
	}
	if args.GetType(index) != ValueTypeBinary {
		return fmt.Errorf("process message argument %d is not binary but %v", index, args.GetType(index))
	}

	binary := args.Binary(index)
	if binary.ptr() == nil {
		return fmt.Errorf("process message argument %d has no binary value", index)
	}
	defer binary.Release()

	return gob.NewDecoder(binary.Data()).Decode(target)
}

func init() {
	_ = RegisterMessageIdentifier(processMessageEvaluateJavaScriptRequest, func(message ProcessMessage) (any, error) {
		val := &javaScriptEvaluationRequest{}
		if err := decodeBinaryPayload(message, processMessagePayloadArgumentIndex, val); err != nil {
			return nil, err
		}
		return val, nil
	})
	_ = RegisterMessageIdentifier(processMessageEvaluateJavaScriptResponse, func(message ProcessMessage) (any, error) {
		val := &javaScriptEvaluationResponse{}
		if err := decodeBinaryPayload(message, processMessagePayloadArgumentIndex, val); err != nil {
			return nil, err
		}
		return val, nil
	})
}

type (
	Dispatcher interface {
		sendProcessMessage(id CefProcessId, message ProcessMessage) bool
	}
	DispatchWith interface {
		DispatchWith(dispatcher Dispatcher, target CefProcessId) (bool, error)
	}
	javaScriptEvaluationRequest struct {
		Id      uuid.UUID
		Line    int64
		Code    string
		Url     *url.URL
		Frame   string
		Browser int32
	}

	javaScriptEvaluationResponse struct {
		Id        uuid.UUID
		Exception *ExceptionError
		Result    *Exported
	}
)

// dispatchGobPayload encodes payload, wraps it into a process message and
// sends it. The message reference is consumed by sendProcessMessage; on the
// error paths it is released here.
func dispatchGobPayload(name string, payload any, dispatcher Dispatcher, target CefProcessId) (bool, error) {
	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(payload); err != nil {
		return false, err
	}

	msg := newProcessMessage(name)

	args := msg.argumentList()
	if args.ptr() == nil {
		msg.release()
		return false, errors.New("could not obtain argument list")
	}

	ok := args.SetSize(1) &&
		// SetBinary consumes the binary-value reference
		args.SetBinary(processMessagePayloadArgumentIndex, BinaryValueFromBuffer(buf))
	args.Release()

	if !ok {
		msg.release()
		return false, errors.New("could not attach payload to process message")
	}

	return dispatcher.sendProcessMessage(target, msg), nil
}

func (e javaScriptEvaluationRequest) DispatchWith(dispatcher Dispatcher, target CefProcessId) (bool, error) {
	return dispatchGobPayload(processMessageEvaluateJavaScriptRequest, e, dispatcher, target)
}

func (e javaScriptEvaluationResponse) DispatchWith(dispatcher Dispatcher, target CefProcessId) (bool, error) {
	return dispatchGobPayload(processMessageEvaluateJavaScriptResponse, e, dispatcher, target)
}

var (
	_ DispatchWith = (*javaScriptEvaluationRequest)(nil)
	_ DispatchWith = (*javaScriptEvaluationResponse)(nil)
)
