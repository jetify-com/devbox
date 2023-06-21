package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/auth"
)

// TODO(landau): We could make these customizable so folks can use their own
// buckets and roles. Would require removing user from this lib.
const (
	roleArn = "arn:aws:iam::984256416385:role/JetpackS3Federated"
	bucket  = "devbox.sh"
)

func assumeRole(ctx context.Context, user *auth.User) (*aws.Config, error) {
	noPermsConfig, _ := config.LoadDefaultConfig(ctx)
	stsClient := sts.NewFromConfig(noPermsConfig)
	creds, err := stsClient.AssumeRoleWithWebIdentity(
		ctx,
		&sts.AssumeRoleWithWebIdentityInput{
			RoleArn:          aws.String(roleArn),
			RoleSessionName:  aws.String(user.Email()),
			WebIdentityToken: aws.String(user.IDToken.Raw),
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
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &config, err
}
