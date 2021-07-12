package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gabriel-vasile/mimetype"
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

	PutBucketWebsite(ctx context.Context,
		params *s3.PutBucketWebsiteInput,
		optFns ...func(*s3.Options)) (*s3.PutBucketWebsiteOutput, error)

	PutObject(ctx context.Context,
		params *s3.PutObjectInput,
		optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
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

func EnableWebHosting(c context.Context, api S3BucketAPI, input *s3.PutBucketWebsiteInput) (*s3.PutBucketWebsiteOutput, error) {
	return api.PutBucketWebsite(c, input)
}

func AddFile(c context.Context, api S3BucketAPI, input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return api.PutObject(c, input)
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

// TODO: Check for existing bucket and if exists don't error
func CreateBucket(p *Platform, client *s3.Client) error {
	input := &s3.CreateBucketInput{
		Bucket: &p.config.BucketName,
	}

	if p.config.Region != "us-east-1" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(p.config.Region),
		}
	}

	_, err := MakeBucket(context.TODO(), client, input)
	return err
}

func PutBucketPolicy(b string, client *s3.Client) error {
	input := &s3.PutBucketPolicyInput{
		Bucket: &b,
		Policy: aws.String(getPolicy(b)),
	}

	_, err := SetPublicBucketPolicy(context.TODO(), client, input)
	return err
}

func PutBucketWebsite(b string, client *s3.Client) error {
	input := &s3.PutBucketWebsiteInput{
		Bucket: &b,
		WebsiteConfiguration: &types.WebsiteConfiguration{
			IndexDocument: &types.IndexDocument{Suffix: aws.String("index.html")},
		},
	}

	_, err := EnableWebHosting(context.TODO(), client, input)
	return err
}

// PutObjects recursively checks for files in build path and uploads
// to specified s3 bucket. The errors slice keeps track of errors found during upload.
func PutObjects(b, subPath string, client *s3.Client, errors *[]string) []string {
	files, err := os.ReadDir("./build/" + subPath)
	if err != nil {
		*errors = append(*errors, err.Error())
	}

	for _, file := range files {
		if file.IsDir() {
			PutObjects(b, subPath+file.Name()+"/", client, errors)
			continue
		}

		f, err := os.Open("./build/" + subPath + file.Name())
		if err != nil {
			*errors = append(*errors, err.Error())
			continue
		}

		defer f.Close()

		// get file size and read file contents into buffer
		fileInfo, _ := f.Stat()
		size := fileInfo.Size()
		buffer := make([]byte, size)
		f.Read(buffer)

		input := &s3.PutObjectInput{
			Bucket:      &b,
			Key:         aws.String(subPath + fileInfo.Name()),
			Body:        bytes.NewReader(buffer),
			ContentType: aws.String(DetectMimeType(fileInfo.Name(), buffer)),
		}

		_, err = AddFile(context.TODO(), client, input)
		if err != nil {
			*errors = append(*errors, err.Error())
			continue
		}
	}

	return *errors
}

func (p *Platform) deploy(ctx context.Context, ui terminal.UI) (*Deployment, error) {
	u := ui.Status()
	defer u.Close()
	u.Step("", "---Deploying S3 assets---")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(p.config.Region))
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	u.Step("", "Creating bucket "+p.config.BucketName)
	err = CreateBucket(p, client)
	if err != nil {
		u.Step(terminal.StatusError, "Could not create bucket "+p.config.BucketName)
		return nil, err
	}

	u.Step(terminal.StatusOK, "Bucket created successfully")
	u.Step("", "Setting bucket permissions")

	err = PutBucketPolicy(p.config.BucketName, client)
	if err != nil {
		u.Step(terminal.StatusError, "Could not set bucket policy")
		return nil, err
	}

	u.Step(terminal.StatusOK, "Bucket policy created")
	u.Step("", "Enabling static website hosting")

	err = PutBucketWebsite(p.config.BucketName, client)
	if err != nil {
		u.Step(terminal.StatusError, "Could not enable static web hosting")
		return nil, err
	}

	u.Step(terminal.StatusOK, "Static website hosting enabled")
	u.Step("", "Pushing static files")

	fileErrors := []string{}
	PutObjects(p.config.BucketName, "", client, &fileErrors)
	if len(fileErrors) > 0 {
		u.Step(terminal.StatusWarn, "Some static files failed to upload")
	}

	u.Step(terminal.StatusOK, "Upload of static files complete")

	return &Deployment{
		Bucket: p.config.BucketName,
		Region: p.config.Region,
	}, nil
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

func DetectMimeType(fname string, buffer []byte) string {
	if strings.HasSuffix(fname, ".css") {
		return "text/css"
	} else if strings.HasSuffix(fname, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(fname, ".map") {
		return "binary/octet-stream"
	}
	return mimetype.Detect(buffer).String()
}
