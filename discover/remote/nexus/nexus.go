package nexus

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kiamev/moogle-mod-manager/config"
	"github.com/kiamev/moogle-mod-manager/config/secrets"
	u "github.com/kiamev/moogle-mod-manager/discover/remote/util"
	"github.com/kiamev/moogle-mod-manager/mods"
)

const (
	nexusApiModUrl         = "https://api.nexusmods.com/v1/games/%s/mods/%s.json"
	nexusApiModDlUrl       = "https://api.nexusmods.com/v1/games/%s/mods/%s/files.json%s"
	nexusApiModDlUrlSuffix = "?category=main,update,optional,miscellaneous"
	nexusUrl               = "https://www.nexusmods.com/%s/mods/%d"
	nexusApiNewestModsUrl  = "https://api.nexusmods.com/v1/games/%s/mods/latest_added.json"

	// nexusUsersApiUrl = "https://users.nexusmods.com/oauth/token"

	// NexusFileDownload file_id, NexusGameID
	NexusFileDownload = "https://www.nexusmods.com/Core/Libs/Common/Widgets/DownloadPopUp?id=%d&game_id=%v"
)

var (
	bbcodeRegex *regexp.Regexp = regexp.MustCompile(`\[\/?[a-zA-Z0-9=\[\]_ ]+\]`)
	cache                      = make(map[config.GameID][]*mods.Mod)
)

type client struct {
	compiler u.ModCompiler
}

func NewClient(compiler u.ModCompiler) *client {
	c := &client{compiler: compiler}
	compiler.SetFinder(c)
	return c
}

func IsNexus(url string) bool {
	return strings.Contains(url, "nexusmods.com")
}

func (c *client) GetFromMod(in *mods.Mod) (found bool, mod *mods.Mod, err error) {
	if secrets.Get(secrets.NexusApiKey) == "" {
		return false, nil, errors.New("no nexus api key set in File->Secrets")
	}
	if len(in.Games) == 0 {
		err = fmt.Errorf("no games found for mod %s", in.Name)
		return
	}
	if !in.ModKind.Kinds.Is(mods.Nexus) || in.ModKind.NexusID == nil {
		err = fmt.Errorf("mod %s is not a nexus mod", in.Name)
		return
	}
	var game config.GameDef
	if game, err = config.GameDefFromID(in.Games[0].ID); err != nil {
		return
	}
	return c.GetFromUrl(fmt.Sprintf(nexusUrl, game.Remote().Nexus.Path, *in.ModKind.NexusID))
}

func (c *client) GetFromID(game config.GameDef, id int) (found bool, mod *mods.Mod, err error) {
	if secrets.Get(secrets.NexusApiKey) == "" {
		return false, nil, nil
	}
	return c.GetFromUrl(fmt.Sprintf(nexusUrl, game.Remote().Nexus.Path, id))
}

func (c *client) GetFromUrl(url string) (found bool, mod *mods.Mod, err error) {
	if secrets.Get(secrets.NexusApiKey) == "" {
		return false, nil, nil
	}
	var (
		sp    = strings.Split(url, "/")
		path  string
		modID string
		b     []byte
		nMod  nexusMod
		nDls  fileParent
	)
	for i, s := range sp {
		if s == "mods" {
			if i > 0 && i < len(sp)-1 {
				path = sp[i-1]
				modID = strings.Split(sp[i+1], "?")[0]
				break
			}
		}
	}
	if path == "" || modID == "" {
		err = fmt.Errorf("could not get GameDef and Mod ModID from %s", url)
		return
	}
	if b, err = sendRequest(fmt.Sprintf(nexusApiModUrl, path, modID)); err != nil {
		return
	}
	if err = json.Unmarshal(b, &nMod); err != nil {
		return
	}
	if nMod.Name == "" {
		err = errors.New("no mod found for " + modID)
		return
	}
	if nDls, err = getDownloads(config.NexusPath(path), modID); err != nil {
		return
	}
	return toMod(nMod, nDls.Files)
}

func (c *client) GetNewestMods(game config.GameDef, lastID int) (result []*mods.Mod, err error) {
	if secrets.Get(secrets.NexusApiKey) == "" {
		return nil, errors.New("no nexus api key set in File->Secrets")
	}
	var (
		b       []byte
		path    = game.Remote().Nexus.Path
		nDls    fileParent
		mod     *mods.Mod
		include bool
	)
	if b, err = sendRequest(fmt.Sprintf(nexusApiNewestModsUrl, path)); err != nil {
		return
	}
	var nMods []nexusMod
	if err = json.Unmarshal(b, &nMods); err != nil {
		return
	}

	result = make([]*mods.Mod, 0, len(nMods))
	for _, nMod := range nMods {
		if nMod.ModID > lastID {
			if nDls, err = getDownloads(path, fmt.Sprintf("%d", nMod.ModID)); err != nil {
				return
			}
			if include, mod, err = toMod(nMod, nDls.Files); err != nil {
				return
			} else if !include {
				continue
			}
			result = append(result, mod)
		}
	}
	return
}

func GetDownloads(game config.GameDef, modID string) (dls []*mods.Download, err error) {
	if secrets.Get(secrets.NexusApiKey) == "" {
		return nil, errors.New("no nexus api key set in File->Secrets")
	}
	var (
		b    []byte
		path = game.Remote().Nexus.Path
		url  = fmt.Sprintf(nexusApiModDlUrl, path, modID, nexusApiModDlUrlSuffix)
		nDls fileParent
	)
	if b, err = sendRequest(url); err != nil {
		return
	}
	if err = json.Unmarshal(b, &nDls); err != nil {
		return
	}
	dls = nDls.ToDownloads()
	return
}

func getDownloads(path config.NexusPath, modID string) (nDls fileParent, err error) {
	var (
		b   []byte
		url = fmt.Sprintf(nexusApiModDlUrl, path, modID, nexusApiModDlUrlSuffix)
	)
	if b, err = sendRequest(url); err != nil {
		return
	}
	err = json.Unmarshal(b, &nDls)
	return
}

func sendRequest(url string) (response []byte, err error) {
	var (
		apiKey = secrets.Get(secrets.NexusApiKey)
		req    *http.Request
		resp   *http.Response
	)
	if apiKey == "" {
		err = errors.New("no Nexus Api Key set. Please go to File->Secrets")
		return
	}
	if req, err = http.NewRequest(http.MethodGet, url, nil); err != nil {
		err = fmt.Errorf("failed to create request to validate user with nexus %s: %v", url, err)
		return
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("apikey", apiKey)

	if resp, err = (&http.Client{}).Do(req); err != nil {
		err = fmt.Errorf("failed to make request to %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	code := resp.StatusCode
	if code < 200 && code >= 300 {
		err = fmt.Errorf("received code [%d] from call to [%s]", code, url)
		return
	}

	if response, err = io.ReadAll(resp.Body); err != nil {
		err = fmt.Errorf("failed to read response's body for %s: %v", url, err)
	}
	return
}

func toMod(n nexusMod, dls []NexusFile) (include bool, mod *mods.Mod, err error) {
	var (
		modID = fmt.Sprintf("%d", n.ModID)
		game  config.GameDef
	)
	if game, err = config.GameDefFromNexusPath(n.GamePath); err != nil {
		return
	}
	mod = mods.NewMod(&mods.ModDef{
		ModID:        mods.NewModID(mods.Nexus, modID),
		Name:         mods.ModName(n.Name),
		Version:      n.Version,
		Author:       n.Author,
		AuthorLink:   n.AuthorLink,
		Description:  bbcodeRegex.ReplaceAllString(n.Summary, ""),
		Category:     "",
		ReleaseDate:  n.CreatedTime.Format("Jan 2, 2006"),
		ReleaseNotes: "",
		Link:         fmt.Sprintf(nexusUrl, n.GamePath, n.ModID),
		Previews: []*mods.Preview{{
			Url:   &n.PictureUrl,
			Local: nil,
		}},
		ModKind: mods.ModKind{
			Kinds:   mods.Kinds{mods.Nexus},
			NexusID: (*mods.NexusModID)(&n.ModID),
		},
		Games: []*mods.Game{{
			ID:       game.ID(),
			Versions: nil,
		}},
		Downloadables:  make([]*mods.Download, len(dls)),
		DonationLinks:  nil,
		AlwaysDownload: nil,
		Configurations: nil,
	})

	var choices []*mods.Choice
	for i, d := range dls {
		mod.Downloadables[i] = d.ToDownload()
		dlf := &mods.DownloadFiles{
			DownloadName: d.Name,
			Dirs: []*mods.ModDir{
				{
					From:      string(game.BaseDir()),
					To:        string(game.BaseDir()),
					Recursive: true,
				},
			},
		}
		choices = append(choices, &mods.Choice{
			Name:                  d.Name,
			Description:           d.Description,
			Preview:               nil,
			DownloadFiles:         dlf,
			NextConfigurationName: nil,
		})
	}

	/*include = true
	if len(choices) > 1 {
		mod.Configurations = []*mods.Configuration{
			{
				Name:        "Choose",
				Description: "",
				Preview:     nil,
				Root:        true,
				Choices:     choices,
			},
		}
	} else if len(choices) == 1 {
		mod.AlwaysDownload = append(mod.AlwaysDownload, choices[0].DownloadFiles)
	} else {
		include = false
	}*/
	return
}

func (c *client) Folder(game config.GameDef) string {
	return filepath.Join(config.PWD, "remote", string(game.ID()), string(mods.Nexus))
}

func (c *client) GetMods(game config.GameDef, rebuildCache bool) (result []*mods.Mod, err error) {
	if secrets.Get(secrets.NexusApiKey) == "" {
		return nil, nil
	}
	if !rebuildCache {
		if l, f := cache[game.ID()]; f {
			result = l
			return
		}
	}
	if game == nil {
		return nil, errors.New("GetMods called with a nil game")
	}
	dir := c.Folder(game)
	_ = os.MkdirAll(dir, 0777)
	if err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "mod.json" || d.Name() == "mod.xml" {
			m := &mods.Mod{}
			if err = m.LoadFromFile(path); err != nil {
				return err
			}
			result = append(result, m)
		}
		return nil
	}); err != nil {
		return
	}
	if result, err = c.compiler.AppendNewMods(c.Folder(game), game, result); err != nil {
		return
	}
	cache[game.ID()] = result
	return
}
