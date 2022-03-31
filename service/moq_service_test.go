// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package service_test

import (
	"context"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"net/http"
	"sync"
)

// Ensure, that DependenciesMock does implement service.Dependencies.
// If this is not the case, regenerate this file with moq.
var _ service.Dependencies = &DependenciesMock{}

// DependenciesMock is a mock implementation of service.Dependencies.
//
// 	func TestSomethingThatUsesDependencies(t *testing.T) {
//
// 		// make and configure a mocked service.Dependencies
// 		mockedDependencies := &DependenciesMock{
// 			DatasetClientFunc: func(s string) downloads.DatasetClient {
// 				panic("mock out the DatasetClient method")
// 			},
// 			FilterClientFunc: func(s string) downloads.FilterClient {
// 				panic("mock out the FilterClient method")
// 			},
// 			HealthCheckFunc: func(configMoqParam *config.Config, s1 string, s2 string, s3 string) (service.HealthChecker, error) {
// 				panic("mock out the HealthCheck method")
// 			},
// 			HttpServerFunc: func(configMoqParam *config.Config, handler http.Handler) service.HTTPServer {
// 				panic("mock out the HttpServer method")
// 			},
// 			ImageClientFunc: func(s string) downloads.ImageClient {
// 				panic("mock out the ImageClient method")
// 			},
// 			S3ClientFunc: func(configMoqParam *config.Config) (content.S3Client, error) {
// 				panic("mock out the S3Client method")
// 			},
// 			VaultClientFunc: func(configMoqParam *config.Config) (content.VaultClient, error) {
// 				panic("mock out the VaultClient method")
// 			},
// 		}
//
// 		// use mockedDependencies in code that requires service.Dependencies
// 		// and then make assertions.
//
// 	}
type DependenciesMock struct {
	// DatasetClientFunc mocks the DatasetClient method.
	DatasetClientFunc func(s string) downloads.DatasetClient

	// FilterClientFunc mocks the FilterClient method.
	FilterClientFunc func(s string) downloads.FilterClient

	// HealthCheckFunc mocks the HealthCheck method.
	HealthCheckFunc func(configMoqParam *config.Config, s1 string, s2 string, s3 string) (service.HealthChecker, error)

	// HttpServerFunc mocks the HttpServer method.
	HttpServerFunc func(configMoqParam *config.Config, handler http.Handler) service.HTTPServer

	// ImageClientFunc mocks the ImageClient method.
	ImageClientFunc func(s string) downloads.ImageClient

	// S3ClientFunc mocks the S3Client method.
	S3ClientFunc func(configMoqParam *config.Config) (content.S3Client, error)

	// VaultClientFunc mocks the VaultClient method.
	VaultClientFunc func(configMoqParam *config.Config) (content.VaultClient, error)

	// calls tracks calls to the methods.
	calls struct {
		// DatasetClient holds details about calls to the DatasetClient method.
		DatasetClient []struct {
			// S is the s argument value.
			S string
		}
		// FilterClient holds details about calls to the FilterClient method.
		FilterClient []struct {
			// S is the s argument value.
			S string
		}
		// HealthCheck holds details about calls to the HealthCheck method.
		HealthCheck []struct {
			// ConfigMoqParam is the configMoqParam argument value.
			ConfigMoqParam *config.Config
			// S1 is the s1 argument value.
			S1 string
			// S2 is the s2 argument value.
			S2 string
			// S3 is the s3 argument value.
			S3 string
		}
		// HttpServer holds details about calls to the HttpServer method.
		HttpServer []struct {
			// ConfigMoqParam is the configMoqParam argument value.
			ConfigMoqParam *config.Config
			// Handler is the handler argument value.
			Handler http.Handler
		}
		// ImageClient holds details about calls to the ImageClient method.
		ImageClient []struct {
			// S is the s argument value.
			S string
		}
		// S3Client holds details about calls to the S3Client method.
		S3Client []struct {
			// ConfigMoqParam is the configMoqParam argument value.
			ConfigMoqParam *config.Config
		}
		// VaultClient holds details about calls to the VaultClient method.
		VaultClient []struct {
			// ConfigMoqParam is the configMoqParam argument value.
			ConfigMoqParam *config.Config
		}
	}
	lockDatasetClient sync.RWMutex
	lockFilterClient  sync.RWMutex
	lockHealthCheck   sync.RWMutex
	lockHttpServer    sync.RWMutex
	lockImageClient   sync.RWMutex
	lockS3Client      sync.RWMutex
	lockVaultClient   sync.RWMutex
}

// DatasetClient calls DatasetClientFunc.
func (mock *DependenciesMock) DatasetClient(s string) downloads.DatasetClient {
	if mock.DatasetClientFunc == nil {
		panic("DependenciesMock.DatasetClientFunc: method is nil but Dependencies.DatasetClient was just called")
	}
	callInfo := struct {
		S string
	}{
		S: s,
	}
	mock.lockDatasetClient.Lock()
	mock.calls.DatasetClient = append(mock.calls.DatasetClient, callInfo)
	mock.lockDatasetClient.Unlock()
	return mock.DatasetClientFunc(s)
}

// DatasetClientCalls gets all the calls that were made to DatasetClient.
// Check the length with:
//     len(mockedDependencies.DatasetClientCalls())
func (mock *DependenciesMock) DatasetClientCalls() []struct {
	S string
} {
	var calls []struct {
		S string
	}
	mock.lockDatasetClient.RLock()
	calls = mock.calls.DatasetClient
	mock.lockDatasetClient.RUnlock()
	return calls
}

// FilterClient calls FilterClientFunc.
func (mock *DependenciesMock) FilterClient(s string) downloads.FilterClient {
	if mock.FilterClientFunc == nil {
		panic("DependenciesMock.FilterClientFunc: method is nil but Dependencies.FilterClient was just called")
	}
	callInfo := struct {
		S string
	}{
		S: s,
	}
	mock.lockFilterClient.Lock()
	mock.calls.FilterClient = append(mock.calls.FilterClient, callInfo)
	mock.lockFilterClient.Unlock()
	return mock.FilterClientFunc(s)
}

// FilterClientCalls gets all the calls that were made to FilterClient.
// Check the length with:
//     len(mockedDependencies.FilterClientCalls())
func (mock *DependenciesMock) FilterClientCalls() []struct {
	S string
} {
	var calls []struct {
		S string
	}
	mock.lockFilterClient.RLock()
	calls = mock.calls.FilterClient
	mock.lockFilterClient.RUnlock()
	return calls
}

// HealthCheck calls HealthCheckFunc.
func (mock *DependenciesMock) HealthCheck(configMoqParam *config.Config, s1 string, s2 string, s3 string) (service.HealthChecker, error) {
	if mock.HealthCheckFunc == nil {
		panic("DependenciesMock.HealthCheckFunc: method is nil but Dependencies.HealthCheck was just called")
	}
	callInfo := struct {
		ConfigMoqParam *config.Config
		S1             string
		S2             string
		S3             string
	}{
		ConfigMoqParam: configMoqParam,
		S1:             s1,
		S2:             s2,
		S3:             s3,
	}
	mock.lockHealthCheck.Lock()
	mock.calls.HealthCheck = append(mock.calls.HealthCheck, callInfo)
	mock.lockHealthCheck.Unlock()
	return mock.HealthCheckFunc(configMoqParam, s1, s2, s3)
}

// HealthCheckCalls gets all the calls that were made to HealthCheck.
// Check the length with:
//     len(mockedDependencies.HealthCheckCalls())
func (mock *DependenciesMock) HealthCheckCalls() []struct {
	ConfigMoqParam *config.Config
	S1             string
	S2             string
	S3             string
} {
	var calls []struct {
		ConfigMoqParam *config.Config
		S1             string
		S2             string
		S3             string
	}
	mock.lockHealthCheck.RLock()
	calls = mock.calls.HealthCheck
	mock.lockHealthCheck.RUnlock()
	return calls
}

// HttpServer calls HttpServerFunc.
func (mock *DependenciesMock) HttpServer(configMoqParam *config.Config, handler http.Handler) service.HTTPServer {
	if mock.HttpServerFunc == nil {
		panic("DependenciesMock.HttpServerFunc: method is nil but Dependencies.HttpServer was just called")
	}
	callInfo := struct {
		ConfigMoqParam *config.Config
		Handler        http.Handler
	}{
		ConfigMoqParam: configMoqParam,
		Handler:        handler,
	}
	mock.lockHttpServer.Lock()
	mock.calls.HttpServer = append(mock.calls.HttpServer, callInfo)
	mock.lockHttpServer.Unlock()
	return mock.HttpServerFunc(configMoqParam, handler)
}

// HttpServerCalls gets all the calls that were made to HttpServer.
// Check the length with:
//     len(mockedDependencies.HttpServerCalls())
func (mock *DependenciesMock) HttpServerCalls() []struct {
	ConfigMoqParam *config.Config
	Handler        http.Handler
} {
	var calls []struct {
		ConfigMoqParam *config.Config
		Handler        http.Handler
	}
	mock.lockHttpServer.RLock()
	calls = mock.calls.HttpServer
	mock.lockHttpServer.RUnlock()
	return calls
}

// ImageClient calls ImageClientFunc.
func (mock *DependenciesMock) ImageClient(s string) downloads.ImageClient {
	if mock.ImageClientFunc == nil {
		panic("DependenciesMock.ImageClientFunc: method is nil but Dependencies.ImageClient was just called")
	}
	callInfo := struct {
		S string
	}{
		S: s,
	}
	mock.lockImageClient.Lock()
	mock.calls.ImageClient = append(mock.calls.ImageClient, callInfo)
	mock.lockImageClient.Unlock()
	return mock.ImageClientFunc(s)
}

// ImageClientCalls gets all the calls that were made to ImageClient.
// Check the length with:
//     len(mockedDependencies.ImageClientCalls())
func (mock *DependenciesMock) ImageClientCalls() []struct {
	S string
} {
	var calls []struct {
		S string
	}
	mock.lockImageClient.RLock()
	calls = mock.calls.ImageClient
	mock.lockImageClient.RUnlock()
	return calls
}

// S3Client calls S3ClientFunc.
func (mock *DependenciesMock) S3Client(configMoqParam *config.Config) (content.S3Client, error) {
	if mock.S3ClientFunc == nil {
		panic("DependenciesMock.S3ClientFunc: method is nil but Dependencies.S3Client was just called")
	}
	callInfo := struct {
		ConfigMoqParam *config.Config
	}{
		ConfigMoqParam: configMoqParam,
	}
	mock.lockS3Client.Lock()
	mock.calls.S3Client = append(mock.calls.S3Client, callInfo)
	mock.lockS3Client.Unlock()
	return mock.S3ClientFunc(configMoqParam)
}

// S3ClientCalls gets all the calls that were made to S3Client.
// Check the length with:
//     len(mockedDependencies.S3ClientCalls())
func (mock *DependenciesMock) S3ClientCalls() []struct {
	ConfigMoqParam *config.Config
} {
	var calls []struct {
		ConfigMoqParam *config.Config
	}
	mock.lockS3Client.RLock()
	calls = mock.calls.S3Client
	mock.lockS3Client.RUnlock()
	return calls
}

// VaultClient calls VaultClientFunc.
func (mock *DependenciesMock) VaultClient(configMoqParam *config.Config) (content.VaultClient, error) {
	if mock.VaultClientFunc == nil {
		panic("DependenciesMock.VaultClientFunc: method is nil but Dependencies.VaultClient was just called")
	}
	callInfo := struct {
		ConfigMoqParam *config.Config
	}{
		ConfigMoqParam: configMoqParam,
	}
	mock.lockVaultClient.Lock()
	mock.calls.VaultClient = append(mock.calls.VaultClient, callInfo)
	mock.lockVaultClient.Unlock()
	return mock.VaultClientFunc(configMoqParam)
}

// VaultClientCalls gets all the calls that were made to VaultClient.
// Check the length with:
//     len(mockedDependencies.VaultClientCalls())
func (mock *DependenciesMock) VaultClientCalls() []struct {
	ConfigMoqParam *config.Config
} {
	var calls []struct {
		ConfigMoqParam *config.Config
	}
	mock.lockVaultClient.RLock()
	calls = mock.calls.VaultClient
	mock.lockVaultClient.RUnlock()
	return calls
}

// Ensure, that HealthCheckerMock does implement service.HealthChecker.
// If this is not the case, regenerate this file with moq.
var _ service.HealthChecker = &HealthCheckerMock{}

// HealthCheckerMock is a mock implementation of service.HealthChecker.
//
// 	func TestSomethingThatUsesHealthChecker(t *testing.T) {
//
// 		// make and configure a mocked service.HealthChecker
// 		mockedHealthChecker := &HealthCheckerMock{
// 			AddCheckFunc: func(s string, checker healthcheck.Checker) error {
// 				panic("mock out the AddCheck method")
// 			},
// 			HandlerFunc: func(responseWriter http.ResponseWriter, request *http.Request)  {
// 				panic("mock out the Handler method")
// 			},
// 			StartFunc: func(contextMoqParam context.Context)  {
// 				panic("mock out the Start method")
// 			},
// 			StopFunc: func()  {
// 				panic("mock out the Stop method")
// 			},
// 		}
//
// 		// use mockedHealthChecker in code that requires service.HealthChecker
// 		// and then make assertions.
//
// 	}
type HealthCheckerMock struct {
	// AddCheckFunc mocks the AddCheck method.
	AddCheckFunc func(s string, checker healthcheck.Checker) error

	// HandlerFunc mocks the Handler method.
	HandlerFunc func(responseWriter http.ResponseWriter, request *http.Request)

	// StartFunc mocks the Start method.
	StartFunc func(contextMoqParam context.Context)

	// StopFunc mocks the Stop method.
	StopFunc func()

	// calls tracks calls to the methods.
	calls struct {
		// AddCheck holds details about calls to the AddCheck method.
		AddCheck []struct {
			// S is the s argument value.
			S string
			// Checker is the checker argument value.
			Checker healthcheck.Checker
		}
		// Handler holds details about calls to the Handler method.
		Handler []struct {
			// ResponseWriter is the responseWriter argument value.
			ResponseWriter http.ResponseWriter
			// Request is the request argument value.
			Request *http.Request
		}
		// Start holds details about calls to the Start method.
		Start []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
		}
		// Stop holds details about calls to the Stop method.
		Stop []struct {
		}
	}
	lockAddCheck sync.RWMutex
	lockHandler  sync.RWMutex
	lockStart    sync.RWMutex
	lockStop     sync.RWMutex
}

// AddCheck calls AddCheckFunc.
func (mock *HealthCheckerMock) AddCheck(s string, checker healthcheck.Checker) error {
	if mock.AddCheckFunc == nil {
		panic("HealthCheckerMock.AddCheckFunc: method is nil but HealthChecker.AddCheck was just called")
	}
	callInfo := struct {
		S       string
		Checker healthcheck.Checker
	}{
		S:       s,
		Checker: checker,
	}
	mock.lockAddCheck.Lock()
	mock.calls.AddCheck = append(mock.calls.AddCheck, callInfo)
	mock.lockAddCheck.Unlock()
	return mock.AddCheckFunc(s, checker)
}

// AddCheckCalls gets all the calls that were made to AddCheck.
// Check the length with:
//     len(mockedHealthChecker.AddCheckCalls())
func (mock *HealthCheckerMock) AddCheckCalls() []struct {
	S       string
	Checker healthcheck.Checker
} {
	var calls []struct {
		S       string
		Checker healthcheck.Checker
	}
	mock.lockAddCheck.RLock()
	calls = mock.calls.AddCheck
	mock.lockAddCheck.RUnlock()
	return calls
}

// Handler calls HandlerFunc.
func (mock *HealthCheckerMock) Handler(responseWriter http.ResponseWriter, request *http.Request) {
	if mock.HandlerFunc == nil {
		panic("HealthCheckerMock.HandlerFunc: method is nil but HealthChecker.Handler was just called")
	}
	callInfo := struct {
		ResponseWriter http.ResponseWriter
		Request        *http.Request
	}{
		ResponseWriter: responseWriter,
		Request:        request,
	}
	mock.lockHandler.Lock()
	mock.calls.Handler = append(mock.calls.Handler, callInfo)
	mock.lockHandler.Unlock()
	mock.HandlerFunc(responseWriter, request)
}

// HandlerCalls gets all the calls that were made to Handler.
// Check the length with:
//     len(mockedHealthChecker.HandlerCalls())
func (mock *HealthCheckerMock) HandlerCalls() []struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
} {
	var calls []struct {
		ResponseWriter http.ResponseWriter
		Request        *http.Request
	}
	mock.lockHandler.RLock()
	calls = mock.calls.Handler
	mock.lockHandler.RUnlock()
	return calls
}

// Start calls StartFunc.
func (mock *HealthCheckerMock) Start(contextMoqParam context.Context) {
	if mock.StartFunc == nil {
		panic("HealthCheckerMock.StartFunc: method is nil but HealthChecker.Start was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
	}{
		ContextMoqParam: contextMoqParam,
	}
	mock.lockStart.Lock()
	mock.calls.Start = append(mock.calls.Start, callInfo)
	mock.lockStart.Unlock()
	mock.StartFunc(contextMoqParam)
}

// StartCalls gets all the calls that were made to Start.
// Check the length with:
//     len(mockedHealthChecker.StartCalls())
func (mock *HealthCheckerMock) StartCalls() []struct {
	ContextMoqParam context.Context
} {
	var calls []struct {
		ContextMoqParam context.Context
	}
	mock.lockStart.RLock()
	calls = mock.calls.Start
	mock.lockStart.RUnlock()
	return calls
}

// Stop calls StopFunc.
func (mock *HealthCheckerMock) Stop() {
	if mock.StopFunc == nil {
		panic("HealthCheckerMock.StopFunc: method is nil but HealthChecker.Stop was just called")
	}
	callInfo := struct {
	}{}
	mock.lockStop.Lock()
	mock.calls.Stop = append(mock.calls.Stop, callInfo)
	mock.lockStop.Unlock()
	mock.StopFunc()
}

// StopCalls gets all the calls that were made to Stop.
// Check the length with:
//     len(mockedHealthChecker.StopCalls())
func (mock *HealthCheckerMock) StopCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockStop.RLock()
	calls = mock.calls.Stop
	mock.lockStop.RUnlock()
	return calls
}

// Ensure, that HTTPServerMock does implement service.HTTPServer.
// If this is not the case, regenerate this file with moq.
var _ service.HTTPServer = &HTTPServerMock{}

// HTTPServerMock is a mock implementation of service.HTTPServer.
//
// 	func TestSomethingThatUsesHTTPServer(t *testing.T) {
//
// 		// make and configure a mocked service.HTTPServer
// 		mockedHTTPServer := &HTTPServerMock{
// 			ListenAndServeFunc: func() error {
// 				panic("mock out the ListenAndServe method")
// 			},
// 			ShutdownFunc: func(ctx context.Context) error {
// 				panic("mock out the Shutdown method")
// 			},
// 		}
//
// 		// use mockedHTTPServer in code that requires service.HTTPServer
// 		// and then make assertions.
//
// 	}
type HTTPServerMock struct {
	// ListenAndServeFunc mocks the ListenAndServe method.
	ListenAndServeFunc func() error

	// ShutdownFunc mocks the Shutdown method.
	ShutdownFunc func(ctx context.Context) error

	// calls tracks calls to the methods.
	calls struct {
		// ListenAndServe holds details about calls to the ListenAndServe method.
		ListenAndServe []struct {
		}
		// Shutdown holds details about calls to the Shutdown method.
		Shutdown []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
	}
	lockListenAndServe sync.RWMutex
	lockShutdown       sync.RWMutex
}

// ListenAndServe calls ListenAndServeFunc.
func (mock *HTTPServerMock) ListenAndServe() error {
	if mock.ListenAndServeFunc == nil {
		panic("HTTPServerMock.ListenAndServeFunc: method is nil but HTTPServer.ListenAndServe was just called")
	}
	callInfo := struct {
	}{}
	mock.lockListenAndServe.Lock()
	mock.calls.ListenAndServe = append(mock.calls.ListenAndServe, callInfo)
	mock.lockListenAndServe.Unlock()
	return mock.ListenAndServeFunc()
}

// ListenAndServeCalls gets all the calls that were made to ListenAndServe.
// Check the length with:
//     len(mockedHTTPServer.ListenAndServeCalls())
func (mock *HTTPServerMock) ListenAndServeCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockListenAndServe.RLock()
	calls = mock.calls.ListenAndServe
	mock.lockListenAndServe.RUnlock()
	return calls
}

// Shutdown calls ShutdownFunc.
func (mock *HTTPServerMock) Shutdown(ctx context.Context) error {
	if mock.ShutdownFunc == nil {
		panic("HTTPServerMock.ShutdownFunc: method is nil but HTTPServer.Shutdown was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	mock.lockShutdown.Lock()
	mock.calls.Shutdown = append(mock.calls.Shutdown, callInfo)
	mock.lockShutdown.Unlock()
	return mock.ShutdownFunc(ctx)
}

// ShutdownCalls gets all the calls that were made to Shutdown.
// Check the length with:
//     len(mockedHTTPServer.ShutdownCalls())
func (mock *HTTPServerMock) ShutdownCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	mock.lockShutdown.RLock()
	calls = mock.calls.Shutdown
	mock.lockShutdown.RUnlock()
	return calls
}
