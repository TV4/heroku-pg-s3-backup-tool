package s3client

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsclient "github.com/aws/aws-sdk-go/aws/client"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Client is a client for interacting with an S3 bucket.
type S3Client struct {
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsRegion          string

	bucket      string
	httpTimeout time.Duration

	awsConfProviderMu sync.Mutex
	awsConfProvider   awsclient.ConfigProvider
}

// New creates a new S3 client.
func New(awsAccessKeyID, awsSecretAccessKey, awsRegion, bucket string, opts ...func(*S3Client)) *S3Client {
	c := &S3Client{
		awsAccessKeyID:     awsAccessKeyID,
		awsSecretAccessKey: awsSecretAccessKey,
		awsRegion:          awsRegion,
		bucket:             bucket,
		httpTimeout:        10 * time.Minute,
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

// Upload reads data from src and uploads it to the given S3 object key (path).
func (c *S3Client) Upload(ctx context.Context, objectKey string, src io.Reader) error {
	cfgProvider, err := c.configProvider()
	if err != nil {
		return err
	}

	uploadInput := &s3manager.UploadInput{
		Body:   src,
		Bucket: &c.bucket,
		Key:    &objectKey,
	}

	uploader := s3manager.NewUploader(cfgProvider)

	if _, err := uploader.UploadWithContext(ctx, uploadInput); err != nil {
		return err
	}

	return nil
}

func (c *S3Client) configProvider() (awsclient.ConfigProvider, error) {
	var err error

	if c.awsAccessKeyID == "" && c.awsSecretAccessKey == "" {
		c.awsConfProvider, err = awssession.NewSession(
			&aws.Config{
				Region: &c.awsRegion,
				HTTPClient: &http.Client{
					Timeout: c.httpTimeout,
				},
			},
		)

		return c.awsConfProvider, err
	}

	c.awsConfProvider, err = awssession.NewSession(
		&aws.Config{
			Credentials: awscredentials.NewStaticCredentialsFromCreds(awscredentials.Value{
				AccessKeyID:     c.awsAccessKeyID,
				SecretAccessKey: c.awsSecretAccessKey,
			}),
			Region: &c.awsRegion,
			HTTPClient: &http.Client{
				Timeout: c.httpTimeout,
			},
		},
	)

	return c.awsConfProvider, err
}
