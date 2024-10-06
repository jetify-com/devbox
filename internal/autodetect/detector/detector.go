package detector

import "context"

type Detector interface {
	IsRelevant(path string) (float64, error)
	Packages(ctx context.Context) ([]string, error)
}
