module github.com/pilot-framework/aws-cloudfront-waypoint-plugin

go 1.14

require (
	github.com/aws/aws-sdk-go-v2 v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.4.1
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.11.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/hashicorp/waypoint-plugin-sdk v0.0.0-20210625180209-eda7ae600c2d
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0 // indirect
	google.golang.org/protobuf v1.27.1
)

// replace github.com/hashicorp/waypoint-plugin-sdk => ../../waypoint-plugin-sdk
