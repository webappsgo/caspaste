// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const baseTheme = "dark/dracula"
const embThemesDir = "data/theme"

type Theme map[string]string
type Themes map[string]Theme

type ThemesListPart map[string]string
type ThemesList map[string]ThemesListPart

// ThemeGroupItem is a single theme entry for display in a grouped selector.
type ThemeGroupItem struct {
	Key  string
	Name string
}

// ThemeGroup is a labeled group of themes for <optgroup> rendering.
type ThemeGroup struct {
	Label string
	Items []ThemeGroupItem
}

func loadThemes(hostThemeDir string, localesList LocalesList, defaultTheme string) (Themes, ThemesList, error) {
	// Normalize default theme aliases before loading
	themeAliases := map[string]string{
		"dark":  "dark/dracula",
		"light": "light/github",
		// fallback for auto
		"auto":  "dark/dracula",
	}
	if normalized, exists := themeAliases[defaultTheme]; exists {
		defaultTheme = normalized
	}

	themes := make(Themes)
	themesList := make(ThemesList)

	for localeCode, _ := range localesList {
		themesList[localeCode] = make(ThemesListPart)
	}

	// Prepare load FS function
	loadThemesFromFS := func(f fs.FS, themeDir string) error {
		// Load themes from directory recursively
		var loadDir func(string, string) error
		loadDir = func(dir string, prefix string) error {
			files, err := fs.ReadDir(f, dir)
			if err != nil {
				return errors.New("web: failed read dir '" + dir + "': " + err.Error())
			}

			for _, fileInfo := range files {
				// Check file
				if fileInfo.IsDir() {
					// Recursively load subdirectories (dark/, light/)
					subdir := filepath.Join(dir, fileInfo.Name())
					if err := loadDir(subdir, fileInfo.Name()+"/"); err != nil {
						return err
					}
					continue
				}

				fileName := fileInfo.Name()
				if !strings.HasSuffix(fileName, ".theme") {
					continue
				}
				themeCode := prefix + fileName[:len(fileName)-6]

				// Read file
				filePath := filepath.Join(dir, fileName)
				fileByte, err := fs.ReadFile(f, filePath)
				if err != nil {
					return errors.New("web: failed open file '" + filePath + "': " + err.Error())
				}

				fileStr := bytes.NewBuffer(fileByte).String()

				// Load theme
				theme, err := readKVCfg(fileStr)
				if err != nil {
					return errors.New("web: failed read file '" + filePath + "': " + err.Error())
				}

				_, exists := themes[themeCode]
				if exists {
					return errors.New("web: theme alredy loaded: " + filePath)
				}

				themes[themeCode] = Theme(theme)
			}
			return nil
		}

		return loadDir(themeDir, "")
	}

	// Load embed themes
	err := loadThemesFromFS(embFS, embThemesDir)
	if err != nil {
		return nil, nil, err
	}

	// Load external themes (if directory exists)
	if hostThemeDir != "" {
		if _, statErr := os.Stat(hostThemeDir); statErr == nil {
			err = loadThemesFromFS(os.DirFS(hostThemeDir), ".")
			if err != nil {
				return nil, nil, err
			}
		}
	}

	// Prepare themes list
	for key, val := range themes {
		// Get theme name
		themeName := val["theme.Name."+baseLocale]
		if themeName == "" {
			return nil, nil, errors.New("web: empty theme.Name." + baseLocale + " parameter in '" + key + "' theme")
		}

		// Append to the translation, if it is not complete
		defTheme := themes[baseTheme]
		defTotal := len(defTheme)
		curTotal := 0
		for defKey, defVal := range defTheme {
			_, isExist := val[defKey]
			if isExist {
				curTotal = curTotal + 1
			} else {
				if strings.HasPrefix(defKey, "theme.Name.") {
					val[defKey] = val["theme.Name."+baseLocale]
				} else {
					val[defKey] = defVal
				}
			}
		}

		if curTotal == 0 {
			return nil, nil, errors.New("web: theme '" + key + "' is empty")
		}

		// Add theme to themes list
		themeNameSuffix := ""
		if curTotal != defTotal {
			themeNameSuffix = fmt.Sprintf(" (%.2f%%)", (float32(curTotal)/float32(defTotal))*100)
		}
		themesList[baseLocale][key] = themeName + themeNameSuffix

		for localeCode, _ := range localesList {
			result, ok := val["theme.Name."+localeCode]
			if ok {
				themesList[localeCode][key] = result + themeNameSuffix
			} else {
				themesList[localeCode][key] = themeName + themeNameSuffix
			}
		}
	}

	// Check default theme exist
	_, ok := themes[defaultTheme]
	if !ok {
		return nil, nil, errors.New("web: default theme '" + defaultTheme + "' not found")
	}

	// "auto" theme: the auto.theme file is loaded above; only set a fallback if it was not present.
	if _, autoExists := themes["auto"]; !autoExists {
		themes["auto"] = themes[defaultTheme]
		for localeCode := range themesList {
			themesList[localeCode]["auto"] = "Auto (System)"
		}
	}

	return themes, themesList, nil
}

func (themesList ThemesList) getForLocale(req *http.Request) ThemesListPart {
	// Get theme by cookie
	langCookie := getCookie(req, "lang")
	if langCookie != "" {
		theme, ok := themesList[langCookie]
		if ok {
			return theme
		}
	}

	// Load default part theme
	theme, _ := themesList[baseLocale]
	return theme
}

// getGroupedForLocale returns themes organised into labeled groups (Auto, Dark, Light, Other)
// suitable for rendering with <optgroup> in a settings selector.
func (themesList ThemesList) getGroupedForLocale(req *http.Request) []ThemeGroup {
	flat := themesList.getForLocale(req)

	auto := ThemeGroup{Label: "Auto"}
	dark := ThemeGroup{Label: "Dark"}
	light := ThemeGroup{Label: "Light"}
	other := ThemeGroup{Label: "Other"}

	for key, name := range flat {
		item := ThemeGroupItem{Key: key, Name: name}
		switch {
		case key == "auto":
			auto.Items = append(auto.Items, item)
		case strings.HasPrefix(key, "dark/"):
			dark.Items = append(dark.Items, item)
		case strings.HasPrefix(key, "light/"):
			light.Items = append(light.Items, item)
		default:
			other.Items = append(other.Items, item)
		}
	}

	// Sort items within each group by name for stable display order
	sortItems := func(items []ThemeGroupItem) {
		for i := 1; i < len(items); i++ {
			for j := i; j > 0 && items[j].Name < items[j-1].Name; j-- {
				items[j], items[j-1] = items[j-1], items[j]
			}
		}
	}
	sortItems(auto.Items)
	sortItems(dark.Items)
	sortItems(light.Items)
	sortItems(other.Items)

	// Build result: omit empty groups; Auto always first
	var groups []ThemeGroup
	if len(auto.Items) > 0 {
		groups = append(groups, auto)
	}
	if len(dark.Items) > 0 {
		groups = append(groups, dark)
	}
	if len(light.Items) > 0 {
		groups = append(groups, light)
	}
	if len(other.Items) > 0 {
		groups = append(groups, other)
	}
	return groups
}

func (themes Themes) findTheme(req *http.Request, defaultTheme string) Theme {
	// Get theme by cookie
	themeCookie := getCookie(req, "theme")
	if themeCookie != "" {
		theme, ok := themes[themeCookie]
		if ok {
			return theme
		}
	}

	// Load default theme
	theme, _ := themes[defaultTheme]
	return theme
}

func (theme Theme) theme(s string) string {
	for key, val := range theme {
		if key == s {
			return val
		}
	}

	panic(errors.New("web: theme: unknown theme key: " + s))
}

func (theme Theme) tryHighlight(source string, lexer string) template.HTML {
	return tryHighlight(source, lexer, theme.theme("highlight.Theme"))
}
