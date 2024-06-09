// Writes your blobs to AWS S3
package s3blobstore

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
)

type BucketContext struct {
	Name *string
	S3   *s3.S3
}

type s3blobstore struct {
	blobNamer *s3BlobNamer
	bucket    *BucketContext
	logl      *logex.Leveled
}

func New(opts string, logger *log.Logger) (*s3blobstore, error) {
	conf, err := deserializeConfig(opts)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(conf.Prefix, "/") {
		return nil, fmt.Errorf("prefix needs to end in '/'; got '%s'", conf.Prefix)
	}

	config := aws.NewConfig().WithRegion(conf.RegionId).WithCredentials(credentials.NewStaticCredentials(conf.AccessKeyId, conf.AccessKeySecret, ""))
	if conf.Endpoint != "" {
		config = config.WithEndpoint(conf.Endpoint)
	}

	awsSession, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	bucket := &BucketContext{
		Name: &conf.Bucket,
		S3: s3.New(
			awsSession,
			config),
	}

	return &s3blobstore{
		blobNamer: &s3BlobNamer{conf.Prefix},
		bucket:    bucket,
		logl:      logex.Levels(logger),
	}, nil
}

func (g *s3blobstore) RawFetch(ctx context.Context, ref stotypes.BlobRef) (io.ReadCloser, error) {
	res, err := g.bucket.S3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: g.bucket.Name,
		Key:    g.blobNamer.Ref(ref),
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == s3.ErrCodeNoSuchKey {
			return nil, os.ErrNotExist
		}

		return nil, fmt.Errorf("s3 GetObject: %v", err)
	}

	return res.Body, nil
}

func (g *s3blobstore) RawStore(ctx context.Context, ref stotypes.BlobRef, content io.Reader) error {
	// since S3 internally requires retry support, it requires a io.ReadSeeker and thus
	// we're forced to buffer
	buf, err := ioutil.ReadAll(content)
	if err != nil {
		return err
	}

	if _, err := g.bucket.S3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: g.bucket.Name,
		Key:    g.blobNamer.Ref(ref),
		Body:   bytes.NewReader(buf),
	}); err != nil {
		return fmt.Errorf("s3 PutObject: %v", err)
	}

	return nil
}

func (s *s3blobstore) RoutingCost() int {
	return 20
}

type s3BlobNamer struct {
	prefix string
}

func (s *s3BlobNamer) Ref(ref stotypes.BlobRef) *string {
	return aws.String(s.prefix + base64.RawURLEncoding.EncodeToString([]byte(ref)))
}

type Config struct {
	Bucket          string
	Prefix          string
	RegionId        string
	AccessKeyId     string
	AccessKeySecret string
	Endpoint        string
}

func (c *Config) Serialize() string {
	return strings.Join([]string{
		c.Bucket,
		c.Prefix,
		c.AccessKeyId,
		c.AccessKeySecret,
		c.RegionId,
		c.Endpoint,
	}, ":")
}

func deserializeConfig(serialized string) (*Config, error) {
	match := strings.Split(serialized, ":")

	// endpoint was added later, so parse it conditionally
	endpoint, err := func() (string, error) {
		switch len(match) {
		case 5:
			return "", nil
		case 6:
			return match[5], nil
		default:
			return "", errors.New("s3 options not in format bucket:prefix:accessKeyId:secret:region[:endpoint]")
		}
	}()
	if err != nil {
		return nil, err
	}

	return &Config{
		Bucket:          match[0],
		Prefix:          match[1],
		AccessKeyId:     match[2],
		AccessKeySecret: match[3],
		RegionId:        match[4],
		Endpoint:        endpoint,
	}, nil
}
