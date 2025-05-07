package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"lab2-advdata/graph"
	"lab2-advdata/models"
	"log"
	"net/http"
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
	limit := flag.Int("limit", -1, "Nombre maximal d'articles à insérer (par défaut : tous)")
	flag.Parse()

	DATA_URL := "http://vmrum.isc.heia-fr.ch/dblpv14.json"

	resp, err := http.Get(DATA_URL)
	if err != nil {
		log.Fatalf("Erreur requête GET JSON: %v", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	uri := "neo4j://localhost:7687"
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
		if *limit >= 0 && count >= *limit {
			break
		}

		var article models.Article
		if err := decoder.Decode(&article); err != nil {
			log.Fatalf("Erreur parsing article: %v", err)
		}

		graph.CreateArticleInGraph(ctx, session, article)
		count++
	}

	fmt.Printf("%d lines test inserted.\n", count)
}
