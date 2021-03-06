// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ONSdigital/dp-download-service/downloads (interfaces: FilterClient,DatasetClient,ImageClient)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	dataset "github.com/ONSdigital/dp-api-clients-go/dataset"
	filter "github.com/ONSdigital/dp-api-clients-go/filter"
	image "github.com/ONSdigital/dp-api-clients-go/image"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockFilterClient is a mock of FilterClient interface
type MockFilterClient struct {
	ctrl     *gomock.Controller
	recorder *MockFilterClientMockRecorder
}

// MockFilterClientMockRecorder is the mock recorder for MockFilterClient
type MockFilterClientMockRecorder struct {
	mock *MockFilterClient
}

// NewMockFilterClient creates a new mock instance
func NewMockFilterClient(ctrl *gomock.Controller) *MockFilterClient {
	mock := &MockFilterClient{ctrl: ctrl}
	mock.recorder = &MockFilterClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFilterClient) EXPECT() *MockFilterClientMockRecorder {
	return m.recorder
}

// GetOutput mocks base method
func (m *MockFilterClient) GetOutput(arg0 context.Context, arg1, arg2, arg3, arg4, arg5 string) (filter.Model, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOutput", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(filter.Model)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOutput indicates an expected call of GetOutput
func (mr *MockFilterClientMockRecorder) GetOutput(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOutput", reflect.TypeOf((*MockFilterClient)(nil).GetOutput), arg0, arg1, arg2, arg3, arg4, arg5)
}

// MockDatasetClient is a mock of DatasetClient interface
type MockDatasetClient struct {
	ctrl     *gomock.Controller
	recorder *MockDatasetClientMockRecorder
}

// MockDatasetClientMockRecorder is the mock recorder for MockDatasetClient
type MockDatasetClientMockRecorder struct {
	mock *MockDatasetClient
}

// NewMockDatasetClient creates a new mock instance
func NewMockDatasetClient(ctrl *gomock.Controller) *MockDatasetClient {
	mock := &MockDatasetClient{ctrl: ctrl}
	mock.recorder = &MockDatasetClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDatasetClient) EXPECT() *MockDatasetClientMockRecorder {
	return m.recorder
}

// GetVersion mocks base method
func (m *MockDatasetClient) GetVersion(arg0 context.Context, arg1, arg2, arg3, arg4, arg5, arg6, arg7 string) (dataset.Version, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersion", arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
	ret0, _ := ret[0].(dataset.Version)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVersion indicates an expected call of GetVersion
func (mr *MockDatasetClientMockRecorder) GetVersion(arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersion", reflect.TypeOf((*MockDatasetClient)(nil).GetVersion), arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
}

// MockImageClient is a mock of ImageClient interface
type MockImageClient struct {
	ctrl     *gomock.Controller
	recorder *MockImageClientMockRecorder
}

// MockImageClientMockRecorder is the mock recorder for MockImageClient
type MockImageClientMockRecorder struct {
	mock *MockImageClient
}

// NewMockImageClient creates a new mock instance
func NewMockImageClient(ctrl *gomock.Controller) *MockImageClient {
	mock := &MockImageClient{ctrl: ctrl}
	mock.recorder = &MockImageClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockImageClient) EXPECT() *MockImageClientMockRecorder {
	return m.recorder
}

// GetDownloadVariant mocks base method
func (m *MockImageClient) GetDownloadVariant(arg0 context.Context, arg1, arg2, arg3, arg4, arg5 string) (image.ImageDownload, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDownloadVariant", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(image.ImageDownload)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDownloadVariant indicates an expected call of GetDownloadVariant
func (mr *MockImageClientMockRecorder) GetDownloadVariant(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDownloadVariant", reflect.TypeOf((*MockImageClient)(nil).GetDownloadVariant), arg0, arg1, arg2, arg3, arg4, arg5)
}
