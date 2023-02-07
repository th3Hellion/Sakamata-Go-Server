package main

import (
  "encoding/json"
  "fmt"
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

func fetchData() {

  var channelID = os.Getenv("CHANNEL_ID")
  var apiKey = os.Getenv("API_KEY")
  fmt.Println("Fetching Data at:", time.Now().Format(time.RFC1123))

  url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&channelId=%s&channelType=any&order=date&type=video&videoCaption=any&videoDefinition=any&videoDimension=any&videoDuration=any&videoEmbeddable=any&videoLicense=any&videoSyndicated=any&videoType=any&key=%s", channelID, apiKey)
  res, err := http.Get(url)
  if err != nil {
    log.Fatalf("Failed to fetch data: %v", err)
  }
  defer res.Body.Close()

  var result map[string]interface{}
  if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
    log.Fatalf("Failed to decode data: %v", err)
  }
  fmt.Println(result)

  items, ok := result["items"].([]interface{})
  if !ok {
    itemsInterf := result["items"]
    if itemsInterf == nil {
      log.Fatalf("items not found in result")
    }
    items, ok = itemsInterf.([]interface{})
    if !ok || len(items) == 0 {
      log.Fatalf("Failed to get items from the result")
    }
  }

  var liveItem map[string]interface{}
  for _, item := range items {
    itemMap := item.(map[string]interface{})
    if itemMap["snippet"].(map[string]interface{})["liveBroadcastContent"].(string) == "live" {
      liveItem = itemMap
      break
    }
  }

  if liveItem != nil {
    livestreamStatus := liveItem["snippet"].(map[string]interface{})["liveBroadcastContent"].(string)
    videoID := liveItem["id"].(map[string]interface{})["videoId"].(string)
    videoData = VideoData{LivestreamStatus: livestreamStatus, VideoID: videoID, Updated: "Stream is Live", FetchedAt: time.Now()}
  } else {
    var mostRecentVideo map[string]interface{}
    for i, item := range items {
      if i == 0 {
        mostRecentVideo = item.(map[string]interface{})
        continue
      }
      itemMap := item.(map[string]interface{})
      if mostRecentVideo["snippet"].(map[string]interface{})["publishedAt"].(string) < itemMap["snippet"].(map[string]interface{})["publishedAt"].(string) {
        mostRecentVideo = itemMap
      }
    }
    livestreamStatus := mostRecentVideo["snippet"].(map[string]interface{})["liveBroadcastContent"].(string)
    videoID := mostRecentVideo["id"].(map[string]interface{})["videoId"].(string)
    videoData = VideoData{LivestreamStatus: livestreamStatus, VideoID: videoID, Updated: "Stream is Offline", FetchedAt: time.Now()}
  }
}

func main() {

  fetchData()

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(videoData)
  })

  fmt.Println("Listening on port", 3000)
  log.Fatal(http.ListenAndServe(":3000", nil))
}
