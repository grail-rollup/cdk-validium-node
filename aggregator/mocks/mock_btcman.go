package mocks

import (
	mock "github.com/stretchr/testify/mock"
)

type Btcman struct {
	mock.Mock
}

func (_m *Btcman) Inscribe(data []byte) (string, error) {
	return "", nil
}

func (_m *Btcman) DecodeInscription(txHash string) error {
	return nil
}

func (_m *Btcman) Shutdown() {

}

func NewBtcman(t interface {
	mock.TestingT
	Cleanup(func())
}) *Btcman {
	mock := &Btcman{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
