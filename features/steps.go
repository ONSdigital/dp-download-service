package features

import "github.com/cucumber/godog"

func (d *DownloadServiceComponent) iRequestTODownloadTheFile(filename string) error {
        return d.ApiFeature.IGet("/downloads/" + filename)
}

func (d *DownloadServiceComponent) RegisterSteps(ctx *godog.ScenarioContext) {
        ctx.Step(`^I request to download the file "([^"]*)"$`, d.iRequestTODownloadTheFile)
}

