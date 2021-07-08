package platform

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

// S3BucketAPI defines the interface for the CreateBucket function.
type S3BucketAPI interface {
	CreateBucket(ctx context.Context,
		params *s3.CreateBucketInput,
		optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)

	PutBucketPolicy(ctx context.Context,
		params *s3.PutBucketPolicyInput,
		optFns ...func(*s3.Options)) (*s3.PutBucketPolicyOutput, error)
}

// MakeBucket creates an Amazon S3 bucket.
// Inputs:
//     c is the context of the method call, which includes the AWS Region
//     api is the interface that defines the method call
//     input defines the input arguments to the service call.
// Output:
//     If success, a CreateBucketOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to CreateBucket.
func MakeBucket(c context.Context, api S3BucketAPI, input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	return api.CreateBucket(c, input)
}

func SetPublicBucketPolicy(c context.Context, api S3BucketAPI, input *s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error) {
	return api.PutBucketPolicy(c, input)
}

type PlatformConfig struct {
	// AWS region to operate in
	Region string `hcl:"region"`

	// Name of S3 bucket to create
	BucketName string `hcl:"bucket"`
}

type Platform struct {
	config PlatformConfig
}

// Implement Configurable
func (p *Platform) Config() (interface{}, error) {
	return &p.config, nil
}

// Implement ConfigurableNotify
func (p *Platform) ConfigSet(config interface{}) error {
	c, ok := config.(*PlatformConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("expected *PlatformConfig as parameter")
	}

	_, err := os.Stat("./build")

	// validate the config
	if err != nil {
		return fmt.Errorf("no build directory exists")
	}

	if c.Region == "" {
		return fmt.Errorf("region must be specified")
	}

	if c.BucketName == "" {
		return fmt.Errorf("bucket name must be specified")
	}

	return nil
}

// Implement Platform
func (p *Platform) DeployFunc() interface{} {
	// return a function which will be called by Waypoint
	return p.deploy
}

func (p *Platform) deploy(ctx context.Context, ui terminal.UI) (*Deployment, error) {
	u := ui.Status()
	defer u.Close()
	u.Update("Deploy S3 assets")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	cBInput := &s3.CreateBucketInput{
		Bucket: &p.config.BucketName,
	}

	u.Step(terminal.StatusOK, "Creating bucket "+p.config.BucketName)
	_, err = MakeBucket(context.TODO(), client, cBInput)
	if err != nil {
		u.Step(terminal.StatusError, "Could not create bucket "+p.config.BucketName)
		return nil, err
	}

	u.Step(terminal.StatusOK, "Bucket created successfully")

	u.Step(terminal.StatusOK, "Setting bucket permissions")

	policy := getPolicy(p.config.BucketName)

	pBPInput := &s3.PutBucketPolicyInput{
		Bucket: &p.config.BucketName,
		Policy: &policy,
	}

	_, err = SetPublicBucketPolicy(context.TODO(), client, pBPInput)
	if err != nil {
		u.Step(terminal.StatusError, "Could not set bucket policy")
		return nil, err
	}

	return &Deployment{}, nil
}

func getPolicy(b string) string {
	return fmt.Sprintf(`{
		"Version":"2012-10-17",
		"Statement":[
			{
				"Sid":"PublicRead",
				"Effect":"Allow",
				"Principal": "*",
				"Action":["s3:GetObject","s3:GetObjectVersion"],
				"Resource":["arn:aws:s3:::%s/*"]
			}
		]
	}`, b)
}
