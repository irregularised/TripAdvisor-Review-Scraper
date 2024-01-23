package tripadvisor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// MakeRequest is a function that sends a POST request to the TripAdvisor GraphQL endpoint
func MakeRequest(client *http.Client, queryID string, language string, locationID uint32, offset uint32, limit uint32) (responses *Responses, err error) {

	/*
	* Prepare the request body
	 */
	requestFilter := Filter{
		Axis:       "LANGUAGE",
		Selections: []string{language},
	}

	requestVariables := Variables{
		LocationID:     locationID,
		Offset:         offset,
		Filters:        Filters{requestFilter},
		Limit:          limit,
		NeedKeywords:   false,
		PrefsCacheKey:  fmt.Sprintf("locationReviewPrefs_%d", locationID),
		KeywordVariant: "location_keywords_v2_llr_order_30_en",
		InitialPrefs:   struct{}{},
		FilterCacheKey: nil,
		Prefs:          nil,
	}

	requestExtensions := Extensions{
		PreRegisteredQueryID: queryID,
	}

	requestPayload := Request{
		Variables:  requestVariables,
		Extensions: requestExtensions,
	}

	request := Requests{requestPayload}

	// Marshal the request body into JSON
	jsonPayload, err := json.Marshal(request)
	if err != nil {
		log.Fatal("Error marshalling request body: ", err)
	}

	// Create a new request using http.NewRequest, setting the method to POST
	req, err := http.NewRequest(http.MethodPost, EndPointURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %w", err)
	}

	// Set the necessary headers as per the original Axios request
	req.Header.Set("Origin", "https://www.tripadvisor.com")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_0_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.101 Safari/537.36")
	req.Header.Set("X-Requested-By", "someone-special")
	req.Header.Set("Cookie", "asdasdsa")
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("Rate Limit Detected: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("Error response status code: %d", resp.StatusCode)
	}

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %w", err)
	}

	// Marshal the response body into the Response struct
	responseData := Responses{}
	err = json.Unmarshal(responseBody, &responseData)

	return &responseData, err
}

// GetQueryID is a function that returns the query ID for the given query type
func GetQueryID(queryType string) (queryID string) {

	switch queryType {
	case "HOTEL":
		return HotelQueryID
	case "AIRLINE":
		return AirlineQueryID
	default:
		return HotelQueryID
	}
}

// FetchReviewCount is a function that fetches the review count for the given location ID and query type
func FetchReviewCount(client *http.Client, locationID uint32, queryType string) (reviewCount int, err error) {

	// Get the query ID for the given query type.
	queryID := GetQueryID(queryType)

	// Make the request to the TripAdvisor GraphQL endpoint.
	responses, err := MakeRequest(client, queryID, "en", locationID, 0, 1)
	if err != nil {
		return 0, fmt.Errorf("error making request: %w", err)
	}

	// Check if responses is nil before dereferencing
	if responses == nil {
		return 0, fmt.Errorf("received nil response for location ID %d", locationID)
	}

	// Now it's safe to dereference responses
	response := *responses
	if len(response) > 0 && len(response[0].Data.Locations) > 0 {
		reviewCount = response[0].Data.Locations[0].ReviewListPage.TotalCount
		return reviewCount, nil
	}

	return 0, fmt.Errorf("no reviews found for location ID %d", locationID)
}

// CalculateIterations is a function that calculates the number of iterations required to fetch all reviews
func CalculateIterations(reviewCount uint32) (iterations uint32) {

	// Calculate the number of iterations required to fetch all reviews
	iterations = reviewCount / ReviewLimit

	// If the review count is not a multiple of ReviewLimit, add one more iteration
	if reviewCount%ReviewLimit != 0 {
		return iterations + 1
	}

	return iterations
}

// CalculateOffset is a function that calculates the offset for the given iteration
func CalculateOffset(iteration uint32) (offset uint32) {
	// Calculate the offset for the given iteration
	offset = iteration * ReviewLimit
	return offset
}
