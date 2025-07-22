package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Asset represents a photo/video asset from Immich
// Add more fields as needed
type Asset struct {
	ID               string `json:"id"`
	OriginalFileName string `json:"originalFileName"`
	FileCreatedAt    string `json:"fileCreatedAt"`
}

func main() {
	apiURL := os.Getenv("IMMICH_API_URL")
	apiKey := os.Getenv("IMMICH_API_KEY")
	if apiURL == "" || apiKey == "" {
		log.Fatal("IMMICH_API_URL and IMMICH_API_KEY must be set in environment variables")
	}

	searchDate := "2023-08-01" // TODO: make this configurable

	assets, err := getAssetsByDate(apiURL, apiKey, searchDate)
	if err != nil {
		log.Fatalf("Failed to get assets: %v", err)
	}

	if len(assets) == 0 {
		fmt.Println("No assets found.")
		return
	}

	// Asset ID to focus on
	targetID := "d70e3a48-7dbb-474f-821d-f9e95d1106d6"

	found := false
	for _, asset := range assets {
		// if asset.ID != targetID {
		// 	continue
		// }
		found = true
		fmt.Printf("Processing asset: %s (%s)\n", asset.OriginalFileName, asset.ID)

		filenameDate, err := extractDateFromFilename(asset.OriginalFileName)
		if err != nil {
			log.Printf("Could not extract date from filename: %v", err)
			continue
		}
		fmt.Printf("Extracted date from filename: %s\n", filenameDate)

		if asset.FileCreatedAt != filenameDate {
			fmt.Printf("Updating fileCreatedAt from %s to %s\n", asset.FileCreatedAt, filenameDate)
			err = updateAssetFileCreatedAt(apiURL, apiKey, asset.ID, filenameDate)
			if err != nil {
				log.Printf("Failed to update asset: %v", err)
			} else {
				fmt.Println("Asset updated successfully.")
			}
		} else {
			fmt.Println("fileCreatedAt already matches filename date. No update needed.")
		}
		// break
	}
	if !found {
		fmt.Printf("Asset with ID %s not found.\n", targetID)
	}
}

func getAssetsByDate(apiURL, apiKey, date string) ([]Asset, error) {
	start := date + "T00:00:00.000Z"
	end := date + "T23:59:59.999Z"

	allAssets := []Asset{}
	page := 1
	for {
		body := map[string]interface{}{
			"page":        page,
			"withExif":    true,
			"isVisible":   true,
			"language":    "en-US",
			"takenAfter":  start,
			"takenBefore": end,
		}
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		url := strings.TrimRight(apiURL, "/") + "/search/metadata"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(bodyBytes))
		}

		var result struct {
			Assets struct {
				Items []Asset `json:"items"`
			} `json:"assets"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		if len(result.Assets.Items) == 0 {
			break
		}
		allAssets = append(allAssets, result.Assets.Items...)
		page++
	}
	return allAssets, nil
}

// extractDateFromFilename tries to extract a date from various filename patterns and returns it as ISO8601 if valid
func extractDateFromFilename(filename string) (string, error) {
	var (
		matches []string
		timeStr string
		t       time.Time
		err     error
	)

	switch {
	// Screenshot_2016-04-16-09-12-10.png
	case regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})[-_](\d{2})-(\d{2})-(\d{2})`).MatchString(filename):
		matches = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})[-_](\d{2})-(\d{2})-(\d{2})`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s-%s-%s-%s-%s-%s", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6])
		t, err = time.Parse("2006-01-02-15-04-05", timeStr)
	// IMG_20190130_172450.jpg
	case regexp.MustCompile(`(\d{4})(\d{2})(\d{2})[_-](\d{2})(\d{2})(\d{2})`).MatchString(filename):
		matches = regexp.MustCompile(`(\d{4})(\d{2})(\d{2})[_-](\d{2})(\d{2})(\d{2})`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s%s%s_%s%s%s", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6])
		t, err = time.Parse("20060102_150405", timeStr)
	// 2020-12-31 23.59.59.jpg
	case regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})[ _](\d{2})\.(\d{2})\.(\d{2})`).MatchString(filename):
		matches = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})[ _](\d{2})\.(\d{2})\.(\d{2})`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s-%s-%s %s.%s.%s", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6])
		t, err = time.Parse("2006-01-02 15.04.05", timeStr)
	// photo-20211225-153045.jpg
	case regexp.MustCompile(`(\d{4})(\d{2})(\d{2})[-_](\d{2})(\d{2})(\d{2})`).MatchString(filename):
		matches = regexp.MustCompile(`(\d{4})(\d{2})(\d{2})[-_](\d{2})(\d{2})(\d{2})`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s%s%s-%s%s%s", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6])
		t, err = time.Parse("20060102-150405", timeStr)
	// IMG-20160123-WA0000.jpg
	case regexp.MustCompile(`IMG-(\d{4})(\d{2})(\d{2})-WA\d+`).MatchString(filename):
		matches = regexp.MustCompile(`IMG-(\d{4})(\d{2})(\d{2})-WA\d+`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s-%s-%sT00:00:00Z", matches[1], matches[2], matches[3])
		t, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
	// 2015-11-01_1234.mp4
	case regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})_(\d{2})(\d{2})`).MatchString(filename):
		matches = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})_(\d{2})(\d{2})`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s-%s-%sT%s:%s:00Z", matches[1], matches[2], matches[3], matches[4], matches[5])
		t, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
	// IMG_2016-07-10-18472672.png
	case regexp.MustCompile(`IMG_(\d{4})-(\d{2})-(\d{2})-(\d{2})(\d{2})(\d{2})\d{2}`).MatchString(filename):
		matches = regexp.MustCompile(`IMG_(\d{4})-(\d{2})-(\d{2})-(\d{2})(\d{2})(\d{2})\d{2}`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s-%s-%sT%s:%s:%sZ", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6])
		t, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
	// 2014-09-18.jpg, 2014-08-30(1).jpg, 2015-02-07.gif, 2015-05-17-edited.jpg
	case regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})(?:[^\s.]*)?\.[a-zA-Z0-9]+`).MatchString(filename):
		matches = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`).FindStringSubmatch(filename)
		timeStr = fmt.Sprintf("%s-%s-%s", matches[1], matches[2], matches[3])
		t, err = time.Parse("2006-01-02", timeStr)
	// 13-digit Unix timestamp in milliseconds, e.g., 1483613738152.jpg or 1431877338773-edited.jpg (must be exactly 13 digits before optional suffix and extension)
	case regexp.MustCompile(`(?:^|[^0-9])(\d{13})(?:[^\d\s.]*)?\.[a-zA-Z0-9]+$`).MatchString(filename):
		matches = regexp.MustCompile(`(?:^|[^0-9])(\d{13})(?:[^\d\s.]*)?\.[a-zA-Z0-9]+$`).FindStringSubmatch(filename)
		if len(matches) > 1 {
			tsInt, errConv := strconv.ParseInt(matches[1], 10, 64)
			if errConv == nil {
				t = time.Unix(0, tsInt*int64(time.Millisecond))
				if t.Year() >= 2010 && t.Year() <= 2023 {
					return t.UTC().Format("2006-01-02T15:04:05Z"), nil
				}
			}
		}
		return "", fmt.Errorf("invalid unix timestamp in filename: %s", filename)
	// BURST20190113203829.jpg
	case regexp.MustCompile(`BURST(\d{8})(\d{6})`).MatchString(filename):
		matches = regexp.MustCompile(`BURST(\d{8})(\d{6})`).FindStringSubmatch(filename)
		if len(matches) == 3 {
			timeStr = fmt.Sprintf("%sT%s", matches[1], matches[2])
			t, err = time.Parse("20060102T150405", timeStr)
		}
	// _YYYYMMDDHHMMSS before extension, e.g., Burst_Cover_GIF_Action_20180420201801.gif
	case regexp.MustCompile(`_(\d{8})(\d{6})\.[a-zA-Z0-9]+$`).MatchString(filename):
		matches = regexp.MustCompile(`_(\d{8})(\d{6})\.[a-zA-Z0-9]+$`).FindStringSubmatch(filename)
		if len(matches) == 3 {
			timeStr = fmt.Sprintf("%sT%s", matches[1], matches[2])
			t, err = time.Parse("20060102T150405", timeStr)
		}
	// CameraZOOM-YYYYMMDDXXXXXXXXX.jpg (use only the date, ignore time, return midnight)
	case regexp.MustCompile(`CameraZOOM-(\d{4})(\d{2})(\d{2})\d+\.[a-zA-Z0-9]+$`).MatchString(filename):
		matches = regexp.MustCompile(`CameraZOOM-(\d{4})(\d{2})(\d{2})\d+\.[a-zA-Z0-9]+$`).FindStringSubmatch(filename)
		if len(matches) == 4 {
			timeStr = fmt.Sprintf("%s-%s-%sT00:00:00Z", matches[1], matches[2], matches[3])
			t, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
		}
	// CameraZOOM-20150829144707466.jpg (YYYYMMDDHHMMSSsss after dash)
	case regexp.MustCompile(`-(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{3})`).MatchString(filename):
		matches = regexp.MustCompile(`-(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})(\d{3})`).FindStringSubmatch(filename)
		if len(matches) == 8 {
			timeStr = fmt.Sprintf("%s-%s-%sT%s:%s:%s.%sZ", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6], matches[7])
			t, err = time.Parse("2006-01-02T15:04:05.000Z", timeStr)
		}
	default:
		return "", fmt.Errorf("no valid date found in filename: %s", filename)
	}

	if err != nil {
		return "", fmt.Errorf("invalid date in filename: %s", filename)
	}
	return t.UTC().Format("2006-01-02T15:04:05Z"), nil
}

// updateAssetFileCreatedAt sends a PUT request to update dateTimeOriginal
func updateAssetFileCreatedAt(apiURL, apiKey, assetID, newDate string) error {
	url := fmt.Sprintf("%s/assets/%s", strings.TrimRight(apiURL, "/"), assetID)
	body := map[string]string{
		"dateTimeOriginal": newDate,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(bodyBytes))
	}
	return nil
}
