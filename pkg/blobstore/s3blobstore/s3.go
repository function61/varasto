// Writes your blobs to AWS S3
package s3blobstore

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/function61/gokit/aws/s3facade"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
	"io"
	"io/ioutil"
	"log"
	"regexp"
)

type s3blobstore struct {
	bucket string
	client *s3.S3
	logl   *logex.Leveled
}

func New(opts string, logger *log.Logger) (*s3blobstore, error) {
	bucket, regionId, accessKeyId, secret, err := parseOptionsString(opts)
	if err != nil {
		return nil, err
	}

	client, err := s3facade.Client(accessKeyId, secret, regionId)
	if err != nil {
		return nil, err
	}

	return &s3blobstore{
		bucket: bucket,
		client: client,
		logl:   logex.Levels(logger),
	}, nil
}

func (g *s3blobstore) RawFetch(ctx context.Context, ref stotypes.BlobRef) (io.ReadCloser, error) {
	res, err := g.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &g.bucket,
		Key:    aws.String(toS3BlobstoreName(ref)),
	})
	if err != nil {
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

	if _, err := g.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: &g.bucket,
		Key:    aws.String(toS3BlobstoreName(ref)),
		Body:   bytes.NewReader(buf),
	}); err != nil {
		return fmt.Errorf("s3 PutObject: %v", err)
	}

	return nil
}

func (g *s3blobstore) Mountable(ctx context.Context) error {
	_, err := g.client.ListObjectsWithContext(ctx, &s3.ListObjectsInput{
		Bucket:  &g.bucket,
		MaxKeys: aws.Int64(1), // we'll just want to see that the access key works
	})
	return err
}

func (s *s3blobstore) RoutingCost() int {
	return 20
}

func toS3BlobstoreName(ref stotypes.BlobRef) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}

var parseOptionsStringRe = regexp.MustCompile("^([^:]+):([^:]+):([^:]+):([^:]+)$")

func parseOptionsString(serialized string) (string, string, string, string, error) {
	match := parseOptionsStringRe.FindStringSubmatch(serialized)
	if match == nil {
		return "", "", "", "", errors.New("s3 options not in format bucket:region:accessKeyId:secret")
	}

	return match[1], match[2], match[3], match[4], nil
}
