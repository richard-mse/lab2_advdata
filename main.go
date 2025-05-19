package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"lab2-advdata/graph"
	"lab2-advdata/models"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const batchSize = 100

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

func sanitizeMongoJSON(inputPath, outputPath string) error {

	reNumberInt := regexp.MustCompile(`NumberInt\((\-?\d+)\)`)

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(inputFile)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	writer := bufio.NewWriterSize(outputFile, 64*1024) // Buffered writer (64KB)

	for scanner.Scan() {
		line := scanner.Text()
		sanitizedLine := reNumberInt.ReplaceAllString(line, "$1")
		_, err := writer.WriteString(sanitizedLine + "\n")
		if err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error while scanning input file: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("error while flushing output file: %w", err)
	}

	return nil
}

func decodeAndSend(limit int) error {
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://graphdb:7687"
	}
	username := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")

	file, err := os.Open("data/sanitized.json")
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return fmt.Errorf("cannot create driver: %w", err)
	}
	defer driver.Close(context.Background())

	// Vérif de connectivité
	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return fmt.Errorf("neo4j unreachable: %w", err)
	}

	session := driver.NewSession(context.Background(), neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "neo4j",
	})
	defer session.Close(context.Background())

	if err := ClearDatabase(context.Background(), session); err != nil {
		return err
	}

	// Lit le début du tableau JSON
	if tok, err := decoder.Token(); err != nil || tok != json.Delim('[') {
		return fmt.Errorf("invalid JSON array")
	}

	var batch []models.Article
	count := 0

	for decoder.More() {
		if limit >= 0 && count >= limit {
			break
		}
		var art models.Article
		if err := decoder.Decode(&art); err != nil {
			return err
		}
		batch = append(batch, art)
		count++

		if len(batch) >= batchSize {
			if err := graph.CreateArticlesBatchInGraph(context.Background(), session, batch); err != nil {
				return err
			}
			batch = batch[:0] // reset
		}
	}

	// envoyer le reste
	if len(batch) > 0 {
		if err := graph.CreateArticlesBatchInGraph(context.Background(), session, batch); err != nil {
			return err
		}
	}

	fmt.Printf("%d articles inserted.\n", count)
	return nil
}

func main() {
	start := time.Now()
	limit := 10
	flag.Parse()

	err := sanitizeMongoJSON("data/unsanitized.json", "data/sanitized.json")
	if err != nil {
		log.Fatal(err)
	}

	err = decodeAndSend(limit)
	if err != nil {
		log.Fatal(err)
	}

	duration := time.Since(start)
	fmt.Printf("Execution time: %.2f seconds\n", duration.Seconds())
}
