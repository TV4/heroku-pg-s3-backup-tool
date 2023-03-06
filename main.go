// Deprecated: This tool is no longer maintained.
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/TV4/env"
	"github.com/TV4/heroku-pg-s3-backup-tool/s3client"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

var conf *config

func main() {
	fmt.Println("heroku-pg-s3-backup-tool")
	fmt.Println()
	conf = readConfiguration()
	fmt.Fprintln(os.Stderr, conf)

	if err := conf.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	log.Println("Fetching database snapshot download URL")

	url, err := getDownloadURL(fmt.Sprintf("https://postgres-api.heroku.com/client/v11/apps/%s/transfers", conf.herokuAppName), conf.herokuAPIToken)
	if err != nil {
		log.Fatalf("error fetching download URL: %v", err)
	}

	log.Println("Backing up database to S3 bucket")

	if err := transferToS3(url); err != nil {
		log.Fatalf("error transferring backup to S3: %v", err)
	}

	log.Println("Done.")
}

func transferToS3(downloadURL string) error {
	objectKey := fmt.Sprintf("%s_%s.gz", conf.herokuAppName, time.Now().Format("2006-01-02_15:04:05"))

	s3c := s3client.New(conf.awsAccessKeyID, conf.awsSecretAccessKey, conf.awsRegion, conf.s3BucketName)

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	defer resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pr, pw := io.Pipe()

	go func() {
		gw := gzip.NewWriter(pw)
		io.Copy(gw, resp.Body)
		gw.Close()
		pw.Close()
	}()

	if err := s3c.Upload(ctx, objectKey, pr); err != nil {
		return err
	}

	log.Printf("count#%s.postgres.backup=1", conf.herokuAppName)
	log.Printf("Uploaded S3 object %s", objectKey)

	return nil
}

func getDownloadURL(apiURL, apiToken string) (string, error) {
	transferNum, err := getLatestTransfer(apiURL, apiToken)
	if err != nil {
		return "", fmt.Errorf("error fetching latest transfer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%d/actions/public-url", apiURL, transferNum), nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("", apiToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error fetching URL: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got HTTP status %d while fetching URL", resp.StatusCode)
	}

	defer resp.Body.Close()

	var result struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	return result.URL, nil
}

func getLatestTransfer(apiURL, apiToken string) (num int, err error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth("", apiToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("got HTTP status %d while fetching latest transfer", resp.StatusCode)
	}

	defer resp.Body.Close()

	var result []struct {
		Num       *int   `json:"num"`
		Succeeded bool   `json:"succeeded"`
		ToName    string `json:"to_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result) < 1 {
		return 0, errors.New("no transfers found")
	}

	var found bool

	for _, r := range result {
		if r.Succeeded && r.Num != nil && (r.ToName == "BACKUP" || r.ToName == "SCHEDULED BACKUP") {
			num = *r.Num
			found = true
			break
		}
	}

	if !found {
		return 0, errors.New("no backup found")
	}

	return num, nil
}

func readConfiguration() *config {
	return &config{
		herokuAppName:      env.String("HEROKU_APP_NAME", ""),
		herokuAPIToken:     env.String("PGBACKUP_HEROKU_API_TOKEN", ""),
		awsAccessKeyID:     env.String("PGBACKUP_AWS_ACCESS_KEY_ID", ""),
		awsSecretAccessKey: env.String("PGBACKUP_AWS_SECRET_ACCESS_KEY", ""),
		awsRegion:          env.String("PGBACKUP_AWS_REGION", ""),
		s3BucketName:       env.String("PGBACKUP_S3_BUCKET_NAME", ""),
	}
}

type config struct {
	herokuAppName      string
	herokuAPIToken     string
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsRegion          string
	s3BucketName       string
}

func (c *config) Validate() error {
	if c.herokuAppName == "" {
		return errors.New("HEROKU_APP_NAME missing")
	}

	if c.herokuAPIToken == "" {
		return errors.New("PGBACKUP_HEROKU_API_TOKEN missing")
	}

	if c.awsAccessKeyID == "" {
		return errors.New("PGBACKUP_AWS_ACCESS_KEY_ID missing")
	}

	if c.awsSecretAccessKey == "" {
		return errors.New("PGBACKUP_AWS_SECRET_ACCESS_KEY missing")
	}

	if c.awsRegion == "" {
		return errors.New("PGBACKUP_AWS_REGION missing")
	}

	if c.s3BucketName == "" {
		return errors.New("PGBACKUP_S3_BUCKET_NAME missing")
	}

	return nil
}

func (c *config) String() string {
	hideIfSet := func(v interface{}) string {
		s := ""

		switch typedV := v.(type) {
		case string:
			s = typedV
		case []string:
			s = strings.Join(typedV, ",")
		case []byte:
			s = string(typedV)
		case fmt.Stringer:
			if typedV != nil {
				s = typedV.String()
			}
		}

		if s != "" {
			return "<hidden>"
		}
		return ""
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 1, 4, ' ', 0)
	for _, e := range []struct {
		k string
		v interface{}
	}{
		{"HEROKU_APP_NAME", c.herokuAppName},
		{"PGBACKUP_HEROKU_API_TOKEN", hideIfSet(c.herokuAPIToken)},
		{"PGBACKUP_AWS_ACCESS_KEY_ID", c.awsAccessKeyID},
		{"PGBACKUP_AWS_SECRET_ACCESS_KEY", hideIfSet(c.awsSecretAccessKey)},
		{"PGBACKUP_AWS_REGION", c.awsRegion},
		{"PGBACKUP_S3_BUCKET_NAME", c.s3BucketName},
	} {
		fmt.Fprintf(w, "%s\t%v\n", e.k, e.v)
	}
	w.Flush()
	return buf.String()
}
