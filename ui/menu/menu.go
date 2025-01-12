package menu

import (
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/kiamev/moogle-mod-manager/browser"
	"github.com/kiamev/moogle-mod-manager/discover/repo"
	"github.com/kiamev/moogle-mod-manager/files"
	"github.com/kiamev/moogle-mod-manager/mods"
	"github.com/kiamev/moogle-mod-manager/mods/managed"
	"github.com/kiamev/moogle-mod-manager/ui/configure"
	"github.com/kiamev/moogle-mod-manager/ui/local"
	a "github.com/kiamev/moogle-mod-manager/ui/mod-author"
	"github.com/kiamev/moogle-mod-manager/ui/secret"
	"github.com/kiamev/moogle-mod-manager/ui/state"
	"github.com/kiamev/moogle-mod-manager/ui/state/gui"
	"github.com/kiamev/moogle-mod-manager/ui/state/ui"
	"github.com/kiamev/moogle-mod-manager/ui/util"
)

func New() state.Screen {
	m := &MainMenu{}
	gui.Current.AddListener(m)
	return m
}

type MainMenu struct {
	file    *fyne.Menu
	author  *fyne.Menu
	support *fyne.Menu
}

func (m *MainMenu) PreDraw(fyne.Window, ...interface{}) error { return nil }

func (m *MainMenu) OnClose() {}

func (m *MainMenu) DrawAsDialog(fyne.Window) {}

func (m *MainMenu) Draw(w fyne.Window) {
	if ui.Window != nil && m.author == nil {
		m.createAuthorMenu()
		m.createSupportMenu()
		m.DataChanged()
		w.SetMainMenu(fyne.NewMainMenu(m.file, m.author, m.support))
	}
}

func (m *MainMenu) DataChanged() {
	if m.author != nil {
		m.createFileMenu()
		var mm *fyne.MainMenu
		if i, err := gui.Current.Get(); err == nil && i != int(state.None) {
			mm = fyne.NewMainMenu(m.file, m.author, m.support)
		} else {
			mm = fyne.NewMainMenu(m.file, m.support)
		}
		ui.Window.SetMainMenu(mm)
	}
}

func (m *MainMenu) createFileMenu() {
	m.file = fyne.NewMenu("File")
	if state.GetCurrentGUI() == state.LocalMods {
		m.file.Items = append(m.file.Items,
			fyne.NewMenuItem("Select Games", func() {
				state.ShowPreviousScreen()
			}),
			fyne.NewMenuItemSeparator())
	}
	m.file.Items = append(m.file.Items,
		fyne.NewMenuItem("Configure", func() {
			configure.Show(ui.Window, nil)
		}),
		fyne.NewMenuItem("Secrets", func() {
			secret.Show(ui.Window)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Appearance", func() {
			s := settings.NewSettings()
			sd := s.LoadAppearanceScreen(ui.Window)
			d := dialog.NewCustom("Appearance", "Close", sd, ui.Window)
			d.Resize(fyne.NewSize(500, 500))
			d.Show()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Check For App Updates", func() {
			util.PromptForUpdateAsNeeded(false)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Clear Repo Cache (Debug)", func() {
			dialog.ShowConfirm("Clear Cache", "This will cause the application to close. Would you like to continue?", func(ok bool) {
				repo.ClearCache()
				ui.Window.Close()
			}, ui.Window)
		}))
	if state.GetCurrentGUI() == state.LocalMods {
		m.file.Items = append(m.file.Items,
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Force Disable All Mods (Debug)", func() {
				dialog.ShowConfirm(
					"Force Disable All Mods",
					"This will mark all mods as disabled but will not uninstall them. Are you sure you want to continue?",
					func(ok bool) {
						if ok {
							game := state.CurrentGame
							state.ShowScreen(state.None)

							managed.ForceDisableAll(game)
							files.RemoveAllFilesForGame(game)

							state.CurrentGame = game
							state.ShowScreen(state.LocalMods)
						}
					}, ui.Window)
			}),
			fyne.NewMenuItem("Force Disable Current Mod (Debug)", func() {
				tm := state.GetScreen(state.LocalMods).(local.LocalUI).GetSelected()
				if tm == nil || !tm.Enabled() {
					return
				}
				dialog.ShowConfirm(
					"Force Disable Current Mod",
					"This will mark the current mods as disabled but will not uninstall them. Are you sure you want to continue?",
					func(ok bool) {
						if ok {
							game := state.CurrentGame
							state.ShowScreen(state.None)

							managed.ForceDisable(tm)
							files.RemoveAllFilesForMod(game, tm.ID())

							state.CurrentGame = game
							state.ShowScreen(state.LocalMods)
						}
					}, ui.Window)
			}))
	}
}

func (m *MainMenu) createAuthorMenu() {
	if m.author == nil {
		m.author = fyne.NewMenu("Author")
		newMenu := fyne.NewMenuItem("New", nil)
		newMenu.ChildMenu = fyne.NewMenu("",
			fyne.NewMenuItem("Hosted Mod", func() {
				state.GetScreen(state.ModAuthor).(*a.ModAuthorer).NewHostedMod()
				state.ShowScreen(state.ModAuthor)
			}),
			fyne.NewMenuItem("From Nexus", func() {
				state.GetScreen(state.ModAuthor).(*a.ModAuthorer).NewNexusMod()
				state.ShowScreen(state.ModAuthor)
			}),
			fyne.NewMenuItem("From Curseforge", func() {
				state.GetScreen(state.ModAuthor).(*a.ModAuthorer).NewCurseForgeMod()
				state.ShowScreen(state.ModAuthor)
			}),
		)
		if state.GetCurrentGUI() != state.ModAuthor {
			m.author.Items = append(m.author.Items,
				newMenu,
				fyne.NewMenuItem("Edit Mod", func() {
					if state.GetScreen(state.ModAuthor).(*a.ModAuthorer).LoadModToEdit() {
						state.ShowScreen(state.ModAuthor)
					}
				}),
				fyne.NewMenuItem("Edit Current Mod", func() {
					if state.GetCurrentGUI() == state.LocalMods {
						if tm := state.GetScreen(state.LocalMods).(local.LocalUI).GetSelected(); tm != nil {
							state.GetScreen(state.ModAuthor).(*a.ModAuthorer).EditMod(tm.Mod(), func(mod *mods.Mod) {
								tm.SetMod(mod)
								if err := tm.Save(); err != nil {
									util.ShowErrorLong(err)
								}
							})
							state.ShowScreen(state.ModAuthor)
						}
					}
				}))
		} else {
			m.author.Items = append(m.author.Items,
				fyne.NewMenuItem("Close", func() {
					state.GetScreen(state.ModAuthor).OnClose()
					state.ShowPreviousScreen()
				}))
		}
	}
}

func (m *MainMenu) createSupportMenu() {
	m.support = fyne.NewMenu("Support Project")
	m.support.Items = append(m.support.Items, fyne.NewMenuItem("About", func() {
		purl, _ := url.Parse("https://www.patreon.com/kiamev")
		kurl, _ := url.Parse("https://ko-fi.com/kiamev")
		dialog.ShowCustom("About", "ok", container.NewBorder(
			widget.NewRichTextFromMarkdown(fmt.Sprintf(`
## Moogle Mod Manager %s
____________________________
Written by Kiame Vivacity

Contributors:

- Silvris`,
				browser.Version)), nil, nil, nil,
			container.NewVBox(
				widget.NewLabel("If you'd like to support the project:"),
				container.NewHBox(widget.NewLabel("- Patreon"), widget.NewHyperlink("https://www.patreon.com/kiamev", purl)),
				container.NewHBox(widget.NewLabel("- Ko-fi  "), widget.NewHyperlink("https://ko-fi.com/kiamev", kurl))),
		), ui.Window)
	}))
}
