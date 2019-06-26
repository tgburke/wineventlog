package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/eventlog"
)

func main() {
	getLogs()
}

func getLogs() {
	var states map[string]checkpoint.EventLogState
	states = make(map[string]checkpoint.EventLogState)

	channels := []string{"System", "Application", "Security"} // Security requires elevated priviliges
	var configs []*common.Config
	for _, channel := range channels {
		// For config options see: https://github.com/elastic/beats/blob/master/winlogbeat/docs/winlogbeat-options.asciidoc
		config, _ := common.NewConfigFrom(common.MapStr{
			"name":            channel,
			"api":             "wineventlog",
			"include_xml":     false,
			"ignore_older":    "1m",
			"no_more_events":  "wait",
			"batch_read_size": 100,
		})
		configs = append(configs, config)
		states[channel] = checkpoint.EventLogState{}
	}

	var wg sync.WaitGroup
	wg.Add(len(configs))

	for _, config := range configs {
		go func(config *common.Config, wg *sync.WaitGroup) {
			defer wg.Done()
			for { // loop until ctrl-c
				// example of log open/close on each iteration keeping track of the EventLogState
				//   so we don't lose the Bookmark of the last event received
				log, err := eventlog.New(config)
				if err != nil {
					fmt.Println(err)
					return
				}

				if err = log.Open(states[log.Name()]); err != nil {
					fmt.Println(err)
					return
				}

				records, err := log.Read()
				if err != nil {
					fmt.Println(err)
					return

				} else if len(records) == 0 {
					fmt.Printf("channel[%s] sleeping\n", log.Name())
					time.Sleep(10 * time.Second)

				} else {
					// do something with the Records  ref: https://github.com/elastic/beats/blob/master/winlogbeat/eventlog/eventlog.go#L77
					for _, record := range records {
						fmt.Printf("event[%v]\n", record.ToEvent())
					}
					fmt.Printf("channel[%s] count[%d]\n\n", log.Name(), len(records))
					states[log.Name()] = records[len(records)-1].Offset
					fmt.Printf("state %v\n", states[log.Name()])
				}
				log.Close()
			}
		}(config, &wg)
	}
	wg.Wait()
}
