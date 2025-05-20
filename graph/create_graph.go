package graph

import (
	"context"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func CreateArticlesBatchInGraph(ctx context.Context, session neo4j.SessionWithContext, batch []map[string]interface{}) error {
	if len(batch) == 0 {
		log.Println("No articles to process.")
		return nil
	}

	const query = `
	UNWIND $batch AS doc
	MERGE (a:Article {_id: doc.id})
	SET a.title = doc.title

	FOREACH (aut IN doc.authors |
	MERGE (au:Author {_id: aut.id})
	SET au.name = aut.name
	MERGE (au)-[:AUTHORED]->(a)
	)

	FOREACH (refId IN doc.references |
	MERGE (r:Article {_id: refId})
	MERGE (a)-[:CITES]->(r)
	)
	`
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"batch": batch})
	})
	if err != nil {
		log.Printf("Batch write failed: %v", err)
		return err
	}

	log.Println("Graph creation for batch completed.")
	return nil
}
