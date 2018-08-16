/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/pkg/policy"
	"github.com/minio/minio/pkg/event"
	"github.com/minio/minio/pkg/hash"
	"github.com/minio/minio/pkg/madmin"
)

// ObjectLayer implements primitives for object API layer.
type ObjectLayer interface {
	// Storage operations.
	Shutdown(context.Context) error
	StorageInfo(context.Context) StorageInfo

	// Bucket operations.
	MakeBucketWithLocation(ctx context.Context, bucket string, location string) error
	GetBucketInfo(ctx context.Context, bucket string) (bucketInfo BucketInfo, err error)
	ListBuckets(ctx context.Context) (buckets []BucketInfo, err error)
	DeleteBucket(ctx context.Context, bucket string) error
	ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error)
	ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error)
	GetBucketOwner(ctx context.Context, bucket string) (owner string, err error)

	// Object operations.
	GetObject(ctx context.Context, bucket, object string, startOffset int64, length int64, writer io.Writer, etag string) (err error)
	GetObjectInfo(ctx context.Context, bucket, object string) (objInfo ObjectInfo, err error)
	PutObject(ctx context.Context, bucket, object string, data *hash.Reader, metadata map[string]string) (objInfo ObjectInfo, err error)
	CopyObject(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, srcInfo ObjectInfo) (objInfo ObjectInfo, err error)
	DeleteObject(ctx context.Context, bucket, object string) error

	// Multipart operations.
	ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error)
	NewMultipartUpload(ctx context.Context, bucket, object string, metadata map[string]string) (uploadID string, err error)
	CopyObjectPart(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int,
		startOffset int64, length int64, srcInfo ObjectInfo) (info PartInfo, err error)
	PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data *hash.Reader) (info PartInfo, err error)
	ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int, maxParts int) (result ListPartsInfo, err error)
	AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string) error
	CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, uploadedParts []CompletePart) (objInfo ObjectInfo, err error)

	// Healing operations.
	ReloadFormat(ctx context.Context, dryRun bool) error
	HealFormat(ctx context.Context, dryRun bool) (madmin.HealResultItem, error)
	HealBucket(ctx context.Context, bucket string, dryRun bool) ([]madmin.HealResultItem, error)
	HealObject(ctx context.Context, bucket, object string, dryRun bool) (madmin.HealResultItem, error)
	ListBucketsHeal(ctx context.Context) (buckets []BucketInfo, err error)
	ListObjectsHeal(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (ListObjectsInfo, error)

	// Locking operations
	ListLocks(ctx context.Context, bucket, prefix string, duration time.Duration) ([]VolumeLockInfo, error)
	ClearLocks(context.Context, []VolumeLockInfo) error

	// Policy operations
	SetBucketPolicy(context.Context, string, policy.BucketAccessPolicy) error
	GetBucketPolicy(context.Context, string) (policy.BucketAccessPolicy, error)
	RefreshBucketPolicy(context.Context, string) error
	DeleteBucketPolicy(context.Context, string) error

	// Bucket notification operations
	GetBucketNotification(context.Context, string) (*event.Config, error)
	PutBucketNotification(context.Context, string, *event.Config) error

	// Supported operations check
	IsNotificationSupported() bool
	IsEncryptionSupported() bool
}
