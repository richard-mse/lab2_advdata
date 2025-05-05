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
		log.Printf("Fail clean database : %v", err)
		return err
	}

	log.Println("Clean database.")
	return nil
}

func main() {

	const PATH_FILE = "data/test.json"

	raw_file_byte, err := os.ReadFile(PATH_FILE)
	if err != nil {
		log.Fatalf("Erreur lecture fichier JSON: %v", err)
	}

	var articles []models.Article
	if err := json.Unmarshal(raw_file_byte, &articles); err != nil {
		log.Fatalf("Erreur parsing JSON: %v", err)
	}

	uri := "neo4j://localhost:7687"
	username := "neo4j"
	password := "testtest"

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Erreur driver Neo4j: %v", err)
	}
	defer driver.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "neo4j",
	})

	ClearDatabase(ctx, session)

	graph.CreateGraphFromArticles(ctx, session, articles)

	defer session.Close(ctx)
	fmt.Println("Session connexion successfully.")
}
