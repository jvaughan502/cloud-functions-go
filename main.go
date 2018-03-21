
// Copyright 2017 Google Inc.
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

package main

import (
	"flag"
	"fmt"
	"net/http"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"cloud.google.com/go/pubsub"
	logging "github.com/spf13/jwalterweatherman"
	"errors"
	"io"
	"os"
	"path"
	"syscall"
	"log"
	"strings"
	"github.com/PagerDuty/go-pagerduty"

	"./nodego"
)

var pubSubClient *pubsub.Client
var gLogFileHandle io.Writer
var authtoken = "byC3Z-EuEWESdfc2WpX2"
var svcKey string

func main() {
	flag.Parse()
	http.HandleFunc(nodego.HTTPTrigger, func(w http.ResponseWriter, r *http.Request) {
		project := flag.String("project", "", "GCP project")

	flag.Parse()
	if *project == "" {
		*project = "hd-www-stage"
		fmt.Println ("Defaulting to hd-www-stage")
	}
	if *project == "hd-www-prod" {
		svcKey = "c50e8815c149460bbc439a19456bc733"
	} else {
		svcKey = "7b4f4d1ca76a42a8a85487e1fc47a981"
	}

	//Initialize logger, pubsub, then create list of expected topics/subscriptions and check against project
	// InitLogger()
	InitPubSub(*project)
	topSubs := createList(*project)
	topicSubscriptions(*project, topSubs)

	})

	nodego.TakeOver()
}


//Initialize PubSub
func InitPubSub(project string) error {
	ctx := context.Background()
	proj := project
	fmt.Println("Using Pubsub project: [" + proj + "]")
	client, err := pubsub.NewClient(ctx, proj)
	if err != nil {
		fmt.Println(err)
		return errors.New("pubsub initialization failed. Could not create pubsub Client")
	}
	pubSubClient = client

	return nil
}

//Create project specific list of topics and subscriptions
func createList(project string) (map[string][]string) {
	var zonalTopics = []string {"/topics/sunrise-product-indexer","/topics/sunrise-product-indexer-delta","/topics/sunrise-metadata-indexer","/topics/sunrise-content-indexer","/topics/sunrise-rule-preview-indexer","/topics/sunrise-rule-live-indexer","/topics/sunrise-orchestration-mzone"}
	var zonalSubs = []string {"/subscriptions/sunrise-product-indexer", "/subscriptions/sunrise-product-indexer-delta","/subscriptions/sunrise-metadata-indexer", "/subscriptions/sunrise-content-indexer", "/subscriptions/sunrise-rule-preview-indexer","/subscriptions/sunrise-rule-live-indexer", "/subscriptions/sunrise-orchestration"}
	var staticTopics = []string {"/topics/SA.EL.RTD.CAT.PUB","/topics/SA.EL.RTP.THD.CAT.VT.SUB.PUB","/topics/solr-inventory","/topics/catalog-search-sku-data","/topics/catalog-search-sku-str-status","/topics/catalog-search-sku-str-pricing","/topics/sunrise-orchestration-szone"}
	var staticSubs = []string {"/subscriptions/sunrise-streaming-rtd","/subscriptions/sunrise-streaming-rtp","/subscriptions/sunrise-streaming-rti", "/subscriptions/sunrise-sku-data","/subscriptions/sunrise-sku-str-status","/subscriptions/sunrise-sku-str-pricing","/subscriptions/sunrise-orchestration"}
	var topics []string
	var subs []string
	var zones []string
	var pairs = make(map[string][]string)

	if project == "hd-www-prod" {
		zones = []string {"-us-east1-b","-us-east1-c","-us-east1-d","-us-central1-b","-us-central1-c","-us-central1-f"}

		for _,v := range zonalTopics {
			v = "projects/" + project + v
			topics = append(topics, v)
		}

		for _,v := range zonalSubs {
			v = "projects/" + project + v
			subs = append(subs, v)
		}

		for k,topic :=range topics {
			pairs[topic] = append(pairs[topic], (subs[k] + zones[0]))
			pairs[topic] = append(pairs[topic], (subs[k] + zones[1]))
			pairs[topic] = append(pairs[topic], (subs[k] + zones[2]))
			pairs[topic] = append(pairs[topic], (subs[k] + zones[3]))
			pairs[topic] = append(pairs[topic], (subs[k] + zones[4]))
			pairs[topic] = append(pairs[topic], (subs[k] + zones[5]))
		}
		//Adding static topics and subs
		for k, topic := range staticTopics {
			topic = "projects/" + project + topic
			pairs[topic] = append(pairs[topic], "projects/" + project +(staticSubs[k]))
		}

	} else if project == "hd-www-stage" {
		zones = []string {"-us-east1-c","-us-central1-c"}

		for _,v := range zonalTopics {
			v = "projects/" + project + v
			topics = append(topics, v)
		}

		for _,v := range zonalSubs {
			v = "projects/" + project + v
			subs = append(subs, v)
		}

		for k,topic :=range topics {
			pairs[topic] = append(pairs[topic], (subs[k] + zones[0]))
			pairs[topic] = append(pairs[topic], (subs[k] + zones[1]))
		}
	//Adding static topics and subs
		for k, topic := range staticTopics {
			topic = "projects/" + project + topic
			pairs[topic] = append(pairs[topic], "projects/" + project +(staticSubs[k]))
		}

	} else if project == "hd-www-dev" {
		zones = []string {"-us-east1-c"}

		for _,v := range zonalTopics {
			v = "projects/" + project + v
			topics = append(topics, v)
		}

		for _,v := range zonalSubs {
			v = "projects/" + project + v
			subs = append(subs, v)
		}

		for k,topic :=range topics {
			pairs[topic] = append(pairs[topic], (subs[k] + zones[0]))
		}
		//Adding static topics and subs
		for k, topic := range staticTopics {
			topic = "projects/" + project + topic
			pairs[topic] = append(pairs[topic], "projects/" + project +(staticSubs[k]))
		}

	} else {
		fmt.Println("ERROR -- Please choose hd-www-dev, hd-www-stage, or hd-www-prod")
	}
	return pairs
}

//Call project topics and subscriptions and compare vs. what is expected
func topicSubscriptions(project string, topSubs map[string][]string){
	ctx := context.Background()

	//Grab all topics in project and confirm expected topics exist
	var topics []string
	it := pubSubClient.Topics(ctx)
	for {
		topic, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err)
		}
		topics = append(topics,topic.String())
	}
	var miss []string
	var exist []string
	for t, _ := range topSubs {
		found := false
		for _, topic := range topics {
			if t == topic {
				found = true
				exist = append(exist, t)
				fmt.Println("Topic Found: ", t)
				break
			}
		}
		if !found {
			miss = append(miss, t)
			fmt.Println("Topic Missing: ", t)
			pdHook("Missing Topic: " + t)
		}
	}

	//Take the found existing topics, and pull their subscriptions and verify against expected
	var prodSubs []string
	for _, e := range exist {
		index := strings.Index(e, "topics/")
		e = e[index+7:]
		topicCheck := pubSubClient.Topic(e)
		for subs := topicCheck.Subscriptions(ctx); ; {
			sub, err := subs.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Println(err)
			}
			prodSubs = append(prodSubs, sub.String())
		}
	}

	var expSubs []string //expected subs
	for _,s :=range topSubs {
		for _, n := range s {
			expSubs = append(expSubs, n)
		}
	}

	miss = nil
	exist = nil
	for _, e := range expSubs {
		found := false
		for _, psub := range prodSubs {
			if e == psub {
				found = true
				exist = append(exist,e)
				fmt.Println("Subscription Found: ", e)
				break
			}
		}
		if !found {
			miss = append(miss, e)
			fmt.Println("Subscription Missing: ", e)
			pdHook("Missing Subscription: " + e)
		}
	}

}

func pdHook (dsc string) {

	event := pagerduty.Event{
		Type: "trigger",
		ServiceKey: svcKey,
		Description: dsc,
	}
	resp, err := pagerduty.CreateEvent(event)
	if err != nil {
		fmt.Println(resp)
		fmt.Println("ERROR: ", err)
	}
	fmt.Println("Incident key: ", resp.IncidentKey)

}


