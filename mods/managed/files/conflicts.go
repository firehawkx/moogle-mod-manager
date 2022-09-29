package files

import (
	"fmt"
	"github.com/kiamev/moogle-mod-manager/config"
	"github.com/kiamev/moogle-mod-manager/mods"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type (
	conflictResult struct {
		skip    map[string]bool
		replace map[string]bool
	}
	DoneCallback func(result mods.Result, skip conflictResult, err ...error)
)

func ResolveConflicts(enabler *mods.ModEnabler, managedFiles map[mods.ModID]*modFiles, modFiles []*mods.DownloadFiles, done DoneCallback) {
	c := config.Get()
	fileToMod := make(map[string]mods.ModID)
	for modID, mf := range managedFiles {
		for _, f := range mf.MovedFiles {
			fileToMod[c.RemoveGameDir(enabler.Game, f.To)] = modID
		}
	}
	toInstall, err := compileFilesToMove(enabler.Game, enabler.TrackedMod, modFiles)
	if err != nil {
		done(mods.Error, conflictResult{}, err)
		return
	}

	detectCollisions(enabler, toInstall, fileToMod, done)
	return
}

func compileFilesToMove(game config.Game, mod *mods.TrackedMod, modFiles []*mods.DownloadFiles) (toInstall []string, err error) {
	var (
		dir string
		f   string
	)
	for _, mf := range modFiles {
		for _, f := range mf.Files {
			to := f.To
			if filepath.Ext(to) == "" {
				to = filepath.Join(to, filepath.Base(f.From))
			}
			toInstall = append(toInstall, strings.ReplaceAll(to, "\\", "/"))
		}
		for _, d := range mf.Dirs {
			if dir, err = config.Get().GetDir(game, config.ModsDirKind); err != nil {
				return
			}
			dir = filepath.Join(dir, mod.GetDirSuffix(), mf.DownloadName, d.From)
			if d.Recursive {
				if err = filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if de == nil {
						return fmt.Errorf("[%s] does not exist", path)
					}
					if de.IsDir() {
						return nil
					}
					f = strings.ReplaceAll(path, "\\", "/")
					i := strings.Index(f, d.To)
					if i == -1 {
						return fmt.Errorf("failed to find `To` dir in `From` file. To: [%s] From: [%s]", d.To, f)
					}
					toInstall = append(toInstall, f[i:])
					return nil
				}); err != nil {
					return
				}
			} else {
				var de []fs.DirEntry
				if de, err = os.ReadDir(dir); err != nil {
					return
				}
				for _, e := range de {
					if e.IsDir() {
						continue
					}
					f = filepath.Join(d.To, e.Name())
					f = strings.ReplaceAll(f, "\\", "/")
					toInstall = append(toInstall, f)
				}
			}
		}
	}
	return
}

func detectCollisions(enabler *mods.ModEnabler, toInstall []string, installedFiles map[string]mods.ModID, done DoneCallback) {
	var (
		newModID   = enabler.TrackedMod.GetModID()
		collisions []*mods.FileConflict
		id         mods.ModID
		found      bool
		cr         = conflictResult{
			skip:    make(map[string]bool),
			replace: make(map[string]bool),
		}
	)
	for _, ti := range toInstall {
		if id, found = installedFiles[ti]; found {
			collisions = append(collisions, &mods.FileConflict{
				File:         ti,
				CurrentModID: id,
				NewModID:     newModID,
			})
		}
	}
	if len(collisions) > 0 {
		enabler.OnConflict(collisions, func(result mods.Result, choices []*mods.FileConflict, err ...error) {
			if result == mods.Error {
				done(result, cr, err...)
				return
			}
			if result == mods.Cancel {
				done(result, cr)
				return
			}
			for _, c := range choices {
				if c.ChoiceName != enabler.TrackedMod.DisplayName {
					cr.skip[c.File] = true
				} else {
					cr.replace[c.File] = true
				}
			}
			done(mods.Ok, cr)
		})
	} else {
		done(mods.Ok, cr)
	}
}