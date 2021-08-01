package release

import (
	"context"
	"fmt"

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

	u.Step("", "Starting goroutine")

	go cfront.PollStatus(release.Id, client, pollStatus, pollError)

	u.Step("", "Waiting on goroutine")

	s := <-pollStatus

	u.Step("", fmt.Sprintf("Received status of %v", s))

	err = <-pollError

	if err != nil && !s {
		u.Step("", fmt.Sprintf("Recieved pollError: %v", err.Error()))
		return err
	}

	u.Step(terminal.StatusOK, "Distribution disabled")

	u.Update("Deleting distribution...")

	err = cfront.RemoveDistribution(release.Id, client)

	if err != nil {
		return err
	}

	u.Step(terminal.StatusOK, "Deleted distribution")

	return nil
}
