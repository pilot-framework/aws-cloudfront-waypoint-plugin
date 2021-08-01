package cfront

import (
	"context"
	"fmt"
	"strings"
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
	GetDistributionConfig(
		ctx context.Context,
		input *cloudfront.GetDistributionConfigInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.GetDistributionConfigOutput, error)
	UpdateDistribution(
		ctx context.Context,
		input *cloudfront.UpdateDistributionInput,
		optFns ...func(*cloudfront.Options),
	) (*cloudfront.UpdateDistributionOutput, error)
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

func GetDistribution(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.GetDistributionInput,
) (*cloudfront.GetDistributionOutput, error) {
	return api.GetDistribution(c, input)
}

func GetAllDistributions(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.ListDistributionsInput,
) (*cloudfront.ListDistributionsOutput, error) {
	return api.ListDistributions(c, input)
}

func GetDistributionConfig(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.GetDistributionConfigInput,
) (*cloudfront.GetDistributionConfigOutput, error) {
	return api.GetDistributionConfig(c, input)
}

func UpdateDistribution(
	c context.Context,
	api CloudfrontAPI,
	input *cloudfront.UpdateDistributionInput,
) (*cloudfront.UpdateDistributionOutput, error) {
	return api.UpdateDistribution(c, input)
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

func RemoveDistribution(id string, client *cloudfront.Client) {
	status, err := PollStatus(id, client)
	if err != nil {
		panic(err)
	}

	distInput := &cloudfront.GetDistributionInput{
		Id: &id,
	}

	dist, err := GetDistribution(context.TODO(), client, distInput)
	if err != nil {
		panic(err)
	}

	delInput := &cloudfront.DeleteDistributionInput{
		Id:      &id,
		IfMatch: dist.ETag,
	}

	if status {
		_, err = DeleteDistribution(context.TODO(), client, delInput)
		if err != nil {
			panic(err)
		}
	}
}

func DisableDistribution(id string, client *cloudfront.Client) error {
	getCfgInput := &cloudfront.GetDistributionConfigInput{
		Id: &id,
	}

	cfg, err := GetDistributionConfig(context.TODO(), client, getCfgInput)
	if err != nil {
		return err
	}

	enabled := false

	cfg.DistributionConfig.Enabled = &enabled

	updateCfgInput := &cloudfront.UpdateDistributionInput{
		DistributionConfig: cfg.DistributionConfig,
		Id:                 &id,
		IfMatch:            cfg.ETag,
	}

	_, err = UpdateDistribution(context.TODO(), client, updateCfgInput)
	if err != nil {
		return err
	}

	return nil
}

func PollStatus(id string, client *cloudfront.Client) (status bool, err error) {
	distInput := &cloudfront.GetDistributionInput{
		Id: &id,
	}

	timedOut := true
	status = false

	// times out after five minutes
	for i := 0; i < 30; i++ {
		dist, getErr := GetDistribution(context.TODO(), client, distInput)
		if getErr != nil {
			err = getErr
			return
		}

		if strings.ToLower(*dist.Distribution.Status) == "deployed" {
			status = true
			return
		}

		time.Sleep(time.Second * 10)
	}

	if timedOut {
		err = fmt.Errorf("operation timed out after 10 minutes")
	}

	return
}
