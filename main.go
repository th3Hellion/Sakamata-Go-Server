package main

import (
  "encoding/json"
  "fmt"
  "github.com/joho/godotenv"
  "github.com/rs/cors"
  "io"
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

  var origin = "https://lizasil.github.io/Sakamata"
  fmt.Println("Fetching Data at:", time.Now().Format(time.RFC1123))

  url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&channelId=%s&channelType=any&order=date&type=video&videoCaption=any&videoDefinition=any&videoDimension=any&videoDuration=any&videoEmbeddable=any&videoLicense=any&videoSyndicated=any&videoType=any&key=%s&origin=%s", channelID, apiKey, origin)
  res, err := http.Get(url)
  if err != nil {
    fmt.Println("Failed to fetch the data")
  }
  defer res.Body.Close()

  var result map[string]interface{}
  if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
    fmt.Println("Failed to decode the result")
  }

  items, ok := result["items"].([]interface{})
  if !ok {
    itemsInterf := result["items"]
    if itemsInterf == nil {
      log.Fatal("Failed to get items from the result")
    }
    items, ok = itemsInterf.([]interface{})
    if !ok || len(items) == 0 {
      log.Fatal("Failed to get items from the result")
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
    videoData = VideoData{LivestreamStatus: livestreamStatus, VideoID: videoID, Updated: fetchEndTime(videoID, apiKey), FetchedAt: time.Now()}
  }
}

func fetchEndTime(videoId, apiKey string) string {
  url := fmt.Sprintf("https://youtube.googleapis.com/youtube/v3/videos?part=liveStreamingDetails&id=%s&key=%s", videoId, apiKey)
  resp, err := http.Get(url)
  if err != nil {
    return ""
  }
  defer resp.Body.Close()
  body, err := io.ReadAll(resp.Body)
  if err != nil {
    return ""
  }
  var data struct {
    Items []struct {
      LiveStreamingDetails struct {
        ActualEndTime string `json:"actualEndTime"`
      } `json:"liveStreamingDetails"`
    } `json:"items"`
  }
  err = json.Unmarshal(body, &data)
  if err != nil {
    return ""
  }
  if len(data.Items) == 0 {
    return ""
  }
  return data.Items[0].LiveStreamingDetails.ActualEndTime
}

func main() {
  handler := cors.AllowAll().Handler(http.DefaultServeMux)
  err := godotenv.Load()
  if err != nil {
    log.Fatalf("Error loading .env file")
  }

  fetchData()

  ticker := time.NewTicker(14 * time.Second)
  go func() {
    for range ticker.C {
      fetchData()
      fmt.Println("Fetching Data at:", time.Now().Format(time.RFC1123))
    }
  }()

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(videoData)
  })

  fmt.Println("Listening on port", 3000)
  log.Fatal(http.ListenAndServe(":3000", handler))
}