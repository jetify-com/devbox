package recommenders

type Recommender interface {
	IsRelevant() bool
	Packages() []string
}
