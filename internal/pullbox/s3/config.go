package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/devbox/devopt"
)

// TODO(landau): We could make these customizable so folks can use their own
// buckets and roles. Would require removing user from this lib.
const (
	roleArn = "arn:aws:iam::984256416385:role/JetpackS3Federated"
	bucket  = "devbox.sh"
	// this is a fixed value the bucket resides in this region, otherwise,
	// user's default region will get pulled from config and region mismatch
	// will result in user not being able to run global push
	region = "us-east-2"
)

func assumeRole(ctx context.Context, c *devopt.Credentials) (*aws.Config, error) {
	noPermsConfig, _ := config.LoadDefaultConfig(ctx)
	stsClient := sts.NewFromConfig(noPermsConfig)
	creds, err := stsClient.AssumeRoleWithWebIdentity(
		ctx,
		&sts.AssumeRoleWithWebIdentityInput{
			RoleArn:          aws.String(roleArn),
			RoleSessionName:  aws.String(c.Email),
			WebIdentityToken: aws.String(c.IDToken),
		},
	)
	if err != nil {
		return nil, err
	}

	config, err := config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				*creds.Credentials.AccessKeyId,
				*creds.Credentials.SecretAccessKey,
				*creds.Credentials.SessionToken,
			),
		),
	)
	config.Region = region
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &config, err
}
