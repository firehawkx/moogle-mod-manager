package mod_author

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/kiamev/moogle-mod-manager/mods"
)

type modCompatabilityDef struct {
	requires *modCompatsDef
	forbids  *modCompatsDef
}

func newModCompatibilityDef() *modCompatabilityDef {
	return &modCompatabilityDef{
		requires: newModCompatsDef("Requires"),
		forbids:  newModCompatsDef("Forbids"),
	}
}

func (d *modCompatabilityDef) draw() fyne.CanvasObject {
	return container.NewVScroll(container.NewVBox(
		d.requires.draw(),
		d.forbids.draw(),
	))
}

func (d *modCompatabilityDef) compile() *mods.ModCompatibility {
	return &mods.ModCompatibility{
		Requires: d.requires.compile(),
		Forbids:  d.forbids.compile(),
	}
}

func (d *modCompatabilityDef) set(compatibility *mods.ModCompatibility) {
	d.requires.clear()
	d.forbids.clear()
	if compatibility != nil {
		for _, i := range compatibility.Requires {
			d.requires.list.AddItem(i)
		}
		for _, i := range compatibility.Forbids {
			d.forbids.list.AddItem(i)
		}
	}
}
