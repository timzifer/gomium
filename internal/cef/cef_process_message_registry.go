package cef

import (
	"errors"
	"sync"
)

var (
	messageFactoryRegistry = map[string]ProcessMessageFactory{}
	messageFactoryMutex    = sync.RWMutex{}
)

type (
	ProcessMessageFactory func(message ProcessMessage) (any, error)
)

var (
	MessageIdentifierAlreadyRegisteredError = errors.New("message identifier already registered")
	MessageIdentifierNotRegisteredError     = errors.New("message identifier not registered")
)

func RegisterMessageIdentifier(identifier string, factoryMethod ProcessMessageFactory) error {
	messageFactoryMutex.Lock()
	defer messageFactoryMutex.Unlock()

	if _, exists := messageFactoryRegistry[identifier]; exists {
		return MessageIdentifierAlreadyRegisteredError
	}
	messageFactoryRegistry[identifier] = factoryMethod

	return nil
}

func getProcessMessageInstance(message ProcessMessage) (any, error) {
	name := message.Name()

	messageFactoryMutex.RLock()
	factory, exists := messageFactoryRegistry[name]
	messageFactoryMutex.RUnlock()

	if exists {
		return factory(message)
	}

	return nil, MessageIdentifierNotRegisteredError
}
