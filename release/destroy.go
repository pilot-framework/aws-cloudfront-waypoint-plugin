package release

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/pilot-framework/aws-cloudfront-waypoint-plugin/cfront"
)

// Implement the Destroyer interface
func (rm *ReleaseManager) DestroyFunc() interface{} {
	return rm.destroy
}

func (rm *ReleaseManager) destroy(ctx context.Context, ui terminal.UI, release *Release) error {
	u := ui.Status()
	defer u.Close()
	u.Step("", "--- Destroying AWS Cloudfront Distribution ---")
	u.Step("", "NOTICE: This can take awhile")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return err
	}

	client := cloudfront.NewFromConfig(cfg)

	pollStatus := make(chan bool, 1)
	pollError := make(chan error)

	u.Update("Disabling distribution...")

	err = cfront.DisableDistribution(release.Id, client)

	if err != nil {
		return err
	}

	go cfront.PollStatus(release.Id, client, pollStatus, pollError)

	s := <-pollStatus

	if !s {
		return <-pollError
	}

	u.Step(terminal.StatusOK, "Distribution disabled")

	return nil
}
