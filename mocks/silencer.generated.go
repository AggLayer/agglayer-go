// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	tx "github.com/0xPolygon/agglayer/tx"
)

// SilencerMock is an autogenerated mock type for the ISilencer type
type SilencerMock struct {
	mock.Mock
}

// Silence provides a mock function with given fields: ctx, signedTx
func (_m *SilencerMock) Silence(ctx context.Context, signedTx tx.SignedTx) error {
	ret := _m.Called(ctx, signedTx)

	if len(ret) == 0 {
		panic("no return value specified for Silence")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, tx.SignedTx) error); ok {
		r0 = rf(ctx, signedTx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewSilencerMock creates a new instance of SilencerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSilencerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *SilencerMock {
	mock := &SilencerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
