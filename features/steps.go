package features

import "github.com/cucumber/godog"

func (d *DownloadServiceComponent) iRequestTODownloadTheFile(filename string) error {
        return d.ApiFeature.IGet("/downloads/" + filename)
}

func (d *DownloadServiceComponent) RegisterSteps(ctx *godog.ScenarioContext) {
        ctx.Step(`^I request to download the file "([^"]*)"$`, d.iRequestTODownloadTheFile)
        ctx.Step(`^I should receive the private file "([^"]*)"$`, iShouldReceiveThePrivateFile)
        ctx.Step(`^is not yet published$`, isNotYetPublished)
        ctx.Step(`^the file "([^"]*)" has been uploaded$`, theFileHasBeenUploaded)

}

func (d *DownloadServiceComponent) iShouldReceiveThePrivateFile(arg1 string) error {
        return godog.ErrPending
}

func (d *DownloadServiceComponent) isNotYetPublished() error {
        return godog.ErrPending
}

func (d *DownloadServiceComponent) theFileHasBeenUploaded(arg1 string) error {
        return godog.ErrPending
}



