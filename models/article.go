package models

type Article struct {
	ID         string   `json:"_id"`
	Title      string   `json:"title"`
	Authors    []Author `json:"authors"`
	References []string `json:"references"`
}
