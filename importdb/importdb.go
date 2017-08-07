package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
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

	elasticURL := flag.String("elastic", "http://localhost:9200", "Elastic Search URL.")
	elasticIndex := flag.String("index", "magnetico", "Name of the ElasticSearch Index.")
	deleteElasticIndex := flag.Bool("deleteIndex", false, "Delete existing ElasticSearch index.")
	databasePath := flag.String("db", "database.sqlite3", "Path of the Magnetico SQLite Database.")

	flag.Parse()

	if _, err := os.Stat(*databasePath); os.IsNotExist(err) {
		log.Fatalln("The database doesn't exist. Please check its path.")
	}

	elaContext := context.Background()
	ela, err := elastic.NewClient(elastic.SetURL(*elasticURL))
	if err != nil {
		log.Fatal(err)
	}

	shouldCreateIndex := false
	if *deleteElasticIndex {
		ela.DeleteIndex(*elasticIndex).Do(elaContext)
		shouldCreateIndex = true

	} else {
		exists, err := ela.IndexExists(*elasticIndex).Do(elaContext)
		if err != nil {
			log.Fatal(err)
		}
		shouldCreateIndex = !exists
	}

	if shouldCreateIndex {
		created, err := ela.CreateIndex(*elasticIndex).BodyString(
			`{
	"settings":{
		"number_of_shards":1,
		"number_of_replicas":0,
		"index":{
			"analysis":{
				"analyzer":{
					"magnetico_analyser":{
						"tokenizer":"pattern",
						"filter":"lowercase"
					}
				}
			}
		}
	},
	"mappings":{
		"torrent":{
			"properties":{
				"hash":{
					"type":"string"
				},
				"name":{
					"type":"text",
					"analyzer": "magnetico_analyser"
				},
				"files": {
					"type": "nested"
				},
				"date": {
					"type": "date"
				},
				"size": {
					"type": "long"
				}
			}
		}
	}
}`).Do(elaContext)
		if err != nil {
			log.Fatal(err)
		}
		if !created.Acknowledged {
			log.Fatalln("Index creation not acknowledged")
		}
	}

	elaBulk, err := ela.BulkProcessor().Name("ImportDBBulk-1").Workers(3).Do(elaContext)
	if err != nil {
		log.Fatal(err)
	}
	defer elaBulk.Close()

	db, err := sql.Open("sqlite3", *databasePath)
	if err != nil {
		log.Fatal(err)
	}
	// crash when debug on windows
	//defer db.Close()

	fmt.Println("Loading torrents files")
	files, err := db.Query("select torrent_id, size, path from files")
	if err != nil {
		log.Fatal(err)
	}
	defer files.Close()

	filesMap := make(map[int][]FileType)

	for files.Next() {
		var torrentID int
		var currentFile FileType
		err := files.Scan(&torrentID, &currentFile.Size, &currentFile.Path)
		if err != nil {
			log.Fatal(err)
		}

		torrentFiles, exist := filesMap[torrentID]
		if !exist {
			torrentFiles = make([]FileType, 0)
			filesMap[torrentID] = torrentFiles
		}
		filesMap[torrentID] = append(torrentFiles, currentFile)
		//filesMap[torrentID] = currentFile
		//fmt.Printf("%vpath: %v\n", torrentID, path)
	}

	fmt.Println("Loading torrents")
	torrents, err := db.Query("select id, lower(hex(info_hash)), name, total_size, discovered_on from torrents")
	if err != nil {
		log.Fatal(err)
	}
	defer torrents.Close()

	for torrents.Next() {
		var torrentID int
		var currentTorrent TorrentType
		err := torrents.Scan(&torrentID, &currentTorrent.Hash, &currentTorrent.Name, &currentTorrent.Size, &currentTorrent.Date)
		if err != nil {
			log.Fatal(err)
		}
		attachedFiles, hasFiles := filesMap[torrentID]
		if hasFiles {
			currentTorrent.Files = attachedFiles
		}
		fmt.Printf("%v: %v file(s)\n", currentTorrent.Hash, len(currentTorrent.Files))
		r := elastic.NewBulkIndexRequest().Index(*elasticIndex).Type("torrent").Id(currentTorrent.Hash).Doc(currentTorrent)
		elaBulk.Add(r)
	}

	fmt.Println("Done")
}
