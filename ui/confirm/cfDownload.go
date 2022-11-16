package confirm

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/kiamev/moogle-mod-manager/mods"
	"github.com/kiamev/moogle-mod-manager/ui/state"
	"net/url"
	"os"
	"path/filepath"
)

type cfConfirmer struct{}

func (_ *cfConfirmer) ConfirmDownload(enabler *mods.ModEnabler, competeCallback DownloadCompleteCallback, done DownloadCallback) (err error) {
	var (
		u    *url.URL
		c    = container.NewVBox(widget.NewLabelWithStyle("Download the following file from Nexus", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		toDl []toDownload
	)
	for _, ti := range enabler.ToInstall {
		if ti.Download != nil && ti.Download.CurseForge != nil {
			dl := toDownload{
				uri: ti.Download.CurseForge.Url,
			}
			if dl.dir, err = ti.GetDownloadLocation(enabler.Game, enabler.TrackedMod); err != nil {
				return
			}
			if _, err = os.Stat(filepath.Join(dl.dir, ti.Download.Nexus.FileName)); err == nil {
				continue
			}
			toDl = append(toDl, dl)
		}
	}

	if len(toDl) == 0 {
		done(enabler, competeCallback, err)
		return
	}

	for _, td := range toDl {
		if u, err = url.Parse(td.uri); err != nil {
			return
		}
		c.Add(widget.NewHyperlink(td.uri, u))

		c.Add(widget.NewLabelWithStyle("Place download in:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

		if u, err = url.Parse(td.dir); err != nil {
			return
		}
		c.Add(widget.NewHyperlink(td.dir, u))
	}
	d := dialog.NewCustomConfirm("Download Files", "Done", "Cancel", container.NewVScroll(c), func(ok bool) {
		if ok {
			done(enabler, competeCallback, err)
		}
	}, state.Window)
	d.Resize(fyne.NewSize(500, 400))
	d.Show()
	return
}