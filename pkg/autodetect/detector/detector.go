package detector

import "context"

type Detector interface {
	Relevance(path string) (float64, error)
	Packages(ctx context.Context) ([]string, error)
}
