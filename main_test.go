package main_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/ONSdigital/log.go/v2/log"

	componenttest "github.com/ONSdigital/dp-component-test"
	"github.com/ONSdigital/dp-download-service/features/steps"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var componentFlag = flag.Bool("component", false, "perform component tests")
var loggingFlag = flag.Bool("logging", false, "output application logging")

type ComponentTest struct {
}

func (t *ComponentTest) InitializeScenario(ctx *godog.ScenarioContext) {
	if !*loggingFlag {
		buf := bytes.NewBufferString("")
		log.SetDestination(buf, buf)
	}

	authorizationFeature := componenttest.NewAuthorizationFeature()

	component := steps.NewDownloadServiceComponent(authorizationFeature.FakeAuthService.ResolveURL(""))
	apiFeature := componenttest.NewAPIFeature(component.Initialiser)
	component.ApiFeature = apiFeature

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		component.Reset()
		apiFeature.Reset()
		authorizationFeature.Reset()
		authorizationFeature.FakeAuthService.NewHandler().Get("/health").Reply(http.StatusOK)

		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		err = component.Close()
		if err != nil {
			log.Error(ctx, "error closing service", err)
		}
		return ctx, nil
	})

	apiFeature.RegisterSteps(ctx)
	authorizationFeature.RegisterSteps(ctx)
	component.RegisterSteps(ctx)
}

func (t *ComponentTest) InitializeTestSuite(ctx *godog.TestSuiteContext) {
}

func TestComponent(t *testing.T) {
	if *componentFlag {
		status := 0

		var opts = godog.Options{
			Output: colors.Colored(os.Stdout),
			Format: "pretty",
			Paths:  flag.Args(),
		}

		f := &ComponentTest{}

		status = godog.TestSuite{
			Name:                 "feature_tests",
			ScenarioInitializer:  f.InitializeScenario,
			TestSuiteInitializer: f.InitializeTestSuite,
			Options:              &opts,
		}.Run()

		fmt.Println("=================================")
		fmt.Printf("Component test coverage: %.2f%%\n", testing.Coverage()*100)
		fmt.Println("=================================")

		if status > 0 {
			t.Fail()
		}
	} else {
		t.Skip("component flag required to run component tests")
	}
}
