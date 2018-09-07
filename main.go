package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

var (
	logMatcher    = flag.String("log", "", "Regex match of log group names")
	streamMatcher = flag.String("stream", "", "Regex match of stream names")
)

type l struct {
	group   string
	streams []*cloudwatchlogs.LogStream
}

func main() {
	flag.Parse()

	session := session.Must(session.NewSession())
	client := cloudwatchlogs.New(session)

	logOutput, err := getGroups(client)
	if err != nil {
		log.Fatal(err)
	}

	matchedGroups := []*cloudwatchlogs.LogGroup{}
	for _, o := range logOutput.LogGroups {
		m, err := regexp.MatchString(*logMatcher, *o.LogGroupName)
		if err != nil {
			log.Fatal(err)
		}
		if m {
			matchedGroups = append(matchedGroups, o)
		}
	}

	matchedStreams := []*l{}
	for _, m := range matchedGroups {
		s, err := getStreams(client, *m.LogGroupName)
		if err != nil {
			log.Fatal(err)
		}

		matchedS := []*cloudwatchlogs.LogStream{}
		for _, ss := range s.LogStreams {
			m, err := regexp.MatchString(*streamMatcher, *ss.LogStreamName)
			if err != nil {
				log.Fatal(err)
			}
			if m {
				matchedS = append(matchedS, ss)
			}
		}

		matchedStreams = append(matchedStreams, &l{group: *m.LogGroupName, streams: matchedS})
	}

	var wg sync.WaitGroup

	for _, s := range matchedStreams {
		for _, ss := range s.streams {
			wg.Add(1)
			go func(streamName string) {
				if err := getLogEvents(client, s.group, streamName); err != nil {
					fmt.Println(">>>>>>>>>", err)
				}
			}(*ss.LogStreamName)
		}
	}

	wg.Wait()
}

func getGroups(client *cloudwatchlogs.CloudWatchLogs) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	input := &cloudwatchlogs.DescribeLogGroupsInput{}
	return client.DescribeLogGroups(input)
}

func getStreams(client *cloudwatchlogs.CloudWatchLogs, groupName string) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	input := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(groupName),
		Descending:   aws.Bool(true),
	}
	return client.DescribeLogStreams(input)
}

func getLogEvents(client *cloudwatchlogs.CloudWatchLogs, group, stream string) error {
	var nextForwardToken string
	for {
		input := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(group),
			LogStreamName: aws.String(stream),
		}

		if nextForwardToken != "" {
			input.SetNextToken(nextForwardToken)
		} else {
			input.SetLimit(100)
		}

		output, err := client.GetLogEvents(input)
		if err != nil {
			return err
		}

		if nextForwardToken != *output.NextForwardToken {
			for _, e := range output.Events {
				suf := strings.Split(group, "/")
				fmt.Println(suf[len(suf)-1], stream, time.Unix(*e.Timestamp/1000, 0).String(), *e.Message)
			}
		}

		nextForwardToken = *output.NextForwardToken

		time.Sleep(10 * time.Second)
	}

	return nil
}
