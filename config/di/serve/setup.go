//go:generate go tool diogen -output=config/diogen/serve/di_gen.go setup.go

package serve

import (
	"gitlab.com/gorib/di"

	"github.com/reindeer/magnifika_bot/config/app"
	_ "github.com/reindeer/magnifika_bot/config/di"
	"github.com/reindeer/magnifika_bot/domain/bot/cmd/serve"
)

func Setup() {
	di.Wire[app.Command](serve.NewServer)
}
