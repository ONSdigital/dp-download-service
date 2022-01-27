package features

import (
        "github.com/cucumber/godog"
        "github.com/stretchr/testify/assert"
        "net/http"
)

func (d *DownloadServiceComponent) iRequestTODownloadTheFile(filename string) error {
        return d.ApiFeature.IGet("/downloads/" + filename)
}

func (d *DownloadServiceComponent) RegisterSteps(ctx *godog.ScenarioContext) {
        ctx.Step(`^I request to download the file "([^"]*)"$`, d.iRequestTODownloadTheFile)
        ctx.Step(`^I should receive the private file "([^"]*)"$`, d.iShouldReceiveThePrivateFile)
        ctx.Step(`^is not yet published$`, d.isNotYetPublished)
        ctx.Step(`^the file "([^"]*)" has been uploaded$`, d.theFileHasBeenUploaded)

}

func (d *DownloadServiceComponent) iShouldReceiveThePrivateFile(filename string) error {
        assert.Equal(d.ApiFeature, http.StatusOK, d.ApiFeature.HttpResponse.StatusCode)
        assert.Equal(d.ApiFeature, "attachment; filename="+filename, d.ApiFeature.HttpResponse.Header.Get("Content-Disposition"))

        //return errors.New("BROKE")
        return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) isNotYetPublished() error {
        return nil
}

func (d *DownloadServiceComponent) theFileHasBeenUploaded(arg1 string) error {
        return nil
}



