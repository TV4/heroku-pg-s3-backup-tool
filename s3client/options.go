package s3client

import "time"

// WithHTTPTimeout is an option to set a custom HTTP timeout when creating a new
// S3Client.
func WithHTTPTimeout(d time.Duration) func(*S3Client) {
	return func(c *S3Client) {
		c.httpTimeout = d
	}
}
