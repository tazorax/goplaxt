package trakt

import (
	"github.com/stretchr/testify/require"
	"github.com/xanderstrike/plexhooks"
	"testing"
)

func TestFindEpisode(t *testing.T) {

	type testCase struct {
		Name            string
		Payload         plexhooks.PlexResponse
		ExpectedEpisode Episode
	}

	var cases = []testCase{
		{
			Name: "Severance S02E03 with tvdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://episode/669013e833d03eeac35f5d09",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "tvdb://10592760",
						},
						{
							Id: "tmdb://5469117",
						},
						{
							Id: "imdb://tt15241840",
						},
					},
				},
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Severance S02E03 with tmdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://episode/669013e833d03eeac35f5d09",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "tmdb://5469117",
						},
						{
							Id: "imdb://tt15241840",
						},
					},
				},
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Severance S02E03 with imdb",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid: "plex://episode/669013e833d03eeac35f5d09",
					ExternalGuid: []plexhooks.ExternalGuid{
						{
							Id: "imdb://tt15241840",
						},
					},
				},
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "Severance S02E03 with title search",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid:             "plex://episode/669013e833d03eeac35f5d09",
					GrandparentTitle: "Severance",
					Index:            3,
					ParentIndex:      2,
					Year:             2022,
				},
			},
			ExpectedEpisode: Episode{
				Title:  "Who Is Alive?",
				Season: 2,
				Number: 3,
				Ids: Ids{
					Trakt:  12103029,
					Tvdb:   10592760,
					Imdb:   "tt15241840",
					Tmdb:   5469117,
					Tvrage: 0,
				},
			},
		},
		{
			Name: "37 secondes S01E05 with title",
			Payload: plexhooks.PlexResponse{
				Metadata: plexhooks.Metadata{
					Guid:             "plex://episode/669013e833d03eeac35f5d09",
					GrandparentTitle: "37 secondes",
					Index:            5,
					ParentIndex:      1,
					Year:             2025,
					Title:            "",
				},
			},
			ExpectedEpisode: Episode{
				Title:  "Episode 5",
				Season: 1,
				Number: 5,
				Ids: Ids{
					Trakt:  11722092,
					Tvdb:   0,
					Imdb:   "",
					Tmdb:   5322130,
					Tvrage: 0,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			episode := findEpisode(c.Payload)
			require.Equal(t, c.ExpectedEpisode.Title, episode.Title, "Title mismatch")
			require.Equal(t, c.ExpectedEpisode.Season, episode.Season, "Season mismatch")
			require.Equal(t, c.ExpectedEpisode.Number, episode.Number, "Number mismatch")
			require.Equal(t, c.ExpectedEpisode.Ids, episode.Ids, "Ids mismatch")
		})
	}
}
