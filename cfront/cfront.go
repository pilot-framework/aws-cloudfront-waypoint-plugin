package cfront

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

// Defines interface for needed Cloudfront functions
type CloudfrontAPI interface {
	GetDistribution(
		ctx context.Context,
		input *cloudfront.GetDistributionInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.GetDistributionOutput, error)
	DeleteDistribution(
		ctx context.Context,
		input *cloudfront.DeleteDistributionInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.DeleteDistributionOutput, error)
	ListDistributions(
		ctx context.Context,
		input *cloudfront.ListDistributionsInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.ListDistributionsOutput, error)
	ListTagsForResource(
		ctx context.Context,
		input *cloudfront.ListTagsForResourceInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.ListTagsForResourceOutput, error)
	CreateDistributionWithTags(
		ctx context.Context,
		input *cloudfront.CreateDistributionWithTagsInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.CreateDistributionWithTagsOutput, error)
	CreateOriginRequestPolicy(
		ctx context.Context,
		input *cloudfront.CreateOriginRequestPolicyInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.CreateOriginRequestPolicyOutput, error)
	DeleteOriginRequestPolicy(
		ctx context.Context,
		input *cloudfront.DeleteOriginRequestPolicyInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.DeleteOriginRequestPolicyOutput, error)
}

func GetAllDistributions(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.ListDistributionsInput,
) (*cloudfront.ListDistributionsOutput, error) {
	return api.ListDistributions(c, input)
}

func DeleteDistribution(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.DeleteDistributionInput,
) (*cloudfront.DeleteDistributionOutput, error) {
	return api.DeleteDistribution(c, input)
}

func GetDistributionTags(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.ListTagsForResourceInput,
) (*cloudfront.ListTagsForResourceOutput, error) {
	return api.ListTagsForResource(c, input)
}

func CreateDistribution(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.CreateDistributionWithTagsInput,
) (*cloudfront.CreateDistributionWithTagsOutput, error) {
	return api.CreateDistributionWithTags(c, input)
}

func CreateOrigin(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.CreateOriginRequestPolicyInput,
) (*cloudfront.CreateOriginRequestPolicyOutput, error) {
	return api.CreateOriginRequestPolicy(c, input)
}

func DeleteOrigin(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.DeleteOriginRequestPolicyInput,
) (*cloudfront.DeleteOriginRequestPolicyOutput, error) {
	return api.DeleteOriginRequestPolicy(c, input)
}

// This function will create the configuration needed to create a new origin
func FormatOrigin(bucket string, region string, root string) types.Origin {
	var http int32 = 80
	var https int32 = 443
	var keepAlive int32 = 5
	var readTimeout int32 = 30

	config := &types.CustomOriginConfig{
		HTTPPort:               &http,
		HTTPSPort:              &https,
		OriginKeepaliveTimeout: &keepAlive,
		OriginProtocolPolicy:   types.OriginProtocolPolicyHttpOnly,
		OriginReadTimeout:      &readTimeout,
	}

	var connAttempts int32 = 3
	var connTimeout int32 = 10
	var domainName string = fmt.Sprintf("%v.s3-website.%v.amazonaws.com", bucket, region)
	var originId string = fmt.Sprintf("pilot-origin-%v", bucket)
	var originPath string = root
	origin := types.Origin{
		ConnectionAttempts: &connAttempts,
		ConnectionTimeout:  &connTimeout,
		CustomOriginConfig: config,
		DomainName:         &domainName,
		Id:                 &originId,
		OriginPath:         &originPath,
	}

	return origin
}

// This function will create the configuration input needed to create a new distribution
func FormatDistributionInput(bucket string, region string, root string) *cloudfront.CreateDistributionWithTagsInput {
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

	input := &cloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &types.DistributionConfigWithTags{
			DistributionConfig: &types.DistributionConfig{
				CallerReference: &callRef,
				Comment:         &comment,
				DefaultCacheBehavior: &types.DefaultCacheBehavior{
					TargetOriginId:       origin.Id,
					ViewerProtocolPolicy: types.ViewerProtocolPolicyAllowAll,
					CachePolicyId:        &cachePolicy,
				},
				Enabled: &enabled,
				Origins: &types.Origins{
					Quantity: &quantity,
					Items:    [](types.Origin){origin},
				},
			},
			Tags: &types.Tags{
				Items: items,
			},
		},
	}

	return input
}
