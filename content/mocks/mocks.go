// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ONSdigital/dp-download-service/content (interfaces: VaultClient)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockVaultClient is a mock of VaultClient interface
type MockVaultClient struct {
	ctrl     *gomock.Controller
	recorder *MockVaultClientMockRecorder
}

// MockVaultClientMockRecorder is the mock recorder for MockVaultClient
type MockVaultClientMockRecorder struct {
	mock *MockVaultClient
}

// NewMockVaultClient creates a new mock instance
func NewMockVaultClient(ctrl *gomock.Controller) *MockVaultClient {
	mock := &MockVaultClient{ctrl: ctrl}
	mock.recorder = &MockVaultClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockVaultClient) EXPECT() *MockVaultClientMockRecorder {
	return m.recorder
}

// ReadKey mocks base method
func (m *MockVaultClient) ReadKey(arg0, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadKey", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadKey indicates an expected call of ReadKey
func (mr *MockVaultClientMockRecorder) ReadKey(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadKey", reflect.TypeOf((*MockVaultClient)(nil).ReadKey), arg0, arg1)
}
