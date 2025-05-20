package graph

import (
	"context"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CreateArticlesBatchInGraph crée/maj plusieurs articles, authors et références
// en une seule transaction Cypher optimisée.
func CreateGraphFromRawArticles(ctx context.Context, session neo4j.SessionWithContext, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		log.Println("No articles to process.")
		return nil
	}

	// Single write transaction for the whole batch
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
UNWIND $batch AS article
// Merge Article node and set title
MERGE (a:Article {_id: article.id})
SET a.title = article.title

// Merge each Author and create AUTHORED relationship
WITH article, a
UNWIND article.authors AS au
MERGE (author:Author {_id: au.id})
SET author.name = au.name
MERGE (author)-[:AUTHORED]->(a)

// Create CITES relationships for references
WITH article, a
UNWIND article.references AS refId
MERGE (ref:Article {_id: refId})
MERGE (a)-[:CITES]->(ref)
`
		params := map[string]interface{}{"batch": batch}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	if err != nil {
		log.Printf("Batch write failed: %v", err)
		return err
	}

	log.Println("Graph creation for batch completed.")
	return nil
}
