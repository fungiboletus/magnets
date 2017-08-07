package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"encoding/json"

	"regexp"

	"github.com/gin-gonic/gin"
	elastic "gopkg.in/olivere/elastic.v5"
)

// FileType is a simple file in magnetico
type FileType struct {
	Size int    `json:"size"`
	Path string `json:"path"`
}

// TorrentType is a torrent in magnetico
type TorrentType struct {
	Hash  string     `json:"hash"`
	Name  string     `json:"name"`
	Date  int        `json:"date"`
	Size  int        `json:"size"`
	Files []FileType `json:"files"`
}

func main() {

	elasticURL := os.Getenv("ELASTIC_URL")
	httpAddr := os.Getenv("HTTP_ADDR")
	elasticIndex := os.Getenv("ELASTIC_INDEX")
	maxFilesPerTorrent, _ := strconv.Atoi(os.Getenv("MAX_FILES_PER_TORRENT"))
	staticFolder := os.Getenv("STATIC_FOLDER")

	// This is so ugly
	// Would like to do toto := stuff()
	if elasticURL == "" {
		elasticURL = "http://localhost:9200"
	}
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	if elasticIndex == "" {
		elasticIndex = "magnetico"
	}
	if maxFilesPerTorrent == 0 {
		maxFilesPerTorrent = 50
	}
	if staticFolder == "" {
		staticFolder = "./static/"
	}

	app := gin.Default()

	app.StaticFile("/", staticFolder+"index.html")
	app.StaticFile("/favicon.ico", staticFolder+"favicon.ico")
	app.StaticFile("/style.css", staticFolder+"style.css")
	app.StaticFile("/script.js", staticFolder+"script.js")

	// Elastic search client initialization
	elaContext := context.Background()
	ela, err := elastic.NewClient(elastic.SetURL(elasticURL))

	if err != nil {
		log.Fatal(err)
	}

	// The search query is splitted in terms
	searchTermsSplit := regexp.MustCompile("[\\s\\-\\.\\[\\]\\_]+")

	// Some parts of the search query should not have fuzziness, such as standard formats
	// or identifiers
	searchTermsMatch := regexp.MustCompile("^(s\\d+e\\d+|mp3|mkv|flac|1080p|720p|x264|x265|\\+.*|\".*?\"|\\d+)$")

	// Simple search
	app.GET("/search", func(c *gin.Context) {
		query := c.Query("q") // GET parameter q

		// Split the query in terms
		queryTerms := searchTermsSplit.Split(query, 10)

		// Construction of the query
		elasticQuery := elastic.NewBoolQuery()

		// The query has a fuzzy part meaning that not all the terms must be present
		elasticFuzzyQueryPart := elastic.NewBoolQuery()
		elasticQuery.Must(elasticFuzzyQueryPart)

		// This magic value should perhaps be updated after some tests
		elasticFuzzyQueryPart.MinimumShouldMatch("75%")

		// For each term
		for _, term := range queryTerms {
			// The search database is indexed on lowercase text
			lTerm := strings.TrimSpace(strings.ToLower(term))
			if len(lTerm) == 0 {
				continue
			}

			// If the term doesn't require fuzziness
			if searchTermsMatch.MatchString(lTerm) {

				disMaxTerm := elastic.NewDisMaxQuery().
					Query(elastic.NewMatchQuery("name", lTerm).Boost(4)).
					Query(elastic.NewMatchQuery("files", lTerm)).
					TieBreaker(0.3)
				elasticQuery.Must(disMaxTerm)

			} else {

				// If it's a fuzzy query
				disMaxTerm := elastic.NewDisMaxQuery().
					Query(elastic.NewMatchQuery("name", lTerm).Boost(4).Fuzziness("AUTO")).
					Query(elastic.NewMatchQuery("files", lTerm).Fuzziness("AUTO")).
					TieBreaker(0.3)

				// Notice the should instead of must
				elasticFuzzyQueryPart.Should(disMaxTerm)
			}
		}

		// Run the search query
		searchResult, err := ela.Search().
			Index(elasticIndex).
			Query(elasticQuery).
			From(0).Size(50). // fixed for some reasons
			Do(elaContext)

		if err != nil {
			// Handle error
			log.Fatal(err)
		}

		// Extract the results to a  new list
		torrents := make([]TorrentType, len(searchResult.Hits.Hits))

		for i, hit := range searchResult.Hits.Hits {
			var torrent TorrentType
			err := json.Unmarshal(*hit.Source, &torrent)
			if err == nil {
				// Some torrents have many files
				// We limit to 50 by default
				if len(torrent.Files) > maxFilesPerTorrent {
					torrent.Files = torrent.Files[:maxFilesPerTorrent]
				}
				torrents[i] = torrent
			}
			/*score := *hit.Score
			fmt.Printf("%v\n", score)*/
		}

		// Return the results
		c.JSON(200, gin.H{
			"totalHits": searchResult.TotalHits(),
			"results":   torrents,
		})
	})

	app.GET("/count", func(c *gin.Context) {
		count, err := ela.Count(elasticIndex).Do(elaContext)
		if err != nil {
			log.Fatal(err)
		}
		c.JSON(200, gin.H{
			"count": count,
		})
	})

	app.Run(httpAddr)
}
