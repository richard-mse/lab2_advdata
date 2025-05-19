package graph

import (
	"context"
	"lab2-advdata/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CreateArticleInGraph inserts/updates an Article node, its authors and its citations
// in a SINGLE transaction & query.
func CreateArticleInGraph(ctx context.Context, session neo4j.SessionWithContext, article models.Article) error {

	// Build the list-of-maps parameter expected by Cypher
	authorsParam := make([]map[string]any, len(article.Authors))
	for i, au := range article.Authors {
		authorsParam[i] = map[string]any{
			"id":   au.ID,
			"name": au.Name,
		}
	}

	query := `
MERGE (a:Article {_id: $id})
SET   a.title = $title

// ---- authors ---------------------------------------------------------------
WITH a
UNWIND $authors AS aut
  MERGE (au:Author {_id: aut.id})
  SET   au.name = aut.name
  MERGE (au)-[:AUTHORED]->(a)

// ---- cited articles --------------------------------------------------------
WITH a
UNWIND $refs AS refId
  MERGE (r:Article {_id: refId})
  MERGE (a)-[:CITES]->(r)
`

	params := map[string]any{
		"id":      article.ID,
		"title":   article.Title,
		"authors": authorsParam,
		"refs":    article.References,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}
