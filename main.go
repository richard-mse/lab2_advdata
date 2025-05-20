package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"lab2-advdata/graph"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const batchSize = 100

func ClearDatabase(ctx context.Context, session neo4j.SessionWithContext) error {
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `MATCH (n) DETACH DELETE n`, nil)
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
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	defer outFile.Close()

	scanner := bufio.NewScanner(inFile)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)
	writer := bufio.NewWriterSize(outFile, 64*1024)

	for scanner.Scan() {
		line := scanner.Text()
		sanitized := reNumberInt.ReplaceAllString(line, "$1")
		if _, err := writer.WriteString(sanitized + "\n"); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return writer.Flush()
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
	dec := json.NewDecoder(file)

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return fmt.Errorf("cannot create driver: %w", err)
	}
	defer driver.Close(context.Background())

	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return fmt.Errorf("neo4j unreachable: %w", err)
	}

	session := driver.NewSession(context.Background(), neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "neo4j",
	})
	defer session.Close(context.Background())

	// Purge la base
	if err := ClearDatabase(context.Background(), session); err != nil {
		return err
	}

	// Vérifier début de tableau JSON
	if tok, err := dec.Token(); err != nil || tok != json.Delim('[') {
		return fmt.Errorf("invalid JSON array start")
	}

	var (
		count int
		batch []map[string]interface{}
	)

	for dec.More() {
		if limit > 0 && count >= limit {
			break
		}
		// Variables pour champs désirés
		var (
			id         string
			title      string
			authors    []map[string]interface{}
			references []string
		)
		// Début de l'objet article
		if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
			return fmt.Errorf("expected object start, got %v, err: %w", tok, err)
		}
		// Lecture champ par champ
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return err
			}
			key := keyTok.(string)
			switch key {
			case "_id":
				if err := dec.Decode(&id); err != nil {
					return err
				}
			case "title":
				if err := dec.Decode(&title); err != nil {
					return err
				}
			case "authors":
				// Struct temporaire pour auteurs
				var tmp []struct {
					ID   string `json:"_id"`
					Name string `json:"name"`
				}
				if err := dec.Decode(&tmp); err != nil {
					return err
				}
				authors = make([]map[string]interface{}, len(tmp))
				for i, a := range tmp {
					authors[i] = map[string]interface{}{"id": a.ID, "name": a.Name}
				}
			case "references":
				if err := dec.Decode(&references); err != nil {
					return err
				}
			default:
				// Ignorer les champs non utilisés
				var skip json.RawMessage
				if err := dec.Decode(&skip); err != nil {
					return err
				}
			}
		}
		// Fin de l'objet
		if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
			return fmt.Errorf("expected object end, got %v, err: %w", tok, err)
		}

		// Ajouter au batch
		batch = append(batch, map[string]interface{}{
			"id":         id,
			"title":      title,
			"authors":    authors,
			"references": references,
		})
		count++

		if len(batch) == batchSize {
			if err := graph.CreateGraphFromRawArticles(context.Background(), session, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	// Fin du tableau JSON
	if tok, err := dec.Token(); err != nil || tok != json.Delim(']') {
		return fmt.Errorf("invalid JSON array end")
	}
	// Dernier batch
	if len(batch) > 0 {
		if err := graph.CreateGraphFromRawArticles(context.Background(), session, batch); err != nil {
			return err
		}
	}

	fmt.Printf("%d articles inserted.\n", count)
	return nil
}

func main() {
	start := time.Now()
	limit := 100

	if err := sanitizeMongoJSON("data/unsanitized.json", "data/sanitized.json"); err != nil {
		log.Fatal(err)
	}
	step := time.Since(start)
	fmt.Printf("Sanitization time: %.2f seconds\n", step.Seconds())

	err := decodeAndSend(limit)
	if err != nil {
		log.Fatal(err)
	}

	step2 := time.Since(start)
	duration := time.Since(start)

	fmt.Printf("Population time: %.2f seconds\n", step2.Seconds()-step.Seconds())
	fmt.Printf("Execution time: %.2f seconds\n", duration.Seconds())
}
