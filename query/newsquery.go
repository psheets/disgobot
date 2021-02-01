package query

type ASource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NewsArticle struct {
	Source      ASource
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
}

type NewsResponse struct {
	Articles []NewsArticle `json:"articles"`
}

func NewsQuery(i int) {

}
