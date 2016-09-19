// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elasticsearch

import (
	"net/url"
	"sync"
	"time"

	"encoding/json"

	"github.com/golang/glog"
	esCommon "k8s.io/heapster/common/elasticsearch"
	event_core "k8s.io/heapster/events/core"
	"k8s.io/heapster/metrics/core"
	kube_api "k8s.io/kubernetes/pkg/api"
)

const (
	typeName = "events"
)

// SaveDataFunc is a pluggable function to enforce limits on the object
type SaveDataFunc func(date time.Time, sinkData []interface{}) error

type elasticSearchSink struct {
	esSvc     esCommon.ElasticSearchService
	saveData  SaveDataFunc
	flushData func() error
	sync.RWMutex
}

type EsSinkPoint struct {
	EventValue     interface{}
	EventTimestamp time.Time
	EventTags      map[string]string
}

// Generate point value for event
func getEventValue(event *kube_api.Event) (string, error) {
	// TODO: check whether indenting is required.
	bytes, err := json.MarshalIndent(event, "", " ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func eventToPoint(event *kube_api.Event) (*EsSinkPoint, error) {
	value, err := getEventValue(event)
	if err != nil {
		return nil, err
	}
	point := EsSinkPoint{
		EventTimestamp: event.LastTimestamp.Time.UTC(),
		EventValue:     value,
		EventTags: map[string]string{
			"eventID": string(event.UID),
		},
	}
	if event.InvolvedObject.Kind == "Pod" {
		point.EventTags[core.LabelPodId.Key] = string(event.InvolvedObject.UID)
		point.EventTags[core.LabelPodName.Key] = event.InvolvedObject.Name
	}
	point.EventTags[core.LabelHostname.Key] = event.Source.Host
	return &point, nil
}

func (sink *elasticSearchSink) ExportEvents(eventBatch *event_core.EventBatch) {
	sink.Lock()
	defer sink.Unlock()
	for _, event := range eventBatch.Events {
		point, err := eventToPoint(event)
		if err != nil {
			glog.Warningf("Failed to convert event to point: %v", err)
		}
		err = sink.saveData(point.EventTimestamp, []interface{}{*point})
		if err != nil {
			glog.Warningf("Failed to export data to ElasticSearch sink: %v", err)
		}
	}
	sink.flushData()
}

func (sink *elasticSearchSink) Name() string {
	return "ElasticSearch Sink"
}

func (sink *elasticSearchSink) Stop() {
	// nothing needs to be done.
}

func NewElasticSearchSink(uri *url.URL) (event_core.EventSink, error) {
	var esSink elasticSearchSink
	esSvc, err := esCommon.CreateElasticSearchService(uri)
	if err != nil {
		glog.Warning("Failed to config ElasticSearch")
		return nil, err
	}

	esSink.esSvc = *esSvc
	esSink.saveData = func(date time.Time, sinkData []interface{}) error {
		return esSvc.SaveData(date, typeName, sinkData)
	}
	esSink.flushData = func() error {
		return esSvc.FlushData()
	}

	glog.V(2).Info("ElasticSearch sink setup successfully")
	return &esSink, nil
}
