package release

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	cfront "github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"github.com/pilot-framework/aws-cloudfront-waypoint-plugin/platform"
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
	CreateDistributionWithTags(
		ctx context.Context,
		input *cfront.CreateDistributionWithTagsInput,
		optFns ...func(*cfront.Options),
	) (*cfront.CreateDistributionWithTagsOutput, error)
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

func CreateDistribution(
	c context.Context,
	api CloudfrontAPI,
	input *cfront.CreateDistributionWithTagsInput,
) (*cfront.CreateDistributionWithTagsOutput, error) {
	return api.CreateDistributionWithTags(c, input)
}

func FormatDistributionInput(bucket string) (*cfront.CreateDistributionWithTagsInput) {
	tagKey := "bucket"
	items := [](types.Tag){
		types.Tag{Key: &tagKey, Value: &bucket},
	}

	callRef := fmt.Sprintf("pilot-ref-%v", time.Now())
	comment := "This distribution was created via Pilot"
	enabled := false


	input := &cfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &types.DistributionConfigWithTags{
			DistributionConfig: &types.DistributionConfig{
				CallerReference: &callRef,
				Comment: &comment,
				DefaultCacheBehavior: &types.DefaultCacheBehavior{},
				Enabled: &enabled,
				Origins: &types.Origins{},
			},
			Tags: &types.Tags{
				Items: items,
			},
		},
	}

	return input
}

type ReleaseConfig struct {}

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
func (rm *ReleaseManager) release(ctx context.Context, ui terminal.UI, target *platform.Deployment) (*Release, error) {
	u := ui.Status()
	defer u.Close()
	u.Step("", "--- Configuring AWS Cloudfront ---")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return nil, err
	}

	client := cfront.NewFromConfig(cfg)

	u.Update("Checking for existing distributions...")
	dists, err := GetAllDistributions(context.TODO(), client, &cfront.ListDistributionsInput{})

	if err != nil {
		u.Step(terminal.StatusError, "Error retrieving distributions")
		return nil, err
	}

	u.Update("Searching for distribution belonging to "+target.Bucket+"...")

	var existingDistribution *string

	for _, v := range dists.DistributionList.Items {
		tagInput := &cfront.ListTagsForResourceInput{
			Resource: v.ARN,
		}

		tags, err := GetDistributionTags(context.TODO(), client, tagInput)
		if err != nil {
			u.Step(terminal.StatusError, fmt.Sprintf("Error retrieving tags for %v", *v.ARN))
			return nil, err
		}

		for _, tag := range tags.Tags.Items {
			if *tag.Key == "bucket" && *tag.Value == target.Bucket {
				existingDistribution = v.ARN
			}
		}
	}

	// TODO: if no existingDistribution, call creation methods (with tag of bucket - BucketName)
	if existingDistribution == nil {
		u.Step("", fmt.Sprintf("Could not find distribution belonging to %v, creating new distribution...", target.Bucket))

		newDistInput := FormatDistributionInput(target.Bucket)

		_, err := CreateDistribution(context.TODO(), client, newDistInput)
		if err != nil {
			u.Step(terminal.StatusError, fmt.Sprintf("Error creating distribution: %v", err.Error()))

			return nil, err
		}
	// TODO: if existingDistribution, call update methods
	} else {
		u.Step(terminal.StatusOK, fmt.Sprintf("Found the following distribution: %v", *existingDistribution))
	}

	return &Release{}, nil
}
