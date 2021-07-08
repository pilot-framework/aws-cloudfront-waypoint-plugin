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

type PlatformConfig struct {
	// AWS region to operate in
	Region string "hcl:region"

	// Name of S3 bucket to create
	BucketName string "hcl:bucket"
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

	input := &s3.CreateBucketInput{
		Bucket: &p.config.BucketName,
	}

	u.Step(terminal.StatusOK, "Creating bucket "+p.config.BucketName)
	_, err = MakeBucket(context.TODO(), client, input)
	if err != nil {
		u.Step(terminal.StatusError, "Could not create bucket "+p.config.BucketName)
	}

	u.Step(terminal.StatusOK, "Bucket created successfully")

	return &Deployment{}, nil
}
