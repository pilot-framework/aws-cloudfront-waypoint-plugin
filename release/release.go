package release

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/pilot-framework/aws-cloudfront-waypoint-plugin/cfront"
	"github.com/pilot-framework/aws-cloudfront-waypoint-plugin/platform"
)

type ReleaseConfig struct {
	// This is the Origin Path that the CDN will treat as `/`
	// default is a 1-1 forward to `/`
	Root string `hcl:"root,optional"`
}

type ReleaseManager struct {
	config ReleaseConfig
}

// Implement Configurable
func (rm *ReleaseManager) Config() (interface{}, error) {
	return &rm.config, nil
}

// Implement ConfigurableNotify
func (rm *ReleaseManager) ConfigSet(config interface{}) error {
	_, ok := config.(*ReleaseConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("expected *ReleaseConfig as parameter")
	}

	// validate the config

	return nil
}

// Implement Builder
func (rm *ReleaseManager) ReleaseFunc() interface{} {
	// return a function which will be called by Waypoint
	return rm.release
}

// In addition to default input parameters the platform.Deployment from the Deploy step
// can also be injected.
//
// The output parameters for ReleaseFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
//
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (rm *ReleaseManager) release(ctx context.Context, ui terminal.UI, target *platform.Deployment) (*Release, error) {
	u := ui.Status()
	defer u.Close()
	u.Step("", "--- Configuring AWS Cloudfront ---")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return nil, err
	}

	client := cloudfront.NewFromConfig(cfg)

	u.Update("Checking for existing distributions...")
	dists, err := cfront.GetAllDistributions(context.TODO(), client, &cloudfront.ListDistributionsInput{})

	if err != nil {
		u.Step(terminal.StatusError, "Error retrieving distributions")
		return nil, err
	}

	u.Update("Searching for distribution belonging to " + target.Bucket + "...")

	distExists := false

	for _, v := range dists.DistributionList.Items {
		tagInput := &cloudfront.ListTagsForResourceInput{
			Resource: v.ARN,
		}

		tags, err := cfront.GetDistributionTags(context.TODO(), client, tagInput)
		if err != nil {
			u.Step(terminal.StatusError, fmt.Sprintf("Error retrieving tags for %v", *v.ARN))
			return nil, err
		}

		for _, tag := range tags.Tags.Items {
			if *tag.Key == "bucket" && *tag.Value == target.Bucket {
				distExists = true
				break
			}
		}
	}

	if !distExists {
		u.Step("", fmt.Sprintf("Could not find distribution belonging to %v, creating new distribution...", target.Bucket))

		newDistInput := cfront.FormatDistributionInput(target.Bucket, target.Region, rm.config.Root)

		newDist, err := cfront.CreateDistribution(context.TODO(), client, newDistInput)
		if err != nil {
			u.Step(terminal.StatusError, fmt.Sprintf("Error creating distribution: %v", err.Error()))

			return nil, err
		}

		u.Step(terminal.StatusOK,
			fmt.Sprintf(
				"Successfully created distribution %v.\nCloudfront URL: %v\nPlease note it may take a few minutes for changes to propagate.",
				*newDist.Distribution.Id,
				*newDist.Distribution.DomainName,
			))
	} else {
		u.Step(terminal.StatusOK, fmt.Sprintf("Found an existing distribution for %v", target.Bucket))
	}

	return &Release{}, nil
}
