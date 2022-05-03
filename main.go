package main

import (
	"context"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/groovili/gogtrends"
	"github.com/pkg/errors"
)

const (
	locUS  = "US"
	catAll = "all"
	langEn = "EN"
)

type Entry struct {
	time  time.Time
	value int
}

var sampleRate int

func main() {
	args := os.Args[1:]
	sampleRate = 2 // sample every x minutes
	ticker := time.NewTicker(time.Minute * time.Duration(sampleRate))
	data := make(map[string][]Entry)

	// populate with first four hour data
	for _, v := range args {
		data[v] = search(v)
	}

	printSet(data)

	for range ticker.C {
		for _, v := range args {
			printSet(data)
			print("fetching: " + v + "\n")
			newData := search(v)
			handleUpdates(newData, data, v)
			printSet(data)
		}
	}
}

func handleUpdates(new []Entry, data map[string][]Entry, key string) {
	old := data[key]
	var oldEntry Entry
	var newEntry Entry
	var overlapEndNew int

	// get point where old data set ends
	// |_________|
	//   |_______|___|
	//           | <- this is the point overlapEndNew
	for i := len(new) - 1; i > 0; i-- {
		if old[len(old)-1].time == new[i].time {
			overlapEndNew = i
			break
		}
	}

	oldEntry = old[len(old)-1]
	newEntry = new[overlapEndNew]

	// check overlap nil
	if oldEntry == (Entry{}) || newEntry == (Entry{}) {
		print("no overlap")
		return
	}

	if oldEntry.value == 0 {
		log.Println("numerator 0")
		return
	}

	if newEntry.value == 0 {
		log.Println(("denom 0"))
	}

	if oldEntry.value == newEntry.value {
		// no change
		data[key] = append(old, new[overlapEndNew:]...)
	} else {
		// get multiplier and update new values
		multiplier := float64(oldEntry.value) / float64(newEntry.value)
		for i := overlapEndNew; i < len(new); i++ {
			new[i].value = int(math.Ceil(float64(new[i].value) * multiplier))
		}
		data[key] = append(data[key], new[overlapEndNew:]...)
	}
}

func printSet(m map[string][]Entry) {
	for k, v := range m {
		println(k)
		for _, d := range v {
			println(d.time.String(), " ", d.value)
		}
	}
}

func search(keyword string) []Entry {
	//Enable debug to see request-response
	// gogtrends.Debug(true)

	ctx := context.Background()

	// get widgets for Golang keyword in programming category
	explore, err := gogtrends.Explore(ctx, &gogtrends.ExploreRequest{
		ComparisonItems: []*gogtrends.ComparisonItem{
			{
				Keyword: keyword,
				Geo:     locUS,
				Time:    "now 4-H",
			},
		},
		Category: 0,
		Property: "",
	}, langEn)
	handleError(err, "Failed to explore widgets")

	overTime, err := gogtrends.InterestOverTime(ctx, explore[0], langEn)
	handleError(err, "Failed in call interest over time")

	return formatData(overTime)
}

func formatData(t []*gogtrends.Timeline) []Entry {
	newTimes := make([]Entry, len(t))
	for i := 0; i < len(t); i++ {
		tFormatted, err := strconv.ParseInt(t[i].Time, 10, 64)
		handleError(err, "base 64 conversion")

		valFormatted, err := strconv.Atoi(t[i].FormattedValue[0])
		handleError(err, "int conversion")

		newTimes[i] = Entry{time.Unix(tFormatted, 0), valFormatted}
	}
	return newTimes[:len(newTimes)-1] // dont include most recent row as its always 0
}

func handleError(err error, errMsg string) {
	if err != nil {
		log.Fatal(errors.Wrap(err, errMsg))
	}
}
