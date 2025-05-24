package main

import (
	"bufio"
	"context"
	"fmt"
	"lab2-advdata/graph"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/bcicen/jstream"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const batchSize = 1000

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

	ctx := context.Background()

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

	dec := jstream.NewDecoder(file, 1)

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return fmt.Errorf("cannot create driver: %w", err)
	}
	defer driver.Close(ctx)

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("neo4j unreachable: %w", err)
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: "neo4j",
	})
	defer session.Close(ctx)

	if err := ClearDatabase(ctx, session); err != nil {
		return err
	}
	log.Println("Clean database.")

	if err := graph.EnsureArticleIndex(ctx, session); err != nil {
		return err
	}
	log.Println("Create index article")

	if err := graph.EnsureAuthorIndex(ctx, session); err != nil {
		return err
	}
	log.Println("Create index author")

	var (
		count          int
		batch          []map[string]interface{}
		totalDuration  time.Duration
		batchExecCount int
	)

	// Parcourt chaque objet JSON du tableau
	for mv := range dec.Stream() {
		if limit > 0 && count >= limit {
			break
		}

		raw, ok := mv.Value.(map[string]interface{})
		if !ok {
			log.Println("Skipping non-object JSON value")
			continue
		}

		// Extraction des champs
		id, _ := raw["_id"].(string)
		title, _ := raw["title"].(string)

		// Authors
		authors := make([]map[string]interface{}, 0)
		if rawAuth, ok := raw["authors"].([]interface{}); ok {
			for _, ai := range rawAuth {
				if aMap, ok := ai.(map[string]interface{}); ok {
					idVal, hasID := aMap["_id"].(string)
					nameVal, hasName := aMap["name"].(string)
					if hasID && hasName && idVal != "" && nameVal != "" {
						authors = append(authors, map[string]interface{}{"id": idVal, "name": nameVal})
					}
				}
			}
		}

		// References
		refs := make([]string, 0)
		if rawRefs, ok := raw["references"].([]interface{}); ok {
			for _, ri := range rawRefs {
				if s, ok := ri.(string); ok {
					refs = append(refs, s)
				}
			}
		}

		batch = append(batch, map[string]interface{}{
			"id":         id,
			"title":      title,
			"authors":    authors,
			"references": refs,
		})
		count++

		if len(batch) == batchSize {
			start := time.Now()
			if err := graph.CreateArticlesBatchInGraph(ctx, session, batch); err != nil {
				return err
			}
			duration := time.Since(start)
			totalDuration += duration
			batchExecCount++
			batch = batch[:0]
			log.Println("Batch created")
		}
	}

	if len(batch) > 0 {
		start := time.Now()
		if err := graph.CreateArticlesBatchInGraph(ctx, session, batch); err != nil {
			return err
		}
		duration := time.Since(start)
		totalDuration += duration
		batchExecCount++
	}

	if batchExecCount > 0 {
		avg := totalDuration / time.Duration(batchExecCount)
		fmt.Printf("%d articles inserted in %d batches. Average batch execution time: %s\n", count, batchExecCount, avg)
	} else {
		fmt.Printf("%d articles inserted. No batch executed.\n", count)
	}

	return nil
}

func main() {
	start := time.Now()
	limit := -1

	if err := sanitizeMongoJSON("data/unsanitized.json", "data/sanitized.json"); err != nil {
		log.Fatal(err)
	}
	step := time.Since(start)

	err := decodeAndSend(limit)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Sanitization time: %.2f seconds\n", step.Seconds())
	duration := time.Since(start)
	fmt.Printf("Population time: %.2f seconds\n", duration.Seconds()-step.Seconds())
	fmt.Printf("TOtal execution time: %.2f seconds\n", duration.Seconds())
}
