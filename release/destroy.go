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
	u.Step("", "\n--- Destroying AWS Cloudfront Distribution ---")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return err
	}

	client := cloudfront.NewFromConfig(cfg)

	u.Update("Disabling distribution...")

	err = cfront.DisableDistribution(release.Id, client)

	if err != nil {
		return err
	}

	go cfront.RemoveDistribution(release.Id, client)

	u.Step(terminal.StatusOK, "Scheduled distribution for deletion")

	return nil
}
