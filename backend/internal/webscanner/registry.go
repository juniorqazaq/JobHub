package webscanner

import (
	"context"
	"net/http"

	"jobhub-ai/backend/internal/providers"
	"jobhub-ai/backend/internal/providers/airastana"
	"jobhub-ai/backend/internal/providers/greenhouse"
	"jobhub-ai/backend/internal/providers/kcell"
)

// ATSAdapter is the narrow bridge from a detected provider key to JobHub's
// provider-neutral result. Adapters collect; they never persist vacancies.
type ATSAdapter interface {
	Collect(context.Context, string, int) (providers.Result, error)
}
type adapterRegistry map[string]ATSAdapter

func defaultRegistry(client *http.Client) adapterRegistry {
	return adapterRegistry{"greenhouse": greenhouseAdapter{client: client}, "kcell": kcellAdapter{client: client}, "airastana": airAstanaAdapter{client: client}}
}

type kcellAdapter struct{ client *http.Client }

func (k kcellAdapter) Collect(ctx context.Context, _ string, limit int) (providers.Result, error) {
	c, err := kcell.NewClient(5, limit, k.client)
	if err != nil {
		return providers.Result{}, err
	}
	return c.Collect(ctx)
}

type airAstanaAdapter struct{ client *http.Client }

func (a airAstanaAdapter) Collect(ctx context.Context, _ string, limit int) (providers.Result, error) {
	c, err := airastana.NewClient(10, limit, a.client)
	if err != nil {
		return providers.Result{}, err
	}
	return c.Collect(ctx)
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
