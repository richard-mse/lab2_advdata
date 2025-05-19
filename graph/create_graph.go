package graph

import (
	"context"
	"lab2-advdata/models"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func CreateArticlesBatchInGraph(ctx context.Context, session neo4j.SessionWithContext, articles []models.Article) error {
	// Prépare la liste de maps pour UNWIND articles,
	// avec authors et refs imbriqués.
	articlesParam := make([]map[string]any, len(articles))
	for i, art := range articles {
		// transforme []Author en []map[string]any
		authList := make([]map[string]any, len(art.Authors))
		for j, au := range art.Authors {
			authList[j] = map[string]any{
				"id":   au.ID,
				"name": au.Name,
			}
		}
		articlesParam[i] = map[string]any{
			"id":      art.ID,
			"title":   art.Title,
			"authors": authList,
			"refs":    art.References,
		}
	}

	const query = `
UNWIND $articles AS doc
  MERGE (a:Article { _id: doc.id })
  SET   a.title = doc.title
  WITH a, doc
  UNWIND doc.authors AS aut
    MERGE (au:Author { _id: aut.id })
    SET   au.name = aut.name
    MERGE (au)-[:AUTHORED]->(a)
  WITH a, doc
  UNWIND doc.refs AS refId
    MERGE (r:Article { _id: refId })
    MERGE (a)-[:CITES]->(r)
`

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]any{"articles": articlesParam})
	})
	return err
}
