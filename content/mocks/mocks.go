// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ONSdigital/dp-download-service/content (interfaces: Writer,S3Client,S3ReadCloser)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	io "io"
	reflect "reflect"

	healthcheck "github.com/ONSdigital/dp-healthcheck/healthcheck"
	gomock "github.com/golang/mock/gomock"
)

// MockWriter is a mock of Writer interface.
type MockWriter struct {
	ctrl     *gomock.Controller
	recorder *MockWriterMockRecorder
}

// MockWriterMockRecorder is the mock recorder for MockWriter.
type MockWriterMockRecorder struct {
	mock *MockWriter
}

// NewMockWriter creates a new mock instance.
func NewMockWriter(ctrl *gomock.Controller) *MockWriter {
	mock := &MockWriter{ctrl: ctrl}
	mock.recorder = &MockWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWriter) EXPECT() *MockWriterMockRecorder {
	return m.recorder
}

// Write mocks base method.
func (m *MockWriter) Write(arg0 []byte) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Write indicates an expected call of Write.
func (mr *MockWriterMockRecorder) Write(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockWriter)(nil).Write), arg0)
}

// MockS3Client is a mock of S3Client interface.
type MockS3Client struct {
	ctrl     *gomock.Controller
	recorder *MockS3ClientMockRecorder
}

// MockS3ClientMockRecorder is the mock recorder for MockS3Client.
type MockS3ClientMockRecorder struct {
	mock *MockS3Client
}

// NewMockS3Client creates a new mock instance.
func NewMockS3Client(ctrl *gomock.Controller) *MockS3Client {
	mock := &MockS3Client{ctrl: ctrl}
	mock.recorder = &MockS3ClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockS3Client) EXPECT() *MockS3ClientMockRecorder {
	return m.recorder
}

// Checker mocks base method.
func (m *MockS3Client) Checker(arg0 context.Context, arg1 *healthcheck.CheckState) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Checker", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Checker indicates an expected call of Checker.
func (mr *MockS3ClientMockRecorder) Checker(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Checker", reflect.TypeOf((*MockS3Client)(nil).Checker), arg0, arg1)
}

// Get mocks base method.
func (m *MockS3Client) Get(arg0 string) (io.ReadCloser, *int64, error) {
	// Must be this one
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(*int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockS3ClientMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockS3Client)(nil).Get), arg0)
}

// MockS3ReadCloser is a mock of S3ReadCloser interface.
type MockS3ReadCloser struct {
	ctrl     *gomock.Controller
	recorder *MockS3ReadCloserMockRecorder
}

// MockS3ReadCloserMockRecorder is the mock recorder for MockS3ReadCloser.
type MockS3ReadCloserMockRecorder struct {
	mock *MockS3ReadCloser
}

// NewMockS3ReadCloser creates a new mock instance.
func NewMockS3ReadCloser(ctrl *gomock.Controller) *MockS3ReadCloser {
	mock := &MockS3ReadCloser{ctrl: ctrl}
	mock.recorder = &MockS3ReadCloserMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockS3ReadCloser) EXPECT() *MockS3ReadCloserMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockS3ReadCloser) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockS3ReadCloserMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockS3ReadCloser)(nil).Close))
}

// Read mocks base method.
func (m *MockS3ReadCloser) Read(arg0 []byte) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", arg0)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockS3ReadCloserMockRecorder) Read(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockS3ReadCloser)(nil).Read), arg0)
}
