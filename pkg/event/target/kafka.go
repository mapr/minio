/*
 * Minio Cloud Storage, (C) 2018 Minio, Inc.
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

package target

import (
	"encoding/json"
	"net/url"

	"github.com/minio/minio/pkg/event"
	xnet "github.com/minio/minio/pkg/net"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// KafkaArgs - Kafka target arguments.
type KafkaArgs struct {
	Enable  bool        `json:"enable"`
	Brokers []xnet.Host `json:"brokers"`
	Topic   string      `json:"topic"`
}

// KafkaTarget - Kafka target.
type KafkaTarget struct {
	id       event.TargetID
	args     KafkaArgs
	producer *kafka.Producer
}

// ID - returns target ID.
func (target *KafkaTarget) ID() event.TargetID {
	return target.id
}

// Send - sends event to Kafka.
func (target *KafkaTarget) Send(eventData event.Event) error {
	objectName, err := url.QueryUnescape(eventData.S3.Object.Key)
	if err != nil {
		return err
	}
	key := eventData.S3.Bucket.Name + "/" + objectName

	data, err := json.Marshal(event.Log{eventData.EventName, key, []event.Event{eventData}})
	if err != nil {
		return err
	}

	msg := kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &target.args.Topic, Partition: kafka.PartitionAny},
		Key:   []byte(key),
		Value: []byte(data),
	}

	return target.producer.Produce(&msg, nil)
}

// Close - closes underneath kafka connection.
func (target *KafkaTarget) Close() error {
	target.producer.Close()
	return nil
}

// NewKafkaTarget - creates new Kafka target.
func NewKafkaTarget(id string, args KafkaArgs) (*KafkaTarget, error) {
	brokers := []string{}
	for _, broker := range args.Brokers {
		brokers = append(brokers, broker.String())
	}
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokers[0]})
	if err != nil {
		return nil, err
	}

	return &KafkaTarget{
		id:       event.TargetID{id, "kafka"},
		args:     args,
		producer: producer,
	}, nil
}
