package platform

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
)

// Implement the Destroyer interface
func (p *Platform) DestroyFunc() interface{} {
	return p.destroy
}

func EmptyBucket(c context.Context, client *s3.Client, bucket string) error {
	items, err := ListItems(c, client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})

	if err != nil {
		return err
	}

	for _, item := range items.Contents {
		_, err := DeleteItem(c, client, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    item.Key,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (p *Platform) destroy(ctx context.Context, ui terminal.UI, deployment *Deployment) error {
	u := ui.Status()
	defer u.Close()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(p.config.Region))
	if err != nil {
		u.Step(terminal.StatusError, "AWS configuration error, "+err.Error())
		return err
	}

	client := s3.NewFromConfig(cfg)

	u.Update("Deleting objects...")

	err = EmptyBucket(ctx, client, p.config.BucketName)
	if err != nil {
		return err
	}

	u.Update("Deleting bucket...")

	_, err = DeleteBucket(ctx, client, &s3.DeleteBucketInput{
		Bucket: aws.String(p.config.BucketName),
	})
	if err != nil {
		return err
	}

	u.Step(terminal.StatusOK, fmt.Sprintf("Deleted S3 bucket %v", p.config.BucketName))

	return nil
}
