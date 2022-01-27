package main_test

import (
	"context"
	"flag"
	componenttest "github.com/ONSdigital/dp-component-test"
	"github.com/ONSdigital/dp-download-service/features"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"os"
	"testing"
)

var componentFlag = flag.Bool("component", false, "perform component tests")

type componentTestSuite struct {

}

func (t *componentTestSuite) InitializeScenario(ctx *godog.ScenarioContext) {
	authorizationFeature := componenttest.NewAuthorizationFeature()
	component := features.NewDownloadServiceComponent( authorizationFeature.FakeAuthService.ResolveURL(""))
	apiFeature := componenttest.NewAPIFeature(component.Initialiser)
	component.ApiFeature = apiFeature

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		apiFeature.Reset()
		authorizationFeature.Reset()

		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		//t.Mongo.Reset()
		//apiFeature.Reset()
		return ctx, nil
	})

	apiFeature.RegisterSteps(ctx)
	authorizationFeature.RegisterSteps(ctx)
	component.RegisterSteps(ctx)
}

func (t *componentTestSuite) InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {

	})

	ctx.AfterSuite(func() {
	})
}

func TestSomething(t *testing.T) {
	if *componentFlag {
		var opts = godog.Options{
			Output: colors.Colored(os.Stdout),
			Format: "pretty",
			Paths:  flag.Args(),
		}

		f := &componentTestSuite{}

		status := godog.TestSuite{
			Name:                 "component_tests",
			ScenarioInitializer:  f.InitializeScenario,
			TestSuiteInitializer: f.InitializeTestSuite,
			Options:              &opts,
		}.Run()

		if status > 0 {
			t.Fail()
		}
	} else {
		t.Skip("component flag required to run component tests")
	}
}
