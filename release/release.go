package release

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	cfront "github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

// Defines interface for needed Cloudfront functions
type CloudfrontAPI interface {
	GetDistribution(
		ctx context.Context,
		input *cfront.GetDistributionInput,
		optFns ...func(*cfront.Options),
	) (*cfront.GetDistributionOutput, error)
	ListDistributions(
		ctx context.Context,
		input *cfront.ListDistributionsInput,
		optFns ...func(*cfront.Options),
	) (*cfront.ListDistributionsOutput, error)
	ListTagsForResource(
		ctx context.Context,
		input *cfront.ListTagsForResourceInput,
		optFns ...func(*cfront.Options),
	) (*cfront.ListTagsForResourceOutput, error)
}

func DistributionExists(
	c context.Context,
	api CloudfrontAPI,
	input *cfront.GetDistributionInput,
) (*cfront.GetDistributionOutput, error) {
	return api.GetDistribution(c, input)
}

func GetAllDistributions(
	c context.Context,
	api CloudfrontAPI,
	input *cfront.ListDistributionsInput,
) (*cfront.ListDistributionsOutput, error) {
	return api.ListDistributions(c, input)
}

func GetDistributionTags(
	c context.Context,
	api CloudfrontAPI,
	input *cfront.ListTagsForResourceInput,
) (*cfront.ListTagsForResourceOutput, error) {
	return api.ListTagsForResource(c, input)
}

type ReleaseConfig struct {
	BucketName string `hcl:"bucket"`
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
		return fmt.Errorf("Expected *ReleaseConfig as parameter")
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
func (rm *ReleaseManager) release(ctx context.Context, ui terminal.UI) (*Release, error) {
	u := ui.Status()
	defer u.Close()
	u.Update("Configuring AWS Cloudfront...")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return nil, err
	}

	client := cfront.NewFromConfig(cfg)

	input := &cfront.GetDistributionInput{
		Id: &rm.config.BucketName,
	}



	u.Step(terminal.StatusOK, "Checking for existing distributions...")
	dists, err := GetAllDistributions(context.TODO(), client, &cfront.ListDistributionsInput{})

	if err != nil {
		u.Step(terminal.StatusError, "Error retrieving distributions")
		return nil, err
	}

	for _, v := range dists.DistributionList.Items {
		tagInput := &cfront.ListTagsForResourceInput{
			Resource: v.ARN,
		}

		tags, err := GetDistributionTags(context.TODO(), client, tagInput)
		if err != nil {
			u.Step(terminal.StatusError, fmt.Sprintf("Error retrieving tags for %+v", v.ARN))
			return nil, err
		}

		for _, tag := range tags.Tags.Items {
			u.Step(terminal.StatusOK, fmt.Sprintf("Found the following for %v: %v - %v", *v.ARN, *tag.Key, *tag.Value))
		}
	}
	
	_, err = DistributionExists(context.TODO(), client, input)
	if err != nil {
		u.Step(terminal.StatusError, "Error creating distribution for "+rm.config.BucketName)
		return nil, err
	}

	return &Release{}, nil
}
