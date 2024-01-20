package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
	"time"
)

type VideoData struct {
	LivestreamStatus string    `json:"livestreamStatus"`
	VideoID          string    `json:"videoId"`
	Updated          string    `json:"updated"`
	FetchedAt        time.Time `json:"fetched_at"`
}

var videoData VideoData
var origin = "https://th3Hellion.github.io/Sakamata"

func makeRequest(url string, target interface{}) error {
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch data from %s: %v", url, err)
		return err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		log.Printf("Failed to decode JSON response from %s: %v", url, err)
		return err
	}

	return nil
}

func handleLiveItem(items []interface{}) {
	for _, item := range items {
		itemMap := item.(map[string]interface{})
		snippet := itemMap["snippet"].(map[string]interface{})
		if liveBroadcastContent, ok := snippet["liveBroadcastContent"].(string); ok && liveBroadcastContent == "live" {
			videoData = VideoData{
				LivestreamStatus: liveBroadcastContent,
				VideoID:          itemMap["id"].(map[string]interface{})["videoId"].(string),
				Updated:          "Stream is Live",
				FetchedAt:        time.Now(),
			}
			break
		}
	}
}

func handleMostRecentVideo(items []interface{}) {
	var mostRecentVideo map[string]interface{}

	for i, item := range items {
		itemMap := item.(map[string]interface{})
		publishedAt := itemMap["snippet"].(map[string]interface{})["publishedAt"]
		if publishedAt == nil || publishedAt == "" {
			continue
		}

		if i == 0 || publishedAt.(string) > mostRecentVideo["snippet"].(map[string]interface{})["publishedAt"].(string) {
			mostRecentVideo = itemMap
		}
	}

	if mostRecentVideo != nil {
		livestreamStatus := mostRecentVideo["snippet"].(map[string]interface{})["liveBroadcastContent"].(string)
		videoID := mostRecentVideo["id"].(map[string]interface{})["videoId"].(string)
		publishedAt := mostRecentVideo["snippet"].(map[string]interface{})["publishedAt"].(string)

		updated := fetchEndTime(videoID)
		if updated == "" {
			updated = publishedAt
		}

		videoData = VideoData{
			LivestreamStatus: livestreamStatus,
			VideoID:          videoID,
			Updated:          updated,
			FetchedAt:        time.Now(),
		}
	} else {
		videoData = VideoData{
			LivestreamStatus: "none",
			VideoID:          "none",
			Updated:          "none",
			FetchedAt:        time.Now(),
		}
	}
}

func fetchEndTime(videoID string) string {
	apiKey := os.Getenv("API_KEY")
	url := fmt.Sprintf("https://youtube.googleapis.com/youtube/v3/videos?part=liveStreamingDetails&id=%s&key=%s", videoID, apiKey)

	var data struct {
		Items []struct {
			LiveStreamingDetails struct {
				ActualEndTime string `json:"actualEndTime"`
			} `json:"liveStreamingDetails"`
		} `json:"items"`
	}

	if err := makeRequest(url, &data); err != nil {
		return ""
	}

	if len(data.Items) == 0 {
		return ""
	}

	return data.Items[0].LiveStreamingDetails.ActualEndTime
}

func fetchData() {
	godotenv.Load()

	channelID := os.Getenv("CHANNEL_ID")
	apiKey := os.Getenv("API_KEY")

	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&channelId=%s&channelType=any&order=date&type=video&videoCaption=any&videoDefinition=any&videoDimension=any&videoDuration=any&videoEmbeddable=any&videoLicense=any&videoSyndicated=any&videoType=any&key=%s&origin=%s", channelID, apiKey, origin)

	var result map[string]interface{}
	if err := makeRequest(url, &result); err != nil {
		return
	}

	items, ok := result["items"].([]interface{})
	if !ok || len(items) == 0 {
		videoData = VideoData{
			LivestreamStatus: "none",
			VideoID:          "none",
			Updated:          "none",
			FetchedAt:        time.Now(),
		}
		return
	}

	handleLiveItem(items)
	if videoData.LivestreamStatus == "" {
		handleMostRecentVideo(items)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	handler := cors.AllowAll().Handler(http.DefaultServeMux)

	fetchData()

	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for range ticker.C {
			fetchData()
			fmt.Println("Fetching Data at:", time.Now().Format(time.RFC1123))
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(videoData)
	})

	fmt.Println("Listening on port", 3000)
	log.Fatal(http.ListenAndServe(":3000", handler))
}
