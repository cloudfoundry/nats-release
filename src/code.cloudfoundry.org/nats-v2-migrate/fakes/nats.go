// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	nats_interface "code.cloudfoundry.org/nats-v2-migrate/nats-interface"
)

type NatsConn struct {
	ConnectedServerVersionStub        func() string
	connectedServerVersionMutex       sync.RWMutex
	connectedServerVersionArgsForCall []struct {
	}
	connectedServerVersionReturns struct {
		result1 string
	}
	connectedServerVersionReturnsOnCall map[int]struct {
		result1 string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *NatsConn) ConnectedServerVersion() string {
	fake.connectedServerVersionMutex.Lock()
	ret, specificReturn := fake.connectedServerVersionReturnsOnCall[len(fake.connectedServerVersionArgsForCall)]
	fake.connectedServerVersionArgsForCall = append(fake.connectedServerVersionArgsForCall, struct {
	}{})
	stub := fake.ConnectedServerVersionStub
	fakeReturns := fake.connectedServerVersionReturns
	fake.recordInvocation("ConnectedServerVersion", []interface{}{})
	fake.connectedServerVersionMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *NatsConn) ConnectedServerVersionCallCount() int {
	fake.connectedServerVersionMutex.RLock()
	defer fake.connectedServerVersionMutex.RUnlock()
	return len(fake.connectedServerVersionArgsForCall)
}

func (fake *NatsConn) ConnectedServerVersionCalls(stub func() string) {
	fake.connectedServerVersionMutex.Lock()
	defer fake.connectedServerVersionMutex.Unlock()
	fake.ConnectedServerVersionStub = stub
}

func (fake *NatsConn) ConnectedServerVersionReturns(result1 string) {
	fake.connectedServerVersionMutex.Lock()
	defer fake.connectedServerVersionMutex.Unlock()
	fake.ConnectedServerVersionStub = nil
	fake.connectedServerVersionReturns = struct {
		result1 string
	}{result1}
}

func (fake *NatsConn) ConnectedServerVersionReturnsOnCall(i int, result1 string) {
	fake.connectedServerVersionMutex.Lock()
	defer fake.connectedServerVersionMutex.Unlock()
	fake.ConnectedServerVersionStub = nil
	if fake.connectedServerVersionReturnsOnCall == nil {
		fake.connectedServerVersionReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.connectedServerVersionReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *NatsConn) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.connectedServerVersionMutex.RLock()
	defer fake.connectedServerVersionMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *NatsConn) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ nats_interface.NatsConn = new(NatsConn)