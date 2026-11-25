package appui

import (
	"path/filepath"
	"strings"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/preview"
)

// Shell layout constants derived from GoIM's Viva desktop geometry.
const (
	TitlebarHeight            = float32(42)
	RailWidth                 = float32(68)
	SidebarDefaultWidth       = float32(288)
	SidebarMinWidth           = float32(240)
	SidebarMaxWidth           = float32(400)
	ChatHeaderHeight          = float32(60)
	ChatFilesPaneWidth        = float32(328)
	WorkspaceInset            = float32(6)
	RadiusControl             = float32(10)
	RadiusCard                = float32(12)
	ComposerToolbarHeight     = float32(38)
	ComposerFooterHeight      = float32(40)
	ComposerEditorHeight      = float32(76)
	SidebarConversationHeight = float32(64)
	SidebarContactHeight      = float32(54)
)

// ColorScheme identifiers matching GoIM's four curated desktop palettes.
const (
	SchemeClassic = "classic"
	SchemePine    = "pine"
	SchemeClay    = "clay"
	SchemeIris    = "iris"
)

var ColorSchemes = []string{SchemeClassic, SchemePine, SchemeClay, SchemeIris}

// ColorSchemeLabel returns the human-readable name for a color scheme key.
func ColorSchemeLabel(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case SchemePine:
		return "Pine"
	case SchemeClay:
		return "Clay"
	case SchemeIris:
		return "Iris"
	default:
		return "Classic"
	}
}

// NormalizeColorScheme ensures a scheme name maps to one of the supported schemes.
func NormalizeColorScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case SchemePine:
		return SchemePine
	case SchemeClay:
		return SchemeClay
	case SchemeIris:
		return SchemeIris
	default:
		return SchemeClassic
	}
}

// Palette holds the layered surface and semantic colors for the GoIM-inspired desktop shell.
// Separating rail, sidebar, chat canvas, and bubble surfaces creates depth without heavy chrome.
type Palette struct {
	Rail          gfx.Color
	Sidebar       gfx.Color
	Chat          gfx.Color
	Header        gfx.Color
	Border        gfx.Color
	Hover         gfx.Color
	Selected      gfx.Color
	SelectedText  gfx.Color
	Icon          gfx.Color
	SecondaryText gfx.Color
	MutedAvatar   gfx.Color
	AvatarText    gfx.Color
	Incoming      gfx.Color
	Own           gfx.Color
	BubbleBorder  gfx.Color
	OwnBorder     gfx.Color
	FileCard      gfx.Color
	FileBorder    gfx.Color
	BadgeMutedBg  gfx.Color
	BadgeMutedFg  gfx.Color
	Success       gfx.Color
	Danger        gfx.Color
	Warning       gfx.Color
}

type schemeTones struct {
	background, surface, ink, mutedInk, primary, selection gfx.Color
}

var schemePresets = map[string]struct{ light, dark schemeTones }{
	SchemePine: {
		light: schemeTones{ui.RGB(238, 246, 241), ui.RGB(251, 254, 251), ui.RGB(26, 52, 44), ui.RGB(75, 101, 90), ui.RGB(18, 104, 79), ui.RGB(199, 232, 214)},
		dark:  schemeTones{ui.RGB(22, 38, 34), ui.RGB(33, 52, 45), ui.RGB(225, 240, 230), ui.RGB(165, 191, 175), ui.RGB(120, 212, 169), ui.RGB(43, 79, 62)},
	},
	SchemeClay: {
		light: schemeTones{ui.RGB(250, 242, 232), ui.RGB(255, 251, 245), ui.RGB(64, 44, 38), ui.RGB(116, 89, 74), ui.RGB(148, 65, 42), ui.RGB(246, 218, 196)},
		dark:  schemeTones{ui.RGB(43, 32, 29), ui.RGB(58, 44, 38), ui.RGB(248, 231, 214), ui.RGB(202, 176, 156), ui.RGB(241, 164, 126), ui.RGB(92, 59, 46)},
	},
	SchemeIris: {
		light: schemeTones{ui.RGB(244, 241, 250), ui.RGB(253, 251, 255), ui.RGB(48, 40, 68), ui.RGB(100, 89, 118), ui.RGB(105, 74, 153), ui.RGB(228, 216, 248)},
		dark:  schemeTones{ui.RGB(32, 29, 43), ui.RGB(46, 41, 60), ui.RGB(238, 232, 249), ui.RGB(186, 174, 205), ui.RGB(202, 176, 246), ui.RGB(72, 57, 94)},
	},
}

func (s schemeTones) apply(t *gfx.Theme) {
	t.Background, t.Card, t.Popover = s.background, s.surface, s.surface
	t.Foreground, t.CardForeground, t.PopoverForeground = s.ink, s.ink, s.ink
	t.MutedForeground = s.mutedInk
	t.Primary, t.Ring = s.primary, s.primary
	t.PrimaryForeground = s.surface
	if s.background.Red < 0.5 {
		t.PrimaryForeground = s.background
	}
	t.Selection, t.SelectionForeground = s.selection, s.ink
	t.CardElevated = s.surface.Mix(s.primary, 0.06)
	t.Hover = s.surface.Mix(s.primary, 0.11)
	t.Secondary, t.Muted, t.Accent = t.CardElevated, t.CardElevated, t.Hover
	t.SecondaryForeground, t.AccentForeground = s.ink, s.ink
	t.Border = s.surface.Mix(s.ink, 0.16)
	t.Input = t.Border
}

// BuildAppearance constructs the Viva gfx.Theme and GoIM-inspired layered Palette
// for a dark/light mode and color scheme.
func BuildAppearance(dark bool, scheme string) (gfx.Theme, Palette) {
	scheme = NormalizeColorScheme(scheme)
	var theme gfx.Theme
	if dark {
		theme = gfx.DarkTheme()
		theme.Background = ui.RGB(33, 39, 46)
		theme.Card = theme.Background
		theme.CardElevated = ui.RGB(43, 51, 60)
		theme.Popover = ui.RGB(40, 47, 56)
		theme.Muted = theme.CardElevated
		theme.Secondary = theme.CardElevated
		theme.Hover = ui.RGB(48, 57, 67)
		theme.Accent = theme.Hover
		theme.MutedForeground = ui.RGB(166, 175, 187)
		theme.Primary = ui.RGB(59, 130, 246)
		theme.PrimaryForeground = ui.RGB(255, 255, 255)
		theme.Ring = theme.Primary
		theme.Border = ui.RGBA(255, 255, 255, 0.10)
		theme.Input = ui.RGBA(255, 255, 255, 0.12)
	} else {
		theme = gfx.LightTheme()
		theme.Primary = ui.RGB(37, 99, 235)
		theme.PrimaryForeground = ui.RGB(255, 255, 255)
		theme.Ring = theme.Primary
	}

	if preset, ok := schemePresets[scheme]; ok {
		if dark {
			preset.dark.apply(&theme)
		} else {
			preset.light.apply(&theme)
		}
	}
	theme.Metrics.RadiusSurface = RadiusCard
	theme = theme.WithRadius(RadiusControl)

	return theme, buildPalette(theme, dark, scheme)
}

func buildPalette(t gfx.Theme, dark bool, scheme string) Palette {
	if _, ok := schemePresets[scheme]; ok {
		rail := t.Background.Mix(t.Foreground, 0.09)
		incoming := t.Card
		mutedBadgeBg := ui.RGB(164, 171, 180)
		mutedBadgeFg := ui.RGB(255, 255, 255)
		if dark {
			rail = t.Background.Mix(ui.RGB(0, 0, 0), 0.22)
			incoming = t.CardElevated
			mutedBadgeBg = ui.RGB(88, 95, 105)
			mutedBadgeFg = ui.RGB(228, 232, 238)
		}
		chatSurf := t.Background.Mix(t.Card, 0.72)
		return Palette{
			Rail:          rail,
			Sidebar:       t.Background,
			Chat:          chatSurf,
			Header:        chatSurf,
			Border:        t.Border,
			Hover:         t.Hover,
			Selected:      t.Selection,
			SelectedText:  t.Primary,
			Icon:          t.MutedForeground,
			SecondaryText: t.MutedForeground,
			MutedAvatar:   t.CardElevated,
			AvatarText:    t.Foreground,
			Incoming:      incoming,
			Own:           t.Selection,
			BubbleBorder:  t.Foreground.WithAlpha(0.09),
			OwnBorder:     t.Primary.WithAlpha(0.25),
			FileCard:      t.CardElevated,
			FileBorder:    t.Border,
			BadgeMutedBg:  mutedBadgeBg,
			BadgeMutedFg:  mutedBadgeFg,
			Success:       ui.RGB(55, 197, 118),
			Danger:        ui.RGB(239, 68, 68),
			Warning:       ui.RGB(245, 158, 11),
		}
	}

	if dark {
		chatSurf := ui.RGB(38, 45, 54)
		return Palette{
			Rail:          ui.RGB(24, 29, 36),
			Sidebar:       t.Background,
			Chat:          chatSurf,
			Header:        chatSurf,
			Border:        ui.RGBA(255, 255, 255, 0.09),
			Hover:         t.Hover,
			Selected:      ui.RGB(44, 66, 98),
			SelectedText:  ui.RGB(152, 195, 255),
			Icon:          ui.RGB(205, 210, 220),
			SecondaryText: t.MutedForeground,
			MutedAvatar:   t.CardElevated,
			AvatarText:    ui.RGB(225, 231, 241),
			Incoming:      ui.RGB(52, 60, 71),
			Own:           ui.RGB(32, 62, 99),
			BubbleBorder:  ui.RGBA(255, 255, 255, 0.06),
			OwnBorder:     ui.RGBA(96, 165, 250, 0.22),
			FileCard:      ui.RGB(44, 52, 64),
			FileBorder:    ui.RGBA(172, 196, 230, 0.14),
			BadgeMutedBg:  ui.RGB(88, 95, 105),
			BadgeMutedFg:  ui.RGB(228, 232, 238),
			Success:       ui.RGB(55, 197, 118),
			Danger:        ui.RGB(255, 112, 116),
			Warning:       ui.RGB(245, 158, 11),
		}
	}

	chatSurf := ui.RGB(250, 251, 253)
	return Palette{
		Rail:          ui.RGB(227, 233, 241),
		Sidebar:       ui.RGB(240, 243, 248),
		Chat:          chatSurf,
		Header:        chatSurf,
		Border:        ui.RGB(231, 235, 241),
		Hover:         ui.RGB(231, 236, 244),
		Selected:      ui.RGB(215, 230, 253),
		SelectedText:  ui.RGB(31, 80, 191),
		Icon:          ui.RGB(78, 86, 100),
		SecondaryText: ui.RGB(96, 107, 124),
		MutedAvatar:   ui.RGB(233, 238, 246),
		AvatarText:    ui.RGB(60, 72, 92),
		Incoming:      ui.RGB(237, 241, 247),
		Own:           ui.RGB(222, 236, 255),
		BubbleBorder:  ui.RGBA(30, 64, 120, 0.06),
		OwnBorder:     ui.RGBA(59, 130, 246, 0.20),
		FileCard:      ui.RGB(244, 248, 255),
		FileBorder:    ui.RGB(208, 224, 250),
		BadgeMutedBg:  ui.RGB(168, 175, 185),
		BadgeMutedFg:  ui.RGB(255, 255, 255),
		Success:       ui.RGB(34, 176, 94),
		Danger:        ui.RGB(220, 48, 54),
		Warning:       ui.RGB(217, 119, 6),
	}
}

// WindowControlsStyle returns custom-titlebar button styling aligned with the active Palette.
func (m *Model) WindowControlsStyle() app.WindowControlsStyle {
	p := m.Palette
	return app.WindowControlsStyle{
		Foreground:      p.Icon,
		Hover:           p.Hover,
		Pressed:         p.Selected,
		Close:           ui.RGB(218, 54, 51),
		CloseForeground: ui.RGB(255, 255, 255),
		Minimize:        "Minimize",
		Maximize:        "Maximize",
		Restore:         "Restore",
		CloseName:       "Close",
	}
}

// avatarAccentColor deterministically assigns a pleasant muted avatar tint for contacts/chats.
func avatarAccentColor(seed string, dark bool) (bg, fg gfx.Color) {
	lightSet := []struct{ bg, fg gfx.Color }{
		{ui.RGB(219, 234, 254), ui.RGB(29, 78, 216)},
		{ui.RGB(209, 250, 229), ui.RGB(4, 120, 87)},
		{ui.RGB(254, 243, 199), ui.RGB(180, 83, 9)},
		{ui.RGB(237, 233, 254), ui.RGB(109, 40, 217)},
		{ui.RGB(252, 231, 243), ui.RGB(190, 24, 93)},
		{ui.RGB(207, 250, 254), ui.RGB(14, 116, 144)},
	}
	darkSet := []struct{ bg, fg gfx.Color }{
		{ui.RGB(37, 63, 102), ui.RGB(147, 197, 253)},
		{ui.RGB(25, 76, 62), ui.RGB(110, 231, 183)},
		{ui.RGB(88, 61, 29), ui.RGB(252, 211, 77)},
		{ui.RGB(67, 46, 102), ui.RGB(196, 181, 253)},
		{ui.RGB(92, 42, 68), ui.RGB(249, 168, 212)},
		{ui.RGB(28, 72, 86), ui.RGB(103, 232, 249)},
	}
	var h uint32
	for _, r := range seed {
		h = h*31 + uint32(r)
	}
	idx := int(h) % len(lightSet)
	if dark {
		return darkSet[idx].bg, darkSet[idx].fg
	}
	return lightSet[idx].bg, lightSet[idx].fg
}

// fileBadgeVisual returns GoIM-style file type icon, short tag, and semantic tint colors.
func fileBadgeVisual(name string, viewType string, dark bool) (icon gfx.Icon, tag string, bg, fg gfx.Color) {
	if viewType == "Webxdc" || strings.EqualFold(filepath.Ext(name), ".xdc") {
		if dark {
			return IconApp, "XDC", ui.RGB(58, 52, 98), ui.RGB(165, 180, 252)
		}
		return IconApp, "XDC", ui.RGB(238, 242, 255), ui.RGB(79, 70, 229)
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".pdf" {
		if dark {
			return IconFile, "PDF", ui.RGB(94, 41, 45), ui.RGB(252, 165, 165)
		}
		return IconFile, "PDF", ui.RGB(254, 226, 226), ui.RGB(220, 38, 38)
	}
	kind := preview.DetectKind(name)
	switch kind {
	case preview.KindImage:
		if dark {
			return IconImage, "IMG", ui.RGB(27, 74, 60), ui.RGB(110, 231, 183)
		}
		return IconImage, "IMG", ui.RGB(209, 250, 229), ui.RGB(5, 150, 105)
	case preview.KindMarkdown:
		if dark {
			return IconCode, "MD", ui.RGB(35, 62, 99), ui.RGB(147, 197, 253)
		}
		return IconCode, "MD", ui.RGB(219, 234, 254), ui.RGB(37, 99, 235)
	case preview.KindCSV:
		if dark {
			return IconTable, "CSV", ui.RGB(24, 76, 73), ui.RGB(94, 234, 212)
		}
		return IconTable, "CSV", ui.RGB(204, 251, 241), ui.RGB(13, 148, 136)
	case preview.KindJSON, preview.KindCode:
		if dark {
			return IconCode, "CODE", ui.RGB(66, 48, 96), ui.RGB(196, 181, 253)
		}
		return IconCode, "CODE", ui.RGB(237, 233, 254), ui.RGB(124, 58, 237)
	case preview.KindArchive:
		if dark {
			return IconArchive, "ZIP", ui.RGB(88, 63, 28), ui.RGB(252, 211, 77)
		}
		return IconArchive, "ZIP", ui.RGB(254, 243, 199), ui.RGB(217, 119, 6)
	case preview.KindVCard:
		if dark {
			return IconUsers, "VCF", ui.RGB(30, 68, 86), ui.RGB(103, 232, 249)
		}
		return IconUsers, "VCF", ui.RGB(207, 250, 254), ui.RGB(8, 145, 178)
	default:
		if dark {
			return IconFile, "FILE", ui.RGB(53, 62, 74), ui.RGB(196, 205, 218)
		}
		return IconFile, "FILE", ui.RGB(232, 237, 245), ui.RGB(71, 85, 105)
	}
}
