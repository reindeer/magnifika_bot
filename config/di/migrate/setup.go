//go:generate go tool diogen -output=config/diogen/migrate/di_gen.go setup.go

package migrate

import (
	"gitlab.com/gorib/di"

	"github.com/reindeer/magnifika_bot/config/app"
	_ "github.com/reindeer/magnifika_bot/config/di"
	"github.com/reindeer/magnifika_bot/domain/bot/cmd/migrate"
)

func Setup() {
	di.Wire[app.Command](migrate.NewMigrator)
}
