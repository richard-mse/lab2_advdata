package graph

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const cypher = `
UNWIND $batch AS doc
MERGE (a:Article {_id: doc.id})
  ON CREATE SET a.title = doc.title
  ON MATCH  SET a.title = coalesce(a.title, doc.title)
CALL {
  WITH a, doc
  UNWIND doc.authors AS aut
  MERGE (au:Author {_id: aut.id})
    ON CREATE SET au.name = aut.name
    ON MATCH  SET au.name = coalesce(au.name, aut.name)
  MERGE (au)-[:AUTHORED]->(a)
}
CALL {
  WITH a, doc
  UNWIND doc.references AS refId
  MERGE (r:Article {_id: refId})
  MERGE (a)-[:CITES]->(r)
}
`

func CreateArticlesBatchInGraph(
	ctx context.Context,
	session neo4j.SessionWithContext,
	batch []map[string]interface{},
) error {

	if len(batch) == 0 {
		return nil
	}

	// recommandation : 500–2 000 docs/batch
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, cypher, map[string]interface{}{"batch": batch})
	})
	return err
}
