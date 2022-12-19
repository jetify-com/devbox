package suggestors

type Suggestor interface {
	IsRelevant(srcDir string) bool
	Packages() []string
}
