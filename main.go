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
	start := time.Now()
	limit := flag.Int("limit", -1, "Nombre maximal d'articles à insérer (par défaut : tous)")
	flag.Parse()

	// 📥 Lire les variables d'environnement
	dataURL := os.Getenv("DATA_URL")
	if dataURL == "" {
		log.Fatal("DATA_URL non défini")
	}

	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://neo4j:7687"
	}
	username := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")

	resp, err := http.Get(dataURL)
	if err != nil {
		log.Fatalf("Erreur requête GET JSON: %v", err)
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var driver neo4j.DriverWithContext
	const maxAttempts = 5
	for i := 1; i <= maxAttempts; i++ {
		driver, err = neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
		if err == nil {
			err = driver.VerifyConnectivity(context.Background())
			if err == nil {
				break
			}
		}
		log.Printf("Tentative %d: échec de connexion à Neo4j : %v", i, err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		log.Fatalf("Impossible de se connecter à Neo4j après %d tentatives: %v", maxAttempts, err)
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
	duration := time.Since(start)
	fmt.Printf("Execution time: %.2f seconds\n", duration.Seconds())
	fmt.Printf("%d articles inserted.\n", count)
}
