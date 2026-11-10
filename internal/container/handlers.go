package container

import (
	"time"

	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/health"
	"github.com/Artem-229/avito-lab/internal/http/ready"
	"github.com/Artem-229/avito-lab/internal/http/trip/create"
	"github.com/Artem-229/avito-lab/internal/http/trip/finish"
	"github.com/Artem-229/avito-lab/internal/http/trip/get"
)

type handlers struct {
	*healthHandler
	*readyHandler
	*createHandler
	*getHandler
	*finishHandler
}
type (
	healthHandler = health.Handler
	readyHandler  = ready.Handler
	createHandler = create.Handler
	getHandler    = get.Handler
	finishHandler = finish.Handler
)

var _ api.ServerInterface = (*handlers)(nil)

func newHandlers(repos *Repositories, readyTimeout time.Duration, usecases *Usecases) *handlers {
	return &handlers{
		healthHandler: health.New(),
		readyHandler:  ready.New(repos, readyTimeout),
		createHandler: create.New(usecases.CreateTrip),
		getHandler:    get.New(usecases.GetTrip),
		finishHandler: finish.New(usecases.FinishTrip),
	}
}
