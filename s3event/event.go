// Copyright 2023 Versity Software
// This file is licensed under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package s3event

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/versity/versitygw/auth"
)

type S3EventSender interface {
	SendEvent(ctx *fiber.Ctx, meta EventMeta)
	Close() error
}

type EventMeta struct {
	BucketOwner string
	EventName   EventType
	ObjectSize  int64
	ObjectETag  *string
	VersionID   *string
}

type EventSchema struct {
	Records []EventRecord
}

type EventRecord struct {
	EventVersion      string                `json:"eventVersion"`
	EventSource       string                `json:"eventSource"`
	AwsRegion         string                `json:"awsRegion"`
	EventTime         string                `json:"eventTime"`
	EventName         EventType             `json:"eventName"`
	UserIdentity      EventUserIdentity     `json:"userIdentity"`
	RequestParameters EventRequestParams    `json:"requestParameters"`
	ResponseElements  EventResponseElements `json:"responseElements"`
	S3                EventS3Data           `json:"s3"`
	GlacierEventData  EventGlacierData      `json:"glacierEventData"`
}

type EventUserIdentity struct {
	PrincipalID string `json:"PrincipalId"`
}

type EventRequestParams struct {
	SourceIPAddress string `json:"sourceIPAddress"`
}

type EventResponseElements struct {
	RequestID string `json:"x-amz-request-id"`
	HostID    string `json:"x-amz-id-2"`
}

type ConfigurationID string

// This field will be changed after implementing per bucket notifications
const (
	ConfigurationIDKafka   ConfigurationID = "kafka-global"
	ConfigurationIDNats    ConfigurationID = "nats-global"
	ConfigurationIDWebhook ConfigurationID = "webhook-global"
)

type EventS3Data struct {
	S3SchemaVersion string            `json:"s3SchemaVersion"`
	ConfigurationID ConfigurationID   `json:"configurationId"`
	Bucket          EventS3BucketData `json:"bucket"`
	Object          EventObjectData   `json:"object"`
}

type EventGlacierData struct {
	RestoreEventData EventRestoreData `json:"restoreEventData"`
}

type EventRestoreData struct {
	LifecycleRestorationExpiryTime string `json:"lifecycleRestorationExpiryTime"`
	LifecycleRestoreStorageClass   string `json:"lifecycleRestoreStorageClass"`
}

type EventS3BucketData struct {
	Name          string            `json:"name"`
	OwnerIdentity EventUserIdentity `json:"ownerIdentity"`
	Arn           string            `json:"arn"`
}

type EventObjectData struct {
	Key       string  `json:"key"`
	Size      int64   `json:"size"`
	ETag      *string `json:"eTag"`
	VersionID *string `json:"versionId"`
	Sequencer string  `json:"sequencer"`
}

type EventConfig struct {
	KafkaURL             string
	KafkaTopic           string
	KafkaTopicKey        string
	NatsURL              string
	NatsTopic            string
	WebhookURL           string
	FilterConfigFilePath string
}

func InitEventSender(cfg *EventConfig) (S3EventSender, error) {
	filter, err := parseEventFiltersFile(cfg.FilterConfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("parse event filter config file %w", err)
	}
	var evSender S3EventSender
	switch {
	case cfg.WebhookURL != "":
		evSender, err = InitWebhookEventSender(cfg.WebhookURL, filter)
		fmt.Printf("initializing S3 Event Notifications with webhook URL %v\n", cfg.WebhookURL)
	case cfg.KafkaURL != "":
		evSender, err = InitKafkaEventService(cfg.KafkaURL, cfg.KafkaTopic, cfg.KafkaTopicKey, filter)
		fmt.Printf("initializing S3 Event Notifications with kafka. URL: %v, topic: %v\n", cfg.WebhookURL, cfg.KafkaTopic)
	case cfg.NatsURL != "":
		evSender, err = InitNatsEventService(cfg.NatsURL, cfg.NatsTopic, filter)
		fmt.Printf("initializing S3 Event Notifications with Nats. URL: %v, topic: %v\n", cfg.NatsURL, cfg.NatsTopic)
	default:
		return nil, nil
	}

	return evSender, err
}

func createEventSchema(ctx *fiber.Ctx, meta EventMeta, configID ConfigurationID) EventSchema {
	path := strings.Split(ctx.Path(), "/")
	bucket, object := path[1], strings.Join(path[2:], "/")
	acc := ctx.Locals("account").(auth.Account)

	return EventSchema{
		Records: []EventRecord{
			{
				EventVersion: "2.2",
				EventSource:  "aws:s3",
				AwsRegion:    ctx.Locals("region").(string),
				EventTime:    time.Now().Format(time.RFC3339),
				EventName:    meta.EventName,
				UserIdentity: EventUserIdentity{
					PrincipalID: acc.Access,
				},
				RequestParameters: EventRequestParams{
					SourceIPAddress: ctx.IP(),
				},
				ResponseElements: EventResponseElements{
					RequestID: ctx.Get("X-Amz-Request-Id"),
					HostID:    ctx.Get("X-Amz-Id-2"),
				},
				S3: EventS3Data{
					S3SchemaVersion: "1.0",
					ConfigurationID: configID,
					Bucket: EventS3BucketData{
						Name: bucket,
						OwnerIdentity: EventUserIdentity{
							PrincipalID: meta.BucketOwner,
						},
						Arn: fmt.Sprintf("arn:aws:s3:::%v", strings.Join(path, "/")),
					},
					Object: EventObjectData{
						Key:       object,
						Size:      meta.ObjectSize,
						ETag:      meta.ObjectETag,
						VersionID: meta.VersionID,
						Sequencer: genSequencer(),
					},
				},
				GlacierEventData: EventGlacierData{
					// Not supported
					RestoreEventData: EventRestoreData{},
				},
			},
		},
	}
}

func generateTestEvent() ([]byte, error) {
	msg := map[string]string{
		"Service": "S3",
		"Event":   "s3:TestEvent",
		"Time":    time.Now().Format(time.RFC3339),
		"Bucket":  "Test-Bucket",
	}

	return json.Marshal(msg)
}
