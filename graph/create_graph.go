package graph

import (
	"context"
	"lab2-advdata/models"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func CreateArticleInGraph(ctx context.Context, session neo4j.SessionWithContext, article models.Article) {
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MERGE (a:Article {_id: $id}) SET a.title = $title`
		params := map[string]any{
			"id":    article.ID,
			"title": article.Title,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	if err != nil {
		log.Printf("Article %s: %v", article.ID, err)
		return
	}

	for _, author := range article.Authors {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MERGE (au:Author {_id: $authorId})
				SET au.name = $name
				WITH au
				MATCH (ar:Article {_id: $articleId})
				MERGE (au)-[:AUTHORED]->(ar)
			`
			params := map[string]any{
				"authorId":  author.ID,
				"name":      author.Name,
				"articleId": article.ID,
			}
			_, err := tx.Run(ctx, query, params)
			return nil, err
		})
		if err != nil {
			log.Printf("Author %s: %v", author.ID, err)
		}
	}

	for _, refId := range article.References {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MATCH (src:Article {_id: $srcId})
				MERGE (tgt:Article {_id: $tgtId})
				MERGE (src)-[:CITES]->(tgt)
			`
			params := map[string]any{
				"srcId": article.ID,
				"tgtId": refId,
			}
			_, err := tx.Run(ctx, query, params)
			return nil, err
		})
		if err != nil {
			log.Printf("CITES %s -> %s: %v", article.ID, refId, err)
		}
	}
}
