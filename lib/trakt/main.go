package trakt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/viscerous/goplaxt/lib/config"
	"github.com/viscerous/goplaxt/lib/store"
	"github.com/xanderstrike/plexhooks"
)

// AuthRequest authorize the connection with Trakt
func AuthRequest(root, username, code, refreshToken, grantType string) (map[string]interface{}, error) {
	values := map[string]string{
		"code":          code,
		"refresh_token": refreshToken,
		"client_id":     config.TraktClientId,
		"client_secret": config.TraktClientSecret,
		"redirect_uri":  fmt.Sprintf("%s/authorize?username=%s", root, url.PathEscape(username)),
		"grant_type":    grantType,
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := http.Post("https://api.trakt.tv/oauth/token", "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Handle determine if an item is a show or a movie
func Handle(pr plexhooks.PlexResponse, user store.User) {
	if pr.Metadata.LibrarySectionType == "show" {
		HandleShow(pr, user.AccessToken)
	} else if pr.Metadata.LibrarySectionType == "movie" {
		HandleMovie(pr, user.AccessToken)
	}
	log.Print("Event logged")
}

// HandleShow start the scrobbling for a show
func HandleShow(pr plexhooks.PlexResponse, accessToken string) {
	event, progress := getAction(pr)

	scrobbleObject := ShowScrobbleBody{
		Progress: progress,
		Episode:  findEpisode(pr),
	}

	scrobbleJSON, err := json.Marshal(scrobbleObject)
	handleErr(err)

	scrobbleRequest(event, scrobbleJSON, accessToken)
}

// HandleMovie start the scrobbling for a movie
func HandleMovie(pr plexhooks.PlexResponse, accessToken string) {
	event, progress := getAction(pr)

	scrobbleObject := MovieScrobbleBody{
		Progress: progress,
		Movie:    findMovie(pr),
	}

	scrobbleJSON, _ := json.Marshal(scrobbleObject)

	scrobbleRequest(event, scrobbleJSON, accessToken)
}

func findEpisode(pr plexhooks.PlexResponse) Episode {
	var traktService = "tvdb"
	var showID []string

	re := regexp.MustCompile(`tvdb(?:://|[2-5]?-)(\d*)/(\d*)/(\d*)`)
	showID = re.FindStringSubmatch(pr.Metadata.Guid)

	// Retry with TheMovieDB
	if showID == nil {
		re := regexp.MustCompile(`themoviedb://(\d*)/(\d*)/(\d*)`)
		showID = re.FindStringSubmatch(pr.Metadata.Guid)
		traktService = "tmdb"
	}

	// Retry with the new Plex TV agent
	if showID == nil {
		var episodeID string

		log.Println("Finding episode with new Plex TV agent")

		traktService = pr.Metadata.ExternalGuid[0].Id[:4]
		episodeID = pr.Metadata.ExternalGuid[0].Id[7:]

		// The new Plex TV agent use episode ID instead of show ID,
		// so we need to do things a bit differently
		URL := fmt.Sprintf("https://api.trakt.tv/search/%s/%s?type=episode", traktService, episodeID)

		respBody := makeRequest(URL)

		var showInfo []ShowInfo
		err := json.Unmarshal(respBody, &showInfo)
		handleErr(err)

		log.Printf("Tracking %s - S%02dE%02d using %s", showInfo[0].Show.Title, showInfo[0].Episode.Season, showInfo[0].Episode.Number, traktService)

		return showInfo[0].Episode
	}

	url := fmt.Sprintf("https://api.trakt.tv/search/%s/%s?type=show", traktService, showID[1])

	log.Printf("Finding show for %s %s %s using %s", showID[1], showID[2], showID[3], traktService)

	respBody := makeRequest(url)

	var showInfo []ShowInfo
	err := json.Unmarshal(respBody, &showInfo)
	handleErr(err)

	url = fmt.Sprintf("https://api.trakt.tv/shows/%d/seasons?extended=episodes", showInfo[0].Show.Ids.Trakt)

	respBody = makeRequest(url)
	var seasons []Season
	err = json.Unmarshal(respBody, &seasons)
	handleErr(err)

	for _, season := range seasons {
		if fmt.Sprintf("%d", season.Number) == showID[2] {
			for _, episode := range season.Episodes {
				if fmt.Sprintf("%d", episode.Number) == showID[3] {
					return episode
				}
			}
		}
	}

	panic("Could not find episode!")
}

func findMovie(pr plexhooks.PlexResponse) Movie {
	log.Printf("Finding movie for %s (%d)", pr.Metadata.Title, pr.Metadata.Year)
	url := fmt.Sprintf("https://api.trakt.tv/search/movie?query=%s", url.PathEscape(pr.Metadata.Title))

	respBody := makeRequest(url)

	var results []MovieSearchResult

	err := json.Unmarshal(respBody, &results)
	handleErr(err)

	for _, result := range results {
		if result.Movie.Year == pr.Metadata.Year {
			return result.Movie
		}
	}
	panic("Could not find movie!")
}

func makeRequest(url string) []byte {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	handleErr(err)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", config.TraktClientId)

	resp, err := client.Do(req)
	handleErr(err)
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	return respBody
}

func scrobbleRequest(action string, body []byte, accessToken string) []byte {
	client := &http.Client{}

	url := fmt.Sprintf("https://api.trakt.tv/scrobble/%s", action)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	handleErr(err)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Add("trakt-api-version", "2")
	req.Header.Add("trakt-api-key", config.TraktClientId)

	resp, err := client.Do(req)
	if err != nil {
		handleErr(err)
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)

	return respBody
}

func getAction(pr plexhooks.PlexResponse) (string, int) {
	switch pr.Event {
	case "media.play":
		return "start", 0
	case "media.pause":
		return "stop", 0
	case "media.resume":
		return "start", 0
	case "media.stop":
		return "stop", 0
	case "media.scrobble":
		return "stop", 90
	}
	return "", 0
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
