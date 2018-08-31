package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	jsonDateFormat = "2006-01-02T15:04:05-07:00"
)

var (
	frabURL   = flag.String("frab", "", "URL to frab conference")
	saveDir   = flag.String("save", ".", "path to save output files to")
	baseID    = flag.Int("id", 0, "base ID to add all object IDs to")
	updatedAt = time.Now().Format(jsonDateFormat)
)

func main() {
	flag.Parse()
	if frabURL == nil || *frabURL == "" {
		log.Fatal("Missing FRAB URL")
	}

	scheduleBody, err := httpGet(fmt.Sprintf("%s/public/schedule.json", *frabURL))
	if err != nil {
		log.Fatal(err)
	}

	speakerBody, err := httpGet(fmt.Sprintf("%s/public/speakers.json", *frabURL))
	if err != nil {
		log.Fatal(err)
	}

	frabSchedule := FrabSchedule{}
	err = json.Unmarshal(scheduleBody, &frabSchedule)
	if err != nil {
		log.Fatal(err)
	}

	frabSpeakers := FrabScheduleSpeakers{}
	err = json.Unmarshal(speakerBody, &frabSpeakers)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(frab.Schedule.Conference.Title)

	eventMap, eventTypesJSON := makeEventTypes(frabSchedule)

	locationMap, locationJSON := makeLocations(frabSchedule)

	speakersJSON := makeSpeakers(frabSchedule, frabSpeakers)

	eventsJSON := makeEvents(frabSchedule, locationMap, eventMap)

	// save data
	if _, err := os.Stat(*saveDir); os.IsNotExist(err) {
		os.Mkdir(*saveDir, os.ModePerm)
	}

	err = saveFile("event_types.json", eventTypesJSON)
	if err != nil {
		log.Fatal(err)
	}
	err = saveFile("locations.json", locationJSON)
	if err != nil {
		log.Fatal(err)
	}
	err = saveFile("speakers.json", speakersJSON)
	if err != nil {
		log.Fatal(err)
	}
	err = saveFile("events.json", eventsJSON)
	if err != nil {
		log.Fatal(err)
	}
}

func saveFile(name string, data []byte) error {
	return ioutil.WriteFile(fmt.Sprintf("%s/%s", *saveDir, name), data, 0644)
}

func httpGet(url string) ([]byte, error) {
	httpClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

func makeEvents(frab FrabSchedule, locationMap, eventMap map[string]int) []byte {
	events := make([]*HackerTrackerEvent, 0, 20)

	// iterate over all events
	for _, day := range frab.Schedule.Conference.Days {
		for room := range day.Rooms {
			for _, frabEvent := range day.Rooms[room] {
				startTime, err := time.Parse(jsonDateFormat, frabEvent.Date)
				if err != nil {
					log.Fatal(err)
				}
				timeParts := strings.Split(frabEvent.Duration, ":")
				if len(timeParts) != 2 {
					log.Fatal(fmt.Errorf("Unable to parse end time %s", frabEvent.Duration))
				}
				duration, err := time.ParseDuration(fmt.Sprintf("%sh%sm", timeParts[0], timeParts[1]))
				if err != nil {
					log.Fatal(err)
				}
				end := startTime.Add(duration)
				event := &HackerTrackerEvent{
					StartDate:   frabEvent.Date,
					ID:          *baseID + frabEvent.ID,
					Description: frabEvent.Abstract,
					Location:    locationMap[frabEvent.Room],
					EndDate:     end.Format(jsonDateFormat),
					Conference:  frab.Schedule.Conference.Acronym,
					EventType:   eventMap[frabEvent.Type],
					Title:       frabEvent.Title,
					UpdatedAt:   updatedAt,
				}
				if len(frabEvent.Links) > 0 {
					event.Link = frabEvent.Links[0].URL
				}
				for _, person := range frabEvent.Persons {
					event.Speakers = append(event.Speakers, *baseID+person.ID)
				}
				events = append(events, event)
			}
		}
	}

	json, err := HackerTrackerMarshal("schedule", events)
	if err != nil {
		log.Fatal(err)
	}
	return json
}

func makeSpeakers(frab FrabSchedule, frabSpeakers FrabScheduleSpeakers) []byte {
	// iterate over speakers
	speakers := make([]*HackerTrackerSpeaker, 0, 10)

	for _, frabSpeaker := range frabSpeakers.ScheduleSpeakers.Speakers {
		speaker := &HackerTrackerSpeaker{
			Name:        frabSpeaker.PublicName,
			UpdatedAt:   updatedAt,
			Description: frabSpeaker.Abstract,
			ID:          *baseID + frabSpeaker.ID,
			Conference:  frab.Schedule.Conference.Acronym,
		}
		if len(frabSpeaker.Links) > 0 {
			speaker.Link = frabSpeaker.Links[0].URL
		}
		speakers = append(speakers, speaker)
	}

	json, err := HackerTrackerMarshal("speakers", speakers)
	if err != nil {
		log.Fatal(err)
	}
	return json
}

func makeLocations(frab FrabSchedule) (map[string]int, []byte) {
	locationMap := make(map[string]int)
	roomCount := *baseID

	// iterate over all events
	for _, day := range frab.Schedule.Conference.Days {
		for room := range day.Rooms {
			if locationMap[room] == 0 {
				roomCount++
				locationMap[room] = roomCount

			}
		}
	}

	locations := make([]*HackerTrackerLocation, 0, 1)
	for name, id := range locationMap {
		location := &HackerTrackerLocation{
			Name:       name,
			ID:         id,
			UpdatedAt:  updatedAt,
			Conference: frab.Schedule.Conference.Acronym,
		}
		locations = append(locations, location)
	}

	json, err := HackerTrackerMarshal("locations", locations)
	if err != nil {
		log.Fatal(err)
	}
	return locationMap, json
}

func makeEventTypes(frab FrabSchedule) (map[string]int, []byte) {
	// find all event types
	eventMap := make(map[string]int)
	eventCount := *baseID

	// iterate over all events
	for _, day := range frab.Schedule.Conference.Days {
		for room := range day.Rooms {
			for _, event := range day.Rooms[room] {
				//fmt.Println("AAA", " [", event.Title, "]")
				if eventMap[event.Type] == 0 {
					eventCount++
					eventMap[event.Type] = eventCount
				}
			}
		}
	}

	eventTypes := make([]*HackerTrackerEventType, 0, 1)
	for name, id := range eventMap {
		eventType := &HackerTrackerEventType{
			Name:       strings.Title(strings.Replace(name, "_", " ", 0)),
			Conference: frab.Schedule.Conference.Acronym,
			ID:         id,
			UpdatedAt:  updatedAt,
		}
		eventTypes = append(eventTypes, eventType)
	}

	json, err := HackerTrackerMarshal("event_types", eventTypes)
	if err != nil {
		log.Fatal(err)
	}

	return eventMap, json
}
