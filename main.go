package main

import (
	"context"
	"encoding/json"
	"fmt"
	"lab2-advdata/graph"
	"lab2-advdata/models"
	"log"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func ClearDatabase(ctx context.Context, session neo4j.SessionWithContext) error {
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `MATCH (n) DETACH DELETE n`
		_, err := tx.Run(ctx, query, nil)
		return nil, err
	})
	if err != nil {
		log.Printf("Fail clean database: %v", err)
		return err
	}
	log.Println("Clean database.")
	return nil
}

func main() {
	const PATH_FILE = "data/biggertest.json"
	file, err := os.Open(PATH_FILE)

	if err != nil {
		log.Fatalf("Erreur lecture fichier JSON: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	uri := "neo4j://neo4j:7687"
	username := "neo4j"
	password := "testtest"

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Erreur driver Neo4j: %v", err)
	}
	defer driver.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "neo4j",
	})
	defer session.Close(ctx)

	if err := ClearDatabase(ctx, session); err != nil {
		log.Fatal("Erreur nettoyage base")
	}

	t, err := decoder.Token()
	if err != nil || t != json.Delim('[') {
		log.Fatalf("Format JSON invalide")
	}

	count := 0
	for decoder.More() {
		var article models.Article
		if err := decoder.Decode(&article); err != nil {
			log.Fatalf("Erreur parsing article: %v", err)
		}
		graph.CreateArticleInGraph(ctx, session, article)
		count++
	}

	fmt.Printf("Insertion terminée. %d articles traités.\n", count)
}
