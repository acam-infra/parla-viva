package appui

import "github.com/vivaui/viva/gfx"

func lucideIcon(cmds ...gfx.PathCommand) gfx.Icon {
	return gfx.Icon{
		ViewBox:  24,
		Stroke:   1.9,
		Cap:      gfx.CapRound,
		Join:     gfx.JoinRound,
		Commands: cmds,
	}
}

// Crisp 24x24 Lucide-style GPU vector icons (no runtime SVG parsing).
var (
	// IconMessage is a rounded chat bubble glyph for the Chats navigation tab.
	IconMessage = lucideIcon(
		gfx.MoveTo(21, 11.5),
		gfx.CubicTo(21, 16.19, 16.97, 20, 12, 20),
		gfx.CubicTo(10.65, 20, 9.37, 19.72, 8.21, 19.22),
		gfx.LineTo(3, 21),
		gfx.LineTo(4.65, 16.58),
		gfx.CubicTo(3.61, 15.17, 3, 13.41, 3, 11.5),
		gfx.CubicTo(3, 6.81, 7.03, 3, 12, 3),
		gfx.CubicTo(16.97, 3, 21, 6.81, 21, 11.5),
		gfx.ClosePath(),
	)

	// IconUsers is a contacts/group glyph for the Contacts navigation tab.
	IconUsers = lucideIcon(
		gfx.MoveTo(16, 21), gfx.LineTo(16, 19),
		gfx.CubicTo(16, 16.79, 14.21, 15, 12, 15),
		gfx.LineTo(6, 15),
		gfx.CubicTo(3.79, 15, 2, 16.79, 2, 19),
		gfx.LineTo(2, 21),
		gfx.MoveTo(9, 11),
		gfx.CubicTo(11.21, 11, 13, 9.21, 13, 7),
		gfx.CubicTo(13, 4.79, 11.21, 3, 9, 3),
		gfx.CubicTo(6.79, 3, 5, 4.79, 5, 7),
		gfx.CubicTo(5, 9.21, 6.79, 11, 9, 11),
		gfx.MoveTo(22, 21), gfx.LineTo(22, 19),
		gfx.CubicTo(22, 17.13, 20.73, 15.56, 19, 15.13),
		gfx.MoveTo(16, 3.13),
		gfx.CubicTo(17.73, 3.56, 19, 5.13, 19, 7),
		gfx.CubicTo(19, 8.87, 17.73, 10.44, 16, 10.87),
	)

	// IconFolder is a folder glyph for Chat Files and Files & Media views.
	IconFolder = lucideIcon(
		gfx.MoveTo(4, 20), gfx.LineTo(20, 20),
		gfx.QuadTo(22, 20, 22, 18), gfx.LineTo(22, 8),
		gfx.QuadTo(22, 6, 20, 6), gfx.LineTo(12.5, 6),
		gfx.LineTo(10.5, 4), gfx.LineTo(4, 4),
		gfx.QuadTo(2, 4, 2, 6), gfx.LineTo(2, 18),
		gfx.QuadTo(2, 20, 4, 20), gfx.ClosePath(),
	)

	// IconUserPlus is an invite / add contact glyph.
	IconUserPlus = lucideIcon(
		gfx.MoveTo(16, 21), gfx.LineTo(16, 19),
		gfx.CubicTo(16, 16.79, 14.21, 15, 12, 15),
		gfx.LineTo(6, 15),
		gfx.CubicTo(3.79, 15, 2, 16.79, 2, 19),
		gfx.LineTo(2, 21),
		gfx.MoveTo(9, 11),
		gfx.CubicTo(11.21, 11, 13, 9.21, 13, 7),
		gfx.CubicTo(13, 4.79, 11.21, 3, 9, 3),
		gfx.CubicTo(6.79, 3, 5, 4.79, 5, 7),
		gfx.CubicTo(5, 9.21, 6.79, 11, 9, 11),
		gfx.MoveTo(19, 8), gfx.LineTo(19, 14),
		gfx.MoveTo(16, 11), gfx.LineTo(22, 11),
	)

	// IconSun is a daylight theme toggle glyph.
	IconSun = lucideIcon(
		gfx.MoveTo(12, 8),
		gfx.CubicTo(14.21, 8, 16, 9.79, 16, 12),
		gfx.CubicTo(16, 14.21, 14.21, 16, 12, 16),
		gfx.CubicTo(9.79, 16, 8, 14.21, 8, 12),
		gfx.CubicTo(8, 9.79, 9.79, 8, 12, 8),
		gfx.MoveTo(12, 2), gfx.LineTo(12, 4),
		gfx.MoveTo(12, 20), gfx.LineTo(12, 22),
		gfx.MoveTo(4.93, 4.93), gfx.LineTo(6.34, 6.34),
		gfx.MoveTo(17.66, 17.66), gfx.LineTo(19.07, 19.07),
		gfx.MoveTo(2, 12), gfx.LineTo(4, 12),
		gfx.MoveTo(20, 12), gfx.LineTo(22, 12),
		gfx.MoveTo(6.34, 17.66), gfx.LineTo(4.93, 19.07),
		gfx.MoveTo(19.07, 4.93), gfx.LineTo(17.66, 6.34),
	)

	// IconMoon is a dark theme toggle glyph.
	IconMoon = lucideIcon(
		gfx.MoveTo(21, 12.79),
		gfx.CubicTo(20.84, 14.49, 20.2, 16.11, 19.14, 17.45),
		gfx.CubicTo(17.34, 19.72, 14.54, 21, 11.62, 21),
		gfx.CubicTo(6.31, 21, 2, 16.69, 2, 11.38),
		gfx.CubicTo(2, 8.46, 3.28, 5.66, 5.55, 3.86),
		gfx.CubicTo(6.89, 2.8, 8.51, 2.16, 10.21, 2),
		gfx.CubicTo(9.2, 3.37, 8.78, 5.09, 9.04, 6.78),
		gfx.CubicTo(9.45, 9.52, 11.48, 11.55, 14.22, 11.96),
		gfx.CubicTo(15.91, 12.22, 17.63, 11.8, 19, 10.79),
		gfx.LineTo(21, 12.79),
		gfx.ClosePath(),
	)

	// IconRefresh is a circular sync/refresh glyph.
	IconRefresh = lucideIcon(
		gfx.MoveTo(21, 12),
		gfx.CubicTo(21, 16.97, 16.97, 21, 12, 21),
		gfx.CubicTo(8.56, 21, 5.56, 19.07, 4.05, 16.23),
		gfx.MoveTo(3, 12),
		gfx.CubicTo(3, 7.03, 7.03, 3, 12, 3),
		gfx.CubicTo(15.44, 3, 18.44, 4.93, 19.95, 7.77),
		gfx.MoveTo(20, 3), gfx.LineTo(20, 8), gfx.LineTo(15, 8),
		gfx.MoveTo(4, 21), gfx.LineTo(4, 16), gfx.LineTo(9, 16),
	)

	// IconChevronDown is a downward disclosure chevron.
	IconChevronDown = lucideIcon(
		gfx.MoveTo(6, 9), gfx.LineTo(12, 15), gfx.LineTo(18, 9),
	)

	// IconChevronUp is an upward disclosure chevron.
	IconChevronUp = lucideIcon(
		gfx.MoveTo(18, 15), gfx.LineTo(12, 9), gfx.LineTo(6, 15),
	)

	// IconChevronRight is a rightward disclosure/navigation chevron.
	IconChevronRight = lucideIcon(
		gfx.MoveTo(9, 6), gfx.LineTo(15, 12), gfx.LineTo(9, 18),
	)

	// IconSmile is an emoji picker glyph for the composer toolbar.
	IconSmile = lucideIcon(
		gfx.MoveTo(12, 2),
		gfx.CubicTo(17.52, 2, 22, 6.48, 22, 12),
		gfx.CubicTo(22, 17.52, 17.52, 22, 12, 22),
		gfx.CubicTo(6.48, 22, 2, 17.52, 2, 12),
		gfx.CubicTo(2, 6.48, 6.48, 2, 12, 2),
		gfx.MoveTo(8, 14), gfx.QuadTo(12, 18, 16, 14),
		gfx.MoveTo(9, 9), gfx.LineTo(9.01, 9),
		gfx.MoveTo(15, 9), gfx.LineTo(15.01, 9),
	)

	// IconShield is an end-to-end encryption / verified shield glyph.
	IconShield = lucideIcon(
		gfx.MoveTo(12, 22),
		gfx.CubicTo(12, 22, 20, 18, 20, 12),
		gfx.LineTo(20, 5), gfx.LineTo(12, 2),
		gfx.LineTo(4, 5), gfx.LineTo(4, 12),
		gfx.CubicTo(4, 18, 12, 22, 12, 22),
		gfx.ClosePath(),
		gfx.MoveTo(9, 12), gfx.LineTo(11, 14), gfx.LineTo(15, 10),
	)

	// IconInfo is an info circle glyph.
	IconInfo = lucideIcon(
		gfx.MoveTo(12, 2),
		gfx.CubicTo(17.52, 2, 22, 6.48, 22, 12),
		gfx.CubicTo(22, 17.52, 17.52, 22, 12, 22),
		gfx.CubicTo(6.48, 22, 2, 17.52, 2, 12),
		gfx.CubicTo(2, 6.48, 6.48, 2, 12, 2),
		gfx.MoveTo(12, 16), gfx.LineTo(12, 12),
		gfx.MoveTo(12, 8), gfx.LineTo(12.01, 8),
	)

	// IconLock is a security / backup lock glyph.
	IconLock = lucideIcon(
		gfx.MoveTo(5, 11), gfx.LineTo(19, 11),
		gfx.QuadTo(21, 11, 21, 13), gfx.LineTo(21, 20),
		gfx.QuadTo(21, 22, 19, 22), gfx.LineTo(5, 22),
		gfx.QuadTo(3, 22, 3, 20), gfx.LineTo(3, 13),
		gfx.QuadTo(3, 11, 5, 11), gfx.ClosePath(),
		gfx.MoveTo(7, 11), gfx.LineTo(7, 7),
		gfx.CubicTo(7, 4.24, 9.24, 2, 12, 2),
		gfx.CubicTo(14.76, 2, 17, 4.24, 17, 7),
		gfx.LineTo(17, 11),
	)

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
		gfx.MoveTo(5, 3), gfx.LineTo(19, 3),
		gfx.QuadTo(21, 3, 21, 5), gfx.LineTo(21, 19),
		gfx.QuadTo(21, 21, 19, 21), gfx.LineTo(5, 21),
		gfx.QuadTo(3, 21, 3, 19), gfx.LineTo(3, 5),
		gfx.QuadTo(3, 3, 5, 3), gfx.ClosePath(),
		gfx.MoveTo(8.5, 8.5), gfx.LineTo(8.51, 8.5),
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

	// IconBell is a notification bell glyph for unmuted chats.
	IconBell = lucideIcon(
		gfx.MoveTo(6, 8),
		gfx.CubicTo(6, 4.69, 8.69, 2, 12, 2),
		gfx.CubicTo(15.31, 2, 18, 4.69, 18, 8),
		gfx.CubicTo(18, 13, 20, 15, 21, 16),
		gfx.LineTo(3, 16),
		gfx.CubicTo(4, 15, 6, 13, 6, 8),
		gfx.MoveTo(10.3, 21),
		gfx.QuadTo(12, 22.5, 13.7, 21),
	)

	// IconBellOff is a crossed-out notification bell glyph for muted chats.
	IconBellOff = lucideIcon(
		gfx.MoveTo(8.7, 3),
		gfx.CubicTo(9.7, 2.35, 10.82, 2, 12, 2),
		gfx.CubicTo(15.31, 2, 18, 4.69, 18, 8),
		gfx.CubicTo(18, 10.2, 18.4, 11.9, 19.1, 13.2),
		gfx.MoveTo(6.26, 6.26),
		gfx.CubicTo(6.09, 6.81, 6, 7.4, 6, 8),
		gfx.CubicTo(6, 13, 4, 15, 3, 16),
		gfx.LineTo(16, 16),
		gfx.MoveTo(10.3, 21),
		gfx.QuadTo(12, 22.5, 13.7, 21),
		gfx.MoveTo(2, 2), gfx.LineTo(22, 22),
	)

	// IconPin is a compact pushpin glyph for pinned chats & window pin.
	IconPin = lucideIcon(
		gfx.MoveTo(12, 17), gfx.LineTo(12, 22),
		gfx.MoveTo(5, 17), gfx.LineTo(19, 17),
		gfx.LineTo(17, 11), gfx.LineTo(17, 5),
		gfx.LineTo(18, 3), gfx.LineTo(6, 3),
		gfx.LineTo(7, 5), gfx.LineTo(7, 11), gfx.ClosePath(),
	)
)
