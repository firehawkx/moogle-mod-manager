package remote

import (
	"github.com/kiamev/moogle-mod-manager/config"
	"github.com/kiamev/moogle-mod-manager/discover/remote/curseforge"
	"github.com/kiamev/moogle-mod-manager/discover/remote/nexus"
	"github.com/kiamev/moogle-mod-manager/discover/remote/util"
	"github.com/kiamev/moogle-mod-manager/mods"
)

type Client interface {
	GetFromMod(in *mods.Mod) (found bool, mod *mods.Mod, err error)
	GetFromID(game config.Game, id int) (found bool, mod *mods.Mod, err error)
	GetFromUrl(url string) (found bool, mod *mods.Mod, err error)
	GetNewestMods(game config.Game, lastID int) (result []*mods.Mod, err error)
	GetMods(game *config.Game) (result []*mods.Mod, err error)
	Folder(game config.Game) string
}

func NewCurseForgeClient() Client {
	return curseforge.NewClient(util.NewModCompiler(mods.CurseForge))
}

func NewNexusClient() Client {
	return nexus.NewClient(util.NewModCompiler(mods.Nexus))
}
