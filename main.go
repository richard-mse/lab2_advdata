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
	username := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")

	file, err := os.Open("data/sanitized.json")
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	var driver neo4j.DriverWithContext
	driver, err = neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))

	if err != nil {
		return fmt.Errorf("cannot create Neo4j driver: %w", err)
	}
	defer driver.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "neo4j",
	})
	defer session.Close(ctx)

	if err := ClearDatabase(ctx, session); err != nil {
		return fmt.Errorf("Erreur nettoyage base")
	}

	t, err := decoder.Token()
	if err != nil || t != json.Delim('[') {
		fmt.Errorf("Format JSON invalide")
	}
	fmt.Println("hello world")
	count := 0
	for decoder.More() {
		if limit >= 0 && count >= limit {
			break
		}

		var article models.Article
		if err := decoder.Decode(&article); err != nil {
			return err
		}

		graph.CreateArticleInGraph(ctx, session, article)
		count++
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

	step := time.Since(start)
	fmt.Printf("Sanitization time: %.2f seconds\n", step.Seconds())

	err = decodeAndSend(limit)
	if err != nil {
		log.Fatal(err)
	}

	step2 := time.Since(start)
	duration := time.Since(start)

	fmt.Printf("Population time: %.2f seconds\n", step2.Seconds()-step.Seconds())
	fmt.Printf("Execution time: %.2f seconds\n", duration.Seconds())
}
