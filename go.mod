module github.com/pilot-framework/aws-cloudfront-waypoint-plugin

go 1.14

require (
	github.com/aws/aws-sdk-go-v2 v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.4.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.11.0 // indirect
	github.com/go-sql-driver/mysql v1.4.0 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/hashicorp/waypoint-plugin-sdk v0.0.0-20201021094150-1b1044b1478e
	github.com/mitchellh/go-glint v0.0.0-20201015034436-f80573c636de
	google.golang.org/protobuf v1.25.0
)

// replace github.com/hashicorp/waypoint-plugin-sdk => ../../waypoint-plugin-sdk
