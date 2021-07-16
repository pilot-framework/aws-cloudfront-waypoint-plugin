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
	CreateOriginRequestPolicy(
		ctx context.Context,
		input *cfront.CreateOriginRequestPolicyInput,
		optFns ...func(*cfront.Options),
	) (*cfront.CreateOriginRequestPolicyOutput, error)
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

func CreateOrigin(
	c context.Context,
	api CloudfrontAPI,
	input *cfront.CreateOriginRequestPolicyInput,
) (*cfront.CreateOriginRequestPolicyOutput, error) {
	return api.CreateOriginRequestPolicy(c, input)
}

// This function will create the configuration needed to create a new origin
func FormatOrigin(bucket string, region string, root string) (types.Origin) {
	var http int32 = 80
	var https int32 = 443
	var keepAlive int32 = 5
	var readTimeout int32 = 30

	config := &types.CustomOriginConfig{
		HTTPPort: &http,
		HTTPSPort: &https,
		OriginKeepaliveTimeout: &keepAlive,
		OriginProtocolPolicy: types.OriginProtocolPolicyHttpOnly,
		OriginReadTimeout: &readTimeout,
	}

	var connAttempts int32 = 3
	var connTimeout int32 = 10
	var domainName string = fmt.Sprintf("%v.s3-website.%v.amazonaws.com", bucket, region)
	var originId string = fmt.Sprintf("pilot-origin-%v", bucket)
	var originPath string = root
	origin := types.Origin{
		ConnectionAttempts: &connAttempts,
		ConnectionTimeout: &connTimeout,
		CustomOriginConfig: config,
		DomainName: &domainName,
		Id: &originId,
		OriginPath: &originPath,
	}

	return origin
}

// This function will create the configuration input needed to create a new distribution
func FormatDistributionInput(bucket string, region string, root string) (*cfront.CreateDistributionWithTagsInput) {
	// These are the tags that the distribution will have
	// by default we include a bucket - bucket_name k/v to check if a distribution exists
	tagKey := "bucket"
	items := [](types.Tag){
		types.Tag{Key: &tagKey, Value: &bucket},
	}

	callRef := fmt.Sprintf("pilot-ref-%v", time.Now()) // unique identifier for the request
	comment := "This distribution was created via Pilot"
	enabled := true
	var quantity int32 = 1
	origin := FormatOrigin(bucket, region, root)
	// this is the ID for the Managed-CachingOptimized policy
	cachePolicy := "658327ea-f89d-4fab-a63d-7e88639e58f6"

	input := &cfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &types.DistributionConfigWithTags{
			DistributionConfig: &types.DistributionConfig{
				CallerReference: &callRef,
				Comment: &comment,
				DefaultCacheBehavior: &types.DefaultCacheBehavior{
					TargetOriginId: origin.Id,
					ViewerProtocolPolicy: types.ViewerProtocolPolicyAllowAll,
					CachePolicyId: &cachePolicy,
				},
				Enabled: &enabled,
				Origins: &types.Origins{
					Quantity: &quantity,
					Items: [](types.Origin){origin},
				},
			},
			Tags: &types.Tags{
				Items: items,
			},
		},
	}

	return input
}

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

	distExists := false

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
				distExists = true
				break
			}
		}
	}

	if !distExists {
		u.Step("", fmt.Sprintf("Could not find distribution belonging to %v, creating new distribution...", target.Bucket))

		newDistInput := FormatDistributionInput(target.Bucket, target.Region, rm.config.Root)

		newDist, err := CreateDistribution(context.TODO(), client, newDistInput)
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
