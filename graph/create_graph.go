package graph

import (
	"context"
	"lab2-advdata/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CreateArticleInGraph insère/met à jour un Article, ses Authors et ses References
// dans une seule transaction Cypher optimisée.
func CreateArticleInGraph(ctx context.Context, session neo4j.SessionWithContext, article models.Article) error {
	// Préparer la liste d'auteurs pour le paramètre UNWIND
	authorsParam := make([]map[string]any, len(article.Authors))
	for i, au := range article.Authors {
		authorsParam[i] = map[string]any{
			"id":   au.ID,
			"name": au.Name,
		}
	}

	// Cypher unique
	const query = `
MERGE (a:Article {_id: $id})
SET   a.title = $title

WITH a
UNWIND $authors AS aut
  MERGE (au:Author {_id: aut.id})
  SET   au.name = aut.name
  MERGE (au)-[:AUTHORED]->(a)

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

	// Un seul ExecuteWrite pour tout
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, params)
	})
	return err
}
