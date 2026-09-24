package webscanner

import (
	"context"
	"net/http"

	"jobhub-ai/backend/internal/providers"
	"jobhub-ai/backend/internal/providers/greenhouse"
)

// ATSAdapter is the narrow bridge from a detected provider key to JobHub's
// provider-neutral result. Adapters collect; they never persist vacancies.
type ATSAdapter interface {
	Collect(context.Context, string, int) (providers.Result, error)
}
type adapterRegistry map[string]ATSAdapter

func defaultRegistry(client *http.Client) adapterRegistry {
	return adapterRegistry{"greenhouse": greenhouseAdapter{client: client}}
}

type greenhouseAdapter struct{ client *http.Client }

func (g greenhouseAdapter) Collect(ctx context.Context, key string, limit int) (providers.Result, error) {
	budget, err := greenhouse.NewBudget(1, limit)
	if err != nil {
		return providers.Result{}, err
	}
	adapter, err := greenhouse.NewClient(key, budget, g.client)
	if err != nil {
		return providers.Result{}, err
	}
	return adapter.Collect(ctx)
}
