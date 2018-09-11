package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	chrono "github.com/matthewmueller/chrono.go"
)

var (
	logMatcher    = flag.String("log", "", "Regex match of log group names")
	streamMatcher = flag.String("stream", "", "Regex match of stream names")
	debug         = flag.Bool("d", false, "Turn on debug logs")
	start         = flag.String("start", "", "Start time of logs")
)

type l struct {
	group   string
	streams []*cloudwatchlogs.LogStream
}

func main() {
	flag.Parse()

	session := session.Must(session.NewSession())
	client := cloudwatchlogs.New(session)

	startTime, err := parseTime(*start)
	if err != nil {
		log.Fatal(err)
	}

	if *debug {
		fmt.Printf("DEBUG: parsed '%s' as '%s'\n", *start, startTime)
	}

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
			if *debug {
				fmt.Println("DEBUG: matchedGroups", m, *o.LogGroupName)
			}
			matchedGroups = append(matchedGroups, o)
		}
	}

	if len(matchedGroups) == 0 {
		fmt.Println("No groups matched.")
		os.Exit(0)
	}

	matchedStreams := []*l{}
	for _, m := range matchedGroups {
		s, err := getStreams(client, *m.LogGroupName)
		if err != nil {
			log.Fatal(err)
		}

		matchedS := []*cloudwatchlogs.LogStream{}
		for _, ss := range s.LogStreams {
			if *debug {
				fmt.Println("DEBUG: matchedStreams", m, *ss.LogStreamName)
			}
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

beginning:
	for _, s := range matchedStreams {
		for _, ss := range s.streams {
			wg.Add(1)
			go func(streamName, group string) {
				fmt.Printf("Starting watching of group: '%s', stream: '%s'\n", streamName, group)
				if err := getLogEvents(client, group, streamName); err != nil {
					fmt.Println("ERROR", group, streamName, err)
				}
			}(*ss.LogStreamName, s.group)
		}
	}

	wg.Wait()

	time.Sleep(10 * time.Second)

	goto beginning
}

func getGroups(client *cloudwatchlogs.CloudWatchLogs) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	input := &cloudwatchlogs.DescribeLogGroupsInput{}
	return client.DescribeLogGroups(input)
}

func getStreams(client *cloudwatchlogs.CloudWatchLogs, groupName string) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	input := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(groupName),
	}
	return client.DescribeLogStreams(input)
}

func getLogEvents(client *cloudwatchlogs.CloudWatchLogs, group, stream string) error {
	startTime, err := parseTime(*start)
	if err != nil {
		log.Fatal(err)
	}

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
				tim := time.Unix(*e.Timestamp/1000, 0)
				if *start != "" && !tim.After(*startTime) {
					if *debug {
						fmt.Printf(
							"DEBUG: message for group: '%s', stream: '%s' timestamp: '%s' is after given start: '%s'\n",
							group, stream, tim, *startTime)
					}

					continue
				}

				suf := strings.Split(group, "/")
				fmt.Println(suf[len(suf)-1], stream, tim)
				fmt.Println(*e.Message, "\n")
			}
		}

		nextForwardToken = *output.NextForwardToken

		time.Sleep(10 * time.Second)
	}

	return nil
}

func parseTime(t string) (*time.Time, error) {
	c, err := chrono.New()
	if err != nil {
		return nil, err
	}

	return c.ParseDate(t, time.Now())
}
