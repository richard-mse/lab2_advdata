package graph

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const cypher = `
UNWIND $batch AS doc

MERGE (a:Article {_id: doc.id})
  ON CREATE SET a.title = doc.title
WITH a, doc

CALL {
  WITH a, doc
  UNWIND doc.authors AS aut
  MERGE (au:Author {_id: aut.id})
    ON CREATE SET au.name = aut.name
  MERGE (au)-[:AUTHORED]->(a)
}
WITH a, doc

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

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, cypher, map[string]interface{}{"batch": batch})
	})

	return err
}

func EnsureArticleIndex(ctx context.Context, session neo4j.SessionWithContext) error {
	_, err := session.Run(ctx, `
		CREATE RANGE INDEX article_id IF NOT EXISTS
		FOR (a:Article) ON (a._id)`, nil)
	return err
}

func EnsureAuthorIndex(ctx context.Context, session neo4j.SessionWithContext) error {
	_, err := session.Run(ctx, `
		CREATE RANGE INDEX author_id IF NOT EXISTS
		FOR (au:Author) ON (au._id)`, nil)
	return err
}
