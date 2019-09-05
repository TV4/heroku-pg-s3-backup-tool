package s3client

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		c := New("foo-region", "foo-bucket")

		if got, want := c.region, "foo-region"; got != want {
			t.Errorf("c.region = %q, want %q", got, want)
		}

		if got, want := c.bucket, "foo-bucket"; got != want {
			t.Errorf("c.bucket = %q, want %q", got, want)
		}

		if got, want := c.httpTimeout, 30*time.Second; got != want {
			t.Errorf("c.httpTimeout = %s, want %s", got, want)
		}
	})

	t.Run("WithHTTPTimeout", func(t *testing.T) {
		c := New("", "", WithHTTPTimeout(10*time.Second))

		if got, want := c.httpTimeout, 10*time.Second; got != want {
			t.Errorf("c.httpTimeout = %s, want %q", got, want)
		}
	})
}
