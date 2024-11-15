package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	v2 "github.com/matsuri-tech/beds24-sdk-go/v2"
)

func main() {
	token := os.Getenv("BEDS24_API_TOKEN")

	client := v2.NewAPIClient(v2.NewConfiguration())
	client.GetConfig().AddDefaultHeader("token", token)

	// if err := writeAllPropertyIds(client, "property_ids.txt"); err != nil {
	// 	log.Fatal(err)
	// }

	if err := fetchReviewsSince(client, "property_ids.txt", "reviews.json", "2023-06-01"); err != nil {
		log.Fatal(err)
	}

	if err := removeDuplicates("reviews.json", "reviews_unique.json"); err != nil {
		log.Fatal(err)
	}
}

func writeAllPropertyIds(client *v2.APIClient, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	for page := 1; page < 1000; page++ {
		resp, _, err := client.PropertiesAPI.PropertiesGet(context.Background()).Page(int32(page)).Execute()
		if err != nil {
			return err
		}
		if len(resp.Data) == 0 {
			break
		}

		log.Printf("Page %v, got %v", page, len(resp.Data))
		for _, prop := range resp.Data {
			file.WriteString(fmt.Sprintf("%v", prop.GetId()))
			file.WriteString("\n")
		}

	}

	return nil
}

func fetchReviewsSince(client *v2.APIClient, propIdFilePath string, filepath string, since string) error {
	propIdFile, err := os.Open(propIdFilePath)
	if err != nil {
		return err
	}
	defer propIdFile.Close()

	propIds := []int{}
	for {
		var propId int
		_, err := fmt.Fscanln(propIdFile, &propId)
		if err != nil {
			break
		}
		propIds = append(propIds, propId)
	}

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	for _, propId := range propIds {
		log.Printf("Fetching reviews for PropId %v", propId)

		from := since
		for page := 0; page < 1000; page++ {
			time.Sleep(1500 * time.Millisecond)

			resp, h, err := client.ChannelsBookingComAPI.
				ChannelsBookingReviewsGet(context.Background()).
				PropertyId(int32(propId)).
				From(from).
				Execute()
			if err != nil {
				return err
			}
			if page == 0 {
				log.Printf("headers: %v", h.Header)
			}

			log.Printf("PropId %v, Page %v, from %v, got %v", propId, page, from, len(resp.Data))

			for _, review := range resp.Data {
				bs, _ := json.Marshal(review)
				file.Write(bs)
				file.WriteString("\n")

				from = review.GetCreatedTimestamp()[0:10]
			}

			// 日が被っている関係で次のページに行っても1件以上取得される場合がある
			if len(resp.Data) < 100 {
				break
			}
		}
	}

	return nil
}

func removeDuplicates(filepath string, resultFilepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	resultFile, err := os.Create(resultFilepath)
	if err != nil {
		return err
	}
	defer resultFile.Close()

	decoder := json.NewDecoder(file)
	visited := map[string]bool{}
	for {
		var review v2.BookingReview

		if err := decoder.Decode(&review); err != nil {
			break
		}

		current := review.GetReviewId()
		if _, ok := visited[current]; ok {
			continue
		}

		bs, _ := json.Marshal(review)
		resultFile.Write(bs)
		resultFile.WriteString("\n")

		visited[current] = true
	}

	return nil
}
