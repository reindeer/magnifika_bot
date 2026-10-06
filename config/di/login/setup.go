//go:generate go tool diogen -output=config/diogen/login/di_gen.go setup.go

package login

import (
	"os"

	"gitlab.com/gorib/di"

	"github.com/reindeer/magnifika_bot/config/app"
	_ "github.com/reindeer/magnifika_bot/config/di"
	"github.com/reindeer/magnifika_bot/domain/bot/cmd/login"
)

func Setup() {
	di.Wire[app.Command](login.NewLogin, di.Args(map[string]any{"in": os.Stdin}))
}
