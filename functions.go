// Package indexer provides GCS indexer functions
package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavdahiya/openshift-ci-namespace-indexer/prow"
	"google.golang.org/api/option"

	"cloud.google.com/go/storage"
)

// GCSEvent is the payload of a GCS event.
type GCSEvent struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	// SelfLink                string                 `json:"selfLink"`
	Name   string `json:"name"`
	Bucket string `json:"bucket"`
	// Generation              string                 `json:"generation"`
	// Metageneration          string                 `json:"metageneration"`
	ContentType string    `json:"contentType"`
	TimeCreated time.Time `json:"timeCreated"`
	Updated     time.Time `json:"updated"`
	// TemporaryHold           bool                   `json:"temporaryHold"`
	// EventBasedHold          bool                   `json:"eventBasedHold"`
	// RetentionExpirationTime time.Time              `json:"retentionExpirationTime"`
	// StorageClass            string                 `json:"storageClass"`
	// TimeStorageClassUpdated time.Time              `json:"timeStorageClassUpdated"`
	Size      string `json:"size"`
	MD5Hash   string `json:"md5Hash"`
	MediaLink string `json:"mediaLink"`
	// ContentEncoding         string                 `json:"contentEncoding"`
	// ContentDisposition      string                 `json:"contentDisposition"`
	// CacheControl            string                 `json:"cacheControl"`
	Metadata map[string]interface{} `json:"metadata"`
	// CRC32C                  string                 `json:"crc32c"`
	// ComponentCount          int                    `json:"componentCount"`
	// Etag                    string                 `json:"etag"`
	// CustomerEncryption      struct {
	// 	EncryptionAlgorithm string `json:"encryptionAlgorithm"`
	// 	KeySha256           string `json:"keySha256"`
	// }
	// KMSKeyName    string `json:"kmsKeyName"`
	// ResourceState string `json:"resourceState"`
}

// IndexJobsByNamespace creates a index of all jobs within
// a bucket by the work namespace for the jobs. Jobs that have completed are linked from
//
//   gs://BUCKET/index/job-namespace/JOB_NAME/BUILD_NUMBER
//
// with the contents of that file a job result object and a 'link'
// metadata attribute pointing to a gs:// path to the source. The
// 'state' metadata attribute is set to 'success', 'failure', or
// 'error' if the passed attribute is not set in the finished.json.
//
// Readers should not assume anything about the contents of the
// object or that the link is in the same bucket.
func IndexJobsByNamespace(ctx context.Context, e GCSEvent) error {
	base := path.Base(e.Name)
	switch base {
	case "finished.json":
		parts := strings.Split(e.Name, "/")
		if len(parts) < 4 {
			return nil
		}
		client, err := storage.NewClient(ctx, option.WithScopes(storage.ScopeReadWrite))
		if err != nil {
			return err
		}
		r, err := client.Bucket(e.Bucket).Object(e.Name).NewReader(ctx)
		if err != nil {
			return err
		}
		data, err := ioutil.ReadAll(r)
		var finished prow.Finished
		if err := json.Unmarshal(data, &finished); err != nil {
			return err
		}
		if finished.Timestamp == nil || *finished.Timestamp == 0 {
			return nil
		}

		var state string
		switch {
		case finished.Passed == nil:
			state = "error"
		case *finished.Passed:
			state = "success"
		default:
			state = "failed"
		}

		namespace := "unknown"
		if n, ok := finished.Metadata.Strings()["work-namespace"]; ok {
			namespace = n
		}

		// build index components
		build := parts[len(parts)-2]
		job := parts[len(parts)-3]
		finishedAt := time.Unix(*finished.Timestamp, 0)
		key := namespace
		u := (&url.URL{
			Scheme: "gs",
			Host:   e.Bucket,
			Path:   path.Dir(e.Name),
		}).String()
		indexPath := path.Join("index", "job-namespace", key, job, build)

		// set the data for the job to the result
		if data, err = json.Marshal(JobResult{
			State:       state,
			Namespace:   namespace,
			CompletedAt: finishedAt.Unix(),
			Link:        u,
		}); err != nil {
			return fmt.Errorf("could not serialize job result: %v", err)
		}

		// write the link with the metadata contents
		w := client.
			Bucket(e.Bucket).
			Object(indexPath).
			If(storage.Conditions{DoesNotExist: true}).
			NewWriter(ctx)
		w.ObjectAttrs.Metadata = map[string]string{
			"link":      u,
			"state":     state,
			"namespace": namespace,
			"completed": strconv.FormatInt(finishedAt.Unix(), 10),
		}
		if _, err := w.Write(data); err != nil {
			defer w.Close()
			return fmt.Errorf("failed to write %s to %s: %v", indexPath, u, err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("failed to close %s to %s: %v", indexPath, u, err)
		}
		log.Printf("Indexed job %s with state %s to gs://%s/%s", u, state, e.Bucket, indexPath)
	}
	return nil
}

// JobResult stores the result of the job
type JobResult struct {
	State       string `json:"state"`
	Namespace   string `json:"namespace"`
	CompletedAt int64  `json:"completed_at"`
	Link        string `json:"link"`
}
