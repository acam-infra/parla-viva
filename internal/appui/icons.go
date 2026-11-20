package appui

import "github.com/vivaui/viva/gfx"

func lucideIcon(cmds ...gfx.PathCommand) gfx.Icon {
	return gfx.Icon{
		ViewBox:  24,
		Stroke:   2,
		Cap:      gfx.CapRound,
		Join:     gfx.JoinRound,
		Commands: cmds,
	}
}

// Crisp 24x24 Lucide-style GPU vector icons (no runtime SVG parsing).
var (
	// IconPaperclip attaches any file.
	IconPaperclip = lucideIcon(
		gfx.MoveTo(21.44, 11.05),
		gfx.LineTo(12.25, 20.24),
		gfx.CubicTo(9.9, 22.59, 6.1, 22.59, 3.76, 20.24),
		gfx.CubicTo(1.41, 17.9, 1.41, 14.1, 3.76, 11.75),
		gfx.LineTo(12.95, 2.56),
		gfx.CubicTo(14.51, 1.0, 17.04, 1.0, 18.6, 2.56),
		gfx.CubicTo(20.16, 4.12, 20.16, 6.65, 18.6, 8.21),
		gfx.LineTo(9.41, 17.41),
		gfx.CubicTo(8.63, 18.19, 7.36, 18.19, 6.58, 17.41),
		gfx.CubicTo(5.8, 16.63, 5.8, 15.36, 6.58, 14.58),
		gfx.LineTo(15.07, 6.1),
	)

	// IconImage attaches photos/images.
	IconImage = lucideIcon(
		// Rounded frame
		gfx.MoveTo(5, 3), gfx.LineTo(19, 3),
		gfx.QuadTo(21, 3, 21, 5), gfx.LineTo(21, 19),
		gfx.QuadTo(21, 21, 19, 21), gfx.LineTo(5, 21),
		gfx.QuadTo(3, 21, 3, 19), gfx.LineTo(3, 5),
		gfx.QuadTo(3, 3, 5, 3), gfx.ClosePath(),
		// Sun dot
		gfx.MoveTo(8.5, 8.5), gfx.LineTo(8.51, 8.5),
		// Mountain silhouette
		gfx.MoveTo(21, 15), gfx.LineTo(16, 10), gfx.LineTo(5, 21),
	)

	// IconSend is a paper-plane send glyph.
	IconSend = lucideIcon(
		gfx.MoveTo(22, 2), gfx.LineTo(11, 13),
		gfx.MoveTo(22, 2), gfx.LineTo(15, 22),
		gfx.LineTo(11, 13), gfx.LineTo(2, 9), gfx.ClosePath(),
	)

	// IconSearch is a magnifying glass.
	IconSearch = lucideIcon(
		gfx.MoveTo(11, 4),
		gfx.CubicTo(14.87, 4, 18, 7.13, 18, 11),
		gfx.CubicTo(18, 14.87, 14.87, 18, 11, 18),
		gfx.CubicTo(7.13, 18, 4, 14.87, 4, 11),
		gfx.CubicTo(4, 7.13, 7.13, 4, 11, 4),
		gfx.MoveTo(21, 21), gfx.LineTo(16.65, 16.65),
	)

	// IconGallery is a 2x2 media grid.
	IconGallery = lucideIcon(
		gfx.MoveTo(3, 3), gfx.LineTo(10, 3), gfx.LineTo(10, 10), gfx.LineTo(3, 10), gfx.ClosePath(),
		gfx.MoveTo(14, 3), gfx.LineTo(21, 3), gfx.LineTo(21, 10), gfx.LineTo(14, 10), gfx.ClosePath(),
		gfx.MoveTo(14, 14), gfx.LineTo(21, 14), gfx.LineTo(21, 21), gfx.LineTo(14, 21), gfx.ClosePath(),
		gfx.MoveTo(3, 14), gfx.LineTo(10, 14), gfx.LineTo(10, 21), gfx.LineTo(3, 21), gfx.ClosePath(),
	)

	// IconMore is three horizontal dots.
	IconMore = lucideIcon(
		gfx.MoveTo(5, 12), gfx.LineTo(5.01, 12),
		gfx.MoveTo(12, 12), gfx.LineTo(12.01, 12),
		gfx.MoveTo(19, 12), gfx.LineTo(19.01, 12),
	)

	// IconSettings is a sliders/preferences icon.
	IconSettings = lucideIcon(
		gfx.MoveTo(4, 6), gfx.LineTo(20, 6),
		gfx.MoveTo(4, 12), gfx.LineTo(20, 12),
		gfx.MoveTo(4, 18), gfx.LineTo(20, 18),
		gfx.MoveTo(9, 4), gfx.LineTo(9, 8),
		gfx.MoveTo(15, 10), gfx.LineTo(15, 14),
		gfx.MoveTo(11, 16), gfx.LineTo(11, 20),
	)

	// IconPlus is a plus sign.
	IconPlus = lucideIcon(
		gfx.MoveTo(12, 5), gfx.LineTo(12, 19),
		gfx.MoveTo(5, 12), gfx.LineTo(19, 12),
	)

	// IconClose is an X glyph.
	IconClose = lucideIcon(
		gfx.MoveTo(18, 6), gfx.LineTo(6, 18),
		gfx.MoveTo(6, 6), gfx.LineTo(18, 18),
	)

	// IconFile is a document icon for file attachments.
	IconFile = lucideIcon(
		gfx.MoveTo(14, 2), gfx.LineTo(6, 2),
		gfx.QuadTo(4, 2, 4, 4), gfx.LineTo(4, 20),
		gfx.QuadTo(4, 22, 6, 22), gfx.LineTo(18, 22),
		gfx.QuadTo(20, 22, 20, 20), gfx.LineTo(20, 8), gfx.ClosePath(),
		gfx.MoveTo(14, 2), gfx.LineTo(14, 8), gfx.LineTo(20, 8),
	)

	// IconCode is a <> code/json glyph.
	IconCode = lucideIcon(
		gfx.MoveTo(16, 18), gfx.LineTo(22, 12), gfx.LineTo(16, 6),
		gfx.MoveTo(8, 6), gfx.LineTo(2, 12), gfx.LineTo(8, 18),
	)

	// IconTable is a grid/spreadsheet glyph for CSV/TSV files.
	IconTable = lucideIcon(
		gfx.MoveTo(4, 4), gfx.LineTo(20, 4), gfx.LineTo(20, 20), gfx.LineTo(4, 20), gfx.ClosePath(),
		gfx.MoveTo(4, 10), gfx.LineTo(20, 10),
		gfx.MoveTo(4, 15), gfx.LineTo(20, 15),
		gfx.MoveTo(10, 4), gfx.LineTo(10, 20),
	)

	// IconArchive is a package/zip glyph.
	IconArchive = lucideIcon(
		gfx.MoveTo(3, 5), gfx.LineTo(21, 5), gfx.LineTo(21, 9), gfx.LineTo(3, 9), gfx.ClosePath(),
		gfx.MoveTo(5, 9), gfx.LineTo(5, 19), gfx.LineTo(19, 19), gfx.LineTo(19, 9),
		gfx.MoveTo(10, 13), gfx.LineTo(14, 13),
	)

	// IconApp is a cube glyph for Webxdc mini-apps.
	IconApp = lucideIcon(
		gfx.MoveTo(12, 3), gfx.LineTo(20, 7.5), gfx.LineTo(20, 16.5),
		gfx.LineTo(12, 21), gfx.LineTo(4, 16.5), gfx.LineTo(4, 7.5), gfx.ClosePath(),
		gfx.MoveTo(12, 12), gfx.LineTo(20, 7.5),
		gfx.MoveTo(12, 12), gfx.LineTo(12, 21),
		gfx.MoveTo(12, 12), gfx.LineTo(4, 7.5),
	)
)
