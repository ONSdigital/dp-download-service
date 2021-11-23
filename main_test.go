package main

import (
	"flag"
	componenttest "github.com/ONSdigital/dp-component-test"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"net/http"
	"os"
	"testing"
	"github.com/ONSdigital/dp-download-service/features"
)

var componentFlag = flag.Bool("component", false, "perform component tests")

type componentTestSuite struct {
	Mongo *componenttest.MongoFeature
}
func NewServer() *http.Server {
	return &http.Server{}
}

func (t *componentTestSuite) InitializeScenario(ctx *godog.ScenarioContext) {
	server := NewServer()

	component := features.NewDownloadServiceComponent(server.Handler, t.Mongo.Server.URI()) // This is the part that YOU will implement
	apiFeature := componenttest.NewAPIFeature(component.Initialiser(server.Handler))

	ctx.BeforeScenario(func(*godog.Scenario) {
		t.Mongo.Reset()
		apiFeature.Reset()
	})

	ctx.AfterScenario(func(*godog.Scenario, error) {
		t.Mongo.Reset()
		apiFeature.Reset()
	})

	apiFeature.RegisterSteps(ctx)
	t.Mongo.RegisterSteps(ctx)
}

func (t *componentTestSuite) InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		mongoOptions := componenttest.MongoOptions{
			MongoVersion: "4.4.8",
			DatabaseName: "testing",
		}
		t.Mongo = componenttest.NewMongoFeature(mongoOptions)
	})

	ctx.AfterSuite(func() {
		t.Mongo.Close()
	})
}


func TestMain(t *testing.T) {
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