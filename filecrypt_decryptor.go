// ==============================================================================
// Filecrypt Decryptor v2.0.0 (Go Native Engine)
// Copyright (C) 2026 ArvindSaini978 (https://github.com/ArvindSaini978)
// Repository: https://github.com/ArvindSaini978/Filecrypt-Decryptor
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
// ==============================================================================

package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unsafe"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/mattn/go-runewidth"
)

// ==============================================================================
// APPLICATION METADATA & UI WIDTH
// ==============================================================================

const (
	UIWidth          = 108
	AppTitle         = "FILECRYPT DECRYPTOR"
	AppVersion       = "v2.0.0"
	AppAuthor        = "ArvindSaini978"
	AppRepoURL       = "https://github.com/ArvindSaini978/Filecrypt-Decryptor"
	AppGithubProfile = "https://github.com/ArvindSaini978"
	AppLicense       = "GPL-3.0 License"
	AppLicenseURL    = "https://github.com/ArvindSaini978/Filecrypt-Decryptor/blob/master/LICENSE"
	AppYear          = "2026"
)

// ==============================================================================
// SOFT NORD / CATPPUCCIN MOCHA COLOR PALETTE (24-BIT TRUECOLOR ANSI)
// ==============================================================================

const (
	Reset          = "\x1b[0m"
	Bold           = "\x1b[1m"
	BorderSubtle   = "\x1b[38;2;59;66;82m"    // #3b4252
	BorderAccent   = "\x1b[38;2;94;129;172m"  // #5e81ac
	TextMain       = "\x1b[38;2;236;239;244m" // #eceff4
	TextMuted      = "\x1b[38;2;127;132;156m" // #7f849c
	AccentTeal     = "\x1b[38;2;136;192;208m" // #88c0d0
	AccentLavender = "\x1b[38;2;180;190;254m" // #b4befe
	AccentGreen    = "\x1b[38;2;163;190;140m" // #a3be8c
	AccentAmber    = "\x1b[38;2;235;203;139m" // #ebcb8b
	AccentCoral    = "\x1b[38;2;191;97;106m"  // #bf616a
	AccentMauve    = "\x1b[38;2;203;166;247m" // #cba6f7
)

var (
	AppDir     string
	ConfigPath string
	ansiRegex  = regexp.MustCompile(`\x1b\[[0-9;]*m`)

	ErrUserCancelled = errors.New("cancelled by user")
	ErrBrowserClosed = errors.New("browser or tab closed")

	sharedHTTPTransport = &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		MaxIdleConns:        32,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     60 * time.Second,
	}
)

// ==============================================================================
// WINDOWS NATIVE CONSOLE & MSVCRT FFI
// ==============================================================================

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleTitleW           = kernel32.NewProc("SetConsoleTitleW")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procSetConsoleOutputCP         = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP               = kernel32.NewProc("SetConsoleCP")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procSetConsoleScreenBufferSize = kernel32.NewProc("SetConsoleScreenBufferSize")
	procSetConsoleWindowInfo       = kernel32.NewProc("SetConsoleWindowInfo")
	msvcrt                         = syscall.NewLazyDLL("msvcrt.dll")
	procKbhit                      = msvcrt.NewProc("_kbhit")
	procGetwch                     = msvcrt.NewProc("_getwch")
)

type coord struct {
	X int16
	Y int16
}

type smallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

func packCoord(c coord) uintptr {
	return uintptr(uint32(uint16(c.X)) | (uint32(uint16(c.Y)) << 16))
}

func initWindowsConsole() {
	if runtime.GOOS != "windows" {
		return
	}
	procSetConsoleOutputCP.Call(65001)
	procSetConsoleCP.Call(65001)

	const enableVirtualTerminalProcessing = 0x0004
	for _, handleID := range []int{syscall.STD_OUTPUT_HANDLE, syscall.STD_ERROR_HANDLE} {
		h, err := syscall.GetStdHandle(handleID)
		if err == nil && h != syscall.InvalidHandle {
			var mode uint32
			procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
			procSetConsoleMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
		}
	}

	hOut, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err == nil && hOut != syscall.InvalidHandle {
		var info consoleScreenBufferInfo
		ret, _, _ := procGetConsoleScreenBufferInfo.Call(uintptr(hOut), uintptr(unsafe.Pointer(&info)))
		if ret != 0 {
			width := int(info.Window.Right - info.Window.Left + 1)
			targetCols := int16(UIWidth + 6)
			targetLines := int16(42)
			if width < UIWidth+2 {
				newBufSize := info.Size
				if newBufSize.X < targetCols {
					newBufSize.X = targetCols
				}
				if newBufSize.Y < targetLines {
					newBufSize.Y = 3000
				}
				procSetConsoleScreenBufferSize.Call(uintptr(hOut), packCoord(newBufSize))
				rect := smallRect{
					Left:   0,
					Top:    0,
					Right:  targetCols - 1,
					Bottom: targetLines - 1,
				}
				procSetConsoleWindowInfo.Call(uintptr(hOut), 1, uintptr(unsafe.Pointer(&rect)))
			}
		}
	}
}

func setWindowTitle(subtitle string) {
	fullTitle := fmt.Sprintf("Filecrypt Decryptor %s", AppVersion)
	if subtitle != "" {
		fullTitle = fmt.Sprintf("%s  —  %s", fullTitle, subtitle)
	}
	if runtime.GOOS == "windows" {
		ptr, err := syscall.UTF16PtrFromString(fullTitle)
		if err == nil {
			procSetConsoleTitleW.Call(uintptr(unsafe.Pointer(ptr)))
		}
	} else {
		fmt.Printf("\033]0;%s\007", fullTitle)
	}
}

func clearScreen() {
	fmt.Print("\x1b[2J\x1b[3J\x1b[H")
}

func kbhit() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	ret, _, _ := procKbhit.Call()
	return ret != 0
}

func getwch() rune {
	if runtime.GOOS != "windows" {
		return 0
	}
	ret, _, _ := procGetwch.Call()
	return rune(ret)
}

func drainKeyboardBuffer() {
	if runtime.GOOS != "windows" {
		return
	}
	for kbhit() {
		getwch()
	}
}

func checkCancelKeyPressed() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	for kbhit() {
		ch := getwch()
		if ch == 0x03 {
			fmt.Printf("\n  %sExited cleanly.%s\n", AccentAmber, Reset)
			os.Exit(0)
		}
		if ch == 0x00 || ch == 0xe0 {
			if kbhit() {
				getwch()
			}
			continue
		}
		if ch == 0x1b || ch == 'x' || ch == 'X' {
			return true
		}
	}
	return false
}

func printCancelTip(destination string) {
	fmt.Printf("  %sTip: Press%s %s[Esc]%s %sor%s %s[X]%s %sat any time to cancel and return to %s.%s\n\n",
		TextMuted, Reset,
		Bold+AccentAmber, Reset,
		TextMuted, Reset,
		Bold+AccentAmber, Reset,
		TextMuted, destination, Reset)
}

// ==============================================================================
// TRUECOLOR ANSI TEXT & RICH-ACCURATE BOX / TABLE LAYOUT ENGINE
// ==============================================================================

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func visibleWidth(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

func padRight(s string, width int) string {
	w := visibleWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func padLeft(s string, width int) string {
	w := visibleWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

func padCenter(s string, width int) string {
	w := visibleWidth(s)
	if w >= width {
		return s
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func decodeRuneInString(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 0
}

type styledRune struct {
	r     rune
	style string
	width int
}

func parseStyledRunes(s string) []styledRune {
	var res []styledRune
	curStyle := ""
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			start := i
			i++
			for i < len(s) {
				ch := s[i]
				i++
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
					break
				}
			}
			escSeq := s[start:i]
			if escSeq == Reset {
				curStyle = ""
			} else {
				curStyle += escSeq
			}
			continue
		}
		r, size := decodeRuneInString(s[i:])
		res = append(res, styledRune{
			r:     r,
			style: curStyle,
			width: runewidth.RuneWidth(r),
		})
		i += size
	}
	return res
}

func renderStyledRunes(chars []styledRune) string {
	if len(chars) == 0 {
		return ""
	}
	var sb strings.Builder
	curStyle := ""
	for _, c := range chars {
		if c.style != curStyle {
			sb.WriteString(Reset)
			sb.WriteString(c.style)
			curStyle = c.style
		}
		sb.WriteRune(c.r)
	}
	if curStyle != "" {
		sb.WriteString(Reset)
	}
	return sb.String()
}

func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if visibleWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	chars := parseStyledRunes(s)
	curW := 0
	endIdx := 0
	for endIdx < len(chars) {
		if curW+chars[endIdx].width > maxWidth-1 {
			break
		}
		curW += chars[endIdx].width
		endIdx++
	}
	return renderStyledRunes(chars[:endIdx]) + Reset + TextMuted + "…" + Reset
}

var episodeTokenRegex = regexp.MustCompile(`(?i)(?:^|[._\-\s\[])(S\d{1,2}E\d{1,3}|E\d{1,3}|EP\.?\d{1,3}|Part\.?\d{1,3}|Pt\.?\d{1,3})(?:$|[._\-\s\]])`)

// truncateMiddlePlain truncates a plain filename in the middle ("start…end") if it exceeds
// maxWidth, while preserving any episode/part identifier (e.g. S01E04, Part.2) so consecutive
// episodes never look identical when truncated.
func truncateMiddlePlain(s string, maxWidth int, mainStyle string) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return mainStyle + s + Reset
	}
	if maxWidth <= 5 {
		return mainStyle + runewidth.Truncate(s, maxWidth, "…") + Reset
	}

	tailLen := (maxWidth - 1) / 3
	if tailLen < 12 && maxWidth > 26 {
		tailLen = 12
	}
	headLen := maxWidth - 1 - tailLen

	// If an episode/part token (like S01E04) would be cut off by headLen, keep the episode
	// token visible right before the ellipsis so every episode row is clearly distinguishable.
	if subLoc := episodeTokenRegex.FindStringSubmatchIndex(s); len(subLoc) >= 4 {
		epStart, epEnd := subLoc[2], subLoc[3]
		epEndWidth := runewidth.StringWidth(s[:epEnd])
		if epEndWidth > headLen {
			epToken := s[epStart:epEnd]
			epW := runewidth.StringWidth(epToken)
			if epW+6 < headLen {
				prefixBudget := headLen - epW - 1
				runes := []rune(s[:epStart])
				pEnd := 0
				pw := 0
				for i, r := range runes {
					rw := runewidth.RuneWidth(r)
					if pw+rw > prefixBudget {
						break
					}
					pw += rw
					pEnd = i + 1
				}

				allRunes := []rune(s)
				tailStart := len(allRunes)
				tw := 0
				for i := len(allRunes) - 1; i >= 0; i-- {
					rw := runewidth.RuneWidth(allRunes[i])
					if tw+rw > tailLen {
						break
					}
					tw += rw
					tailStart = i
				}

				return mainStyle + string(runes[:pEnd]) + Reset + TextMuted + "…" + Reset +
					Bold + AccentTeal + epToken + Reset +
					TextMuted + "…" + Reset + mainStyle + string(allRunes[tailStart:]) + Reset
			}
		}
	}

	runes := []rune(s)
	headEnd := 0
	hw := 0
	for i, r := range runes {
		rw := runewidth.RuneWidth(r)
		if hw+rw > headLen {
			break
		}
		hw += rw
		headEnd = i + 1
	}

	tailStart := len(runes)
	tw := 0
	for i := len(runes) - 1; i >= headEnd; i-- {
		rw := runewidth.RuneWidth(runes[i])
		if tw+rw > tailLen {
			break
		}
		tw += rw
		tailStart = i
	}

	return mainStyle + string(runes[:headEnd]) + Reset + TextMuted + "…" + Reset + mainStyle + string(runes[tailStart:]) + Reset
}

func wrapSingleStyledLine(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{""}
	}
	if visibleWidth(line) <= maxWidth {
		return []string{line}
	}

	chars := parseStyledRunes(line)
	if len(chars) == 0 {
		return []string{""}
	}

	var out []string
	idx := 0
	n := len(chars)

	for idx < n {
		if len(out) > 0 {
			for idx < n && chars[idx].r == ' ' {
				idx++
			}
			if idx >= n {
				break
			}
		}

		curW := 0
		endIdx := idx
		lastSpaceIdx := -1

		for endIdx < n {
			w := chars[endIdx].width
			if curW+w > maxWidth {
				break
			}
			if chars[endIdx].r == ' ' {
				lastSpaceIdx = endIdx
			}
			curW += w
			endIdx++
		}

		if endIdx == n {
			out = append(out, renderStyledRunes(chars[idx:n]))
			break
		}

		if lastSpaceIdx > idx {
			out = append(out, renderStyledRunes(chars[idx:lastSpaceIdx]))
			idx = lastSpaceIdx + 1
		} else {
			if endIdx == idx {
				endIdx = idx + 1
			}
			out = append(out, renderStyledRunes(chars[idx:endIdx]))
			idx = endIdx
		}
	}

	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func wrapStyledText(text string, maxWidth int) []string {
	rawLines := strings.Split(text, "\n")
	var out []string
	for _, line := range rawLines {
		wrapped := wrapSingleStyledLine(line, maxWidth)
		out = append(out, wrapped...)
	}
	return out
}

func renderPanel(lines []string, title string, borderColor string, padY int, padX int) string {
	innerWidth := UIWidth - 2
	contentWidth := innerWidth - (padX * 2)
	padXStr := strings.Repeat(" ", padX)

	var sb strings.Builder

	if title == "" {
		sb.WriteString(borderColor)
		sb.WriteString("╭")
		sb.WriteString(strings.Repeat("─", innerWidth))
		sb.WriteString("╮")
		sb.WriteString(Reset)
		sb.WriteString("\n")
	} else {
		titleSegmented := " " + title + " "
		tWidth := visibleWidth(titleSegmented)
		rem := innerWidth - 1 - tWidth
		if rem < 1 {
			rem = 1
		}
		sb.WriteString(borderColor)
		sb.WriteString("╭─")
		sb.WriteString(Reset)
		sb.WriteString(titleSegmented)
		sb.WriteString(borderColor)
		sb.WriteString(strings.Repeat("─", rem))
		sb.WriteString("╮")
		sb.WriteString(Reset)
		sb.WriteString("\n")
	}

	emptyLine := borderColor + "│" + Reset + strings.Repeat(" ", innerWidth) + borderColor + "│" + Reset + "\n"
	for i := 0; i < padY; i++ {
		sb.WriteString(emptyLine)
	}

	for _, rawLine := range lines {
		for _, l := range wrapStyledText(rawLine, contentWidth) {
			padded := padRight(l, contentWidth)
			sb.WriteString(borderColor)
			sb.WriteString("│")
			sb.WriteString(Reset)
			sb.WriteString(padXStr)
			sb.WriteString(padded)
			sb.WriteString(padXStr)
			sb.WriteString(borderColor)
			sb.WriteString("│")
			sb.WriteString(Reset)
			sb.WriteString("\n")
		}
	}

	for i := 0; i < padY; i++ {
		sb.WriteString(emptyLine)
	}

	sb.WriteString(borderColor)
	sb.WriteString("╰")
	sb.WriteString(strings.Repeat("─", innerWidth))
	sb.WriteString("╯")
	sb.WriteString(Reset)
	return sb.String()
}

func printPanel(lines []string, title string, borderColor string, padY int, padX int) {
	fmt.Println(renderPanel(lines, title, borderColor, padY, padX))
}

type tableCol struct {
	Header  string
	Width   int
	Justify string
	NoWrap  bool
}

type tableRow struct {
	IsSection bool
	Cells     []string
}

func renderTable(cols []tableCol, rows []tableRow, heavyHeader bool, totalInnerWidth int) []string {
	numCols := len(cols)
	overhead := (3 * numCols) - 1
	availContentWidth := totalInnerWidth - overhead

	fixedSum := 0
	flexCount := 0
	for _, c := range cols {
		if c.Width > 0 {
			fixedSum += c.Width
		} else {
			flexCount++
		}
	}

	availForFlex := availContentWidth - fixedSum
	flexColWidth := 10
	if flexCount > 0 && availForFlex > flexCount*4 {
		flexColWidth = availForFlex / flexCount
	}

	colWidths := make([]int, numCols)
	usedContent := 0
	for i, c := range cols {
		if c.Width > 0 {
			colWidths[i] = c.Width
		} else {
			colWidths[i] = flexColWidth
		}
		usedContent += colWidths[i]
	}
	if usedContent < availContentWidth {
		diff := availContentWidth - usedContent
		added := false
		for i, c := range cols {
			if c.Width == 0 {
				colWidths[i] += diff
				added = true
				break
			}
		}
		if !added && numCols > 0 {
			colWidths[numCols-1] += diff
		}
	}

	formatCellLine := func(text string, w int, justify string) string {
		switch justify {
		case "center":
			return padCenter(text, w)
		case "right":
			return padLeft(text, w)
		default:
			return padRight(text, w)
		}
	}

	var out []string
	out = append(out, "")

	var hdrBuilder strings.Builder
	for i, c := range cols {
		if i > 0 {
			hdrBuilder.WriteString(" ")
		}
		cellText := Bold + TextMuted + c.Header + Reset
		hdrBuilder.WriteString(" ")
		hdrBuilder.WriteString(formatCellLine(cellText, colWidths[i], c.Justify))
		hdrBuilder.WriteString(" ")
	}
	out = append(out, hdrBuilder.String())

	ruleChar := "─"
	if heavyHeader {
		ruleChar = "━"
	}
	out = append(out, BorderSubtle+strings.Repeat(ruleChar, totalInnerWidth)+Reset)

	// Pre-calculate uniform column widths for spanned (bottom-section) rows so that if any
	// bottom action row needs a wider col[1] (e.g. "Decrypt All Links"), ALL bottom action
	// rows use that exact same col[1] width and stay 100% vertically aligned!
	spannedCol1Boost := 0
	if numCols >= 3 && cols[1].Justify == "left" && cols[1].NoWrap {
		for _, r := range rows {
			if r.IsSection {
				continue
			}
			lastVal := ""
			if numCols-1 < len(r.Cells) {
				lastVal = r.Cells[numCols-1]
			}
			if lastVal == "" && len(r.Cells) > 1 {
				w := visibleWidth(r.Cells[1])
				if w-colWidths[1] > spannedCol1Boost {
					spannedCol1Boost = w - colWidths[1]
				}
			}
		}
		if spannedCol1Boost > 8 {
			spannedCol1Boost = 8
		}
	}

	for _, r := range rows {
		if r.IsSection {
			out = append(out, BorderSubtle+strings.Repeat("─", totalInnerWidth)+Reset)
			continue
		}

		// Collapse all trailing empty columns into the last non-empty left-aligned NoWrap
		// column so footer action descriptions span cleanly across the remaining width.
		rowColWidths := make([]int, numCols)
		copy(rowColWidths, colWidths)
		activeCols := numCols

		lastNonEmpty := -1
		for i := numCols - 1; i >= 0; i-- {
			val := ""
			if i < len(r.Cells) {
				val = r.Cells[i]
			}
			if val != "" {
				lastNonEmpty = i
				break
			}
		}
		if lastNonEmpty >= 0 && lastNonEmpty < numCols-1 &&
			cols[lastNonEmpty].Justify == "left" && cols[lastNonEmpty].NoWrap {
			extraWidth := 0
			for i := lastNonEmpty + 1; i < numCols; i++ {
				extraWidth += 3 + rowColWidths[i]
			}
			rowColWidths[lastNonEmpty] += extraWidth
			activeCols = lastNonEmpty + 1

			// Apply the uniform bottom-section boost to col[1] on all spanned rows so
			// [N], [P], [A], and [Esc] descriptions start at the exact same column!
			if lastNonEmpty >= 2 && spannedCol1Boost > 0 && rowColWidths[lastNonEmpty]-spannedCol1Boost >= 20 {
				rowColWidths[1] += spannedCol1Boost
				rowColWidths[lastNonEmpty] -= spannedCol1Boost
			}
		}

		wrappedCells := make([][]string, activeCols)
		maxLines := 1
		for i := 0; i < activeCols; i++ {
			val := ""
			if i < len(r.Cells) {
				val = r.Cells[i]
			}
			var linesForCell []string
			if cols[i].NoWrap {
				linesForCell = []string{truncateWithEllipsis(val, rowColWidths[i])}
			} else {
				linesForCell = wrapStyledText(val, rowColWidths[i])
			}
			if len(linesForCell) == 0 {
				linesForCell = []string{""}
			}
			wrappedCells[i] = linesForCell
			if len(linesForCell) > maxLines {
				maxLines = len(linesForCell)
			}
		}

		for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
			var rowBuilder strings.Builder
			for colIdx := 0; colIdx < activeCols; colIdx++ {
				if colIdx > 0 {
					rowBuilder.WriteString(" ")
				}
				cellLine := ""
				if lineIdx < len(wrappedCells[colIdx]) {
					cellLine = wrappedCells[colIdx][lineIdx]
				}
				rowBuilder.WriteString(" ")
				rowBuilder.WriteString(formatCellLine(cellLine, rowColWidths[colIdx], cols[colIdx].Justify))
				rowBuilder.WriteString(" ")
			}
			out = append(out, rowBuilder.String())
		}
	}

	out = append(out, "")
	return out
}

// ==============================================================================
// HEADER, PREVIEW, AND INFO PANELS
// ==============================================================================

func printAppHeader(breadcrumbPath string, _ *AppConfig, clear bool, windowSubtitle string) {
	if clear {
		clearScreen()
	}

	rawParts := strings.Split(breadcrumbPath, "/")
	var parts []string
	for _, p := range rawParts {
		parts = append(parts, strings.TrimSpace(p))
	}

	if windowSubtitle != "" {
		setWindowTitle(windowSubtitle)
	} else {
		subParts := parts
		if len(parts) > 1 {
			subParts = parts[1:]
		}
		setWindowTitle(strings.Join(subParts, " › "))
	}

	var coloredParts []string
	for i, part := range parts {
		if i == len(parts)-1 {
			coloredParts = append(coloredParts, Bold+AccentLavender+part+Reset)
		} else {
			coloredParts = append(coloredParts, TextMuted+part+Reset)
		}
	}
	crumbStr := strings.Join(coloredParts, " "+TextMuted+"›"+Reset+" ")

	innerContentWidth := UIWidth - 6

	titleLine := Bold + AccentTeal + "◆   " + AppTitle + Reset + "  " +
		Bold + AccentLavender + AppVersion + Reset + "   " +
		Bold + AccentTeal + "◆" + Reset
	centeredTitle := padCenter(titleLine, innerContentWidth)

	ruleLine := BorderSubtle + strings.Repeat("─", innerContentWidth) + Reset

	locationLine := TextMuted + "Location:" + Reset + " " + crumbStr
	metaBadge := ""
	if len(breadcrumbPath) < 35 {
		metaBadge = TextMuted + "by" + Reset + " " + AccentTeal + AppAuthor + Reset + " " +
			BorderSubtle + "│" + Reset + " " + TextMuted + AppLicense + Reset
	}

	locWidth := visibleWidth(locationLine)
	metaWidth := visibleWidth(metaBadge)
	spaceBetween := innerContentWidth - locWidth - metaWidth
	if spaceBetween < 1 {
		spaceBetween = 1
	}
	bottomRow := locationLine + strings.Repeat(" ", spaceBetween) + metaBadge

	printPanel([]string{centeredTitle, ruleLine, bottomRow}, "", BorderAccent, 0, 2)
}

func renderSettingsPreviewPanel(cfg *AppConfig) {
	bName := strings.ToUpper(cfg.Get("General", "browser", "chrome"))
	dbgOn := cfg.GetBool("General", "debug_mode", false)
	invis := cfg.Get("Behavior", "resolve_mode", "invisible") == "invisible"
	saveFileOn := cfg.GetBool("Output", "save_to_file", true)
	clipOn := cfg.GetBool("Output", "copy_to_clipboard", false)
	keepOn := cfg.GetBool("Session", "keep_session_across_containers", true)
	skipOn := cfg.GetBool("Behavior", "skip_offline_links", false)

	dbgStr := TextMuted + "OFF" + Reset
	if dbgOn {
		dbgStr = Bold + AccentAmber + "ON" + Reset
	}
	resStr := Bold + AccentLavender + "SINGLE TAB" + Reset
	if invis {
		resStr = Bold + AccentTeal + "INVISIBLE" + Reset
	}
	saveStr := Bold + AccentAmber + "OFF" + Reset
	if saveFileOn {
		saveStr = Bold + AccentGreen + "TXT" + Reset
	}
	clipStr := TextMuted + "OFF" + Reset
	if clipOn {
		clipStr = Bold + AccentGreen + "ON" + Reset
	}
	outModeStr := saveStr + TextMuted + " / Clip: " + Reset + clipStr
	sessStr := Bold + AccentAmber + "RESET EACH" + Reset
	if keepOn {
		sessStr = Bold + AccentGreen + "KEEP ALIVE" + Reset
	}
	skipStr := TextMuted + "NO" + Reset
	if skipOn {
		skipStr = Bold + AccentAmber + "YES" + Reset
	}

	innerContentWidth := UIWidth - 6
	colW := innerContentWidth / 3

	row1Col1 := padRight(TextMuted+"Browser      :"+Reset+" "+Bold+AccentLavender+bName+Reset, colW)
	row1Col2 := padRight(TextMuted+"Resolver   :"+Reset+" "+resStr, colW)
	row1Col3 := padRight(TextMuted+"Session Mode :"+Reset+" "+sessStr, innerContentWidth-(colW*2))

	row2Col1 := padRight(TextMuted+"Save / Copy  :"+Reset+" "+outModeStr, colW)
	row2Col2 := padRight(TextMuted+"Debug Logs :"+Reset+" "+dbgStr, colW)
	row2Col3 := padRight(TextMuted+"Skip Offline :"+Reset+" "+skipStr, innerContentWidth-(colW*2))

	title := TextMuted + "Current Settings Preview (Press " + Bold + AccentMauve + "S" + Reset + TextMuted + " to Customize)" + Reset
	printPanel([]string{
		row1Col1 + row1Col2 + row1Col3,
		row2Col1 + row2Col2 + row2Col3,
	}, title, BorderSubtle, 0, 2)
}

func renderContainerInfoPanel(containerTitle string, totalLinks int, totalHosts int, hostName string, hostLinks int) {
	innerContentWidth := UIWidth - 6

	var rightText string
	if hostName != "" {
		rightText = fmt.Sprintf("  %s│%s  %sHost:%s %s%s%s %s(%d links)%s",
			BorderSubtle, Reset,
			TextMuted, Reset,
			Bold+AccentTeal, hostName, Reset,
			TextMuted, hostLinks, Reset)
	} else {
		rightText = fmt.Sprintf("  %s│%s  %sFound:%s %s%d%s %slinks across%s %s%d%s %shosts%s",
			BorderSubtle, Reset,
			TextMuted, Reset,
			Bold+AccentTeal, totalLinks, Reset,
			TextMuted, Reset,
			Bold+AccentTeal, totalHosts, Reset,
			TextMuted, Reset)
	}

	rightW := visibleWidth(rightText)
	maxLeftW := innerContentWidth - rightW
	if maxLeftW < 15 {
		maxLeftW = 15
	}

	leftPrefix := TextMuted + "Container:" + Reset + " "
	maxTitleW := maxLeftW - visibleWidth(leftPrefix)
	truncTitle := containerTitle
	if runewidth.StringWidth(truncTitle) > maxTitleW && maxTitleW > 1 {
		truncTitle = runewidth.Truncate(truncTitle, maxTitleW-1, "…")
	}
	leftText := leftPrefix + Bold + TextMain + truncTitle + Reset

	spaceCount := innerContentWidth - visibleWidth(leftText) - rightW
	if spaceCount < 0 {
		spaceCount = 0
	}

	line := leftText + strings.Repeat(" ", spaceCount) + rightText
	printPanel([]string{line}, "", BorderSubtle, 0, 2)
}

// ==============================================================================
// CONFIGURATION MANAGEMENT & VALIDATOR
// ==============================================================================

type SettingOption struct {
	Value string
	Label string
	Desc  string
}

type SettingMeta struct {
	ID      string
	Section string
	Key     string
	Title   string
	Info    string
	Warning string
	Options []SettingOption
}

var SettingsMetadata = map[string]SettingMeta{
	"1": {
		ID:      "1",
		Section: "General",
		Key:     "browser",
		Title:   "Web Browser",
		Info:    "Select which installed web browser should be launched.",
		Warning: "Make sure the browser you choose is installed on your PC (or set executable_path).",
		Options: []SettingOption{
			{"chrome", "Google Chrome", "Recommended"},
			{"msedge", "Microsoft Edge", "Pre-installed on Windows"},
			{"brave", "Brave Browser", "Auto-detects installation path"},
			{"chromium", "Chromium", "Uses installed or auto-detected Chromium"},
		},
	},
	"2": {
		ID:      "2",
		Section: "General",
		Key:     "debug_mode",
		Title:   "Debug Logging Mode",
		Info:    "Controls how much technical detail is printed in the console.",
		Warning: "Keep OFF for a clean progress bar UI. Turn ON only if links fail and you need to debug.",
		Options: []SettingOption{
			{"false", "OFF", "Clean App UI with Progress Bar (Recommended)"},
			{"true", "ON", "Verbose Debug Logs (Shows every redirect hop, HTTP header & ad block)"},
		},
	},
	"3": {
		ID:      "3",
		Section: "Behavior",
		Key:     "resolve_mode",
		Title:   "Link Resolution Method",
		Info:    "Controls how Filecrypt '/Link/' -> '/Go/' redirects are followed to get final URLs.",
		Warning: "'Invisible' resolves in memory with ZERO screen flashing. 'Single Tab' uses one background tab.",
		Options: []SettingOption{
			{"invisible", "INVISIBLE", "Zero-Flash In-Memory Resolver (Recommended - Fastest & Easy on Eyes)"},
			{"single_tab", "SINGLE TAB", "Uses One Persistent Dark Background Tab (Fallback if Invisible is blocked)"},
		},
	},
	"4": {
		ID:      "4",
		Section: "Output",
		Key:     "save_to_file",
		Title:   "Save Decrypted URLs to .txt File",
		Info:    "Controls whether decrypted URLs are saved as .txt files on your disk.",
		Warning: "Turn OFF if you only want to copy/paste links from the console or clipboard without leaving files on disk.",
		Options: []SettingOption{
			{"true", "YES", "Save decrypted URLs to .txt files in your Save Directory (Recommended)"},
			{"false", "NO", "Do NOT save files to disk (Show in console & use [C] / Auto-Copy only)"},
		},
	},
	"5": {
		ID:      "5",
		Section: "Output",
		Key:     "copy_to_clipboard",
		Title:   "Auto-Copy Decrypted URLs to Clipboard",
		Info:    "Automatically copies decrypted URLs to your Windows Clipboard when decryption finishes.",
		Warning: "If YES, whatever text you currently have copied in your clipboard will be overwritten.",
		Options: []SettingOption{
			{"false", "NO", "Do NOT touch my clipboard automatically (You can still press [C] manually)"},
			{"true", "YES", "Copy decrypted URLs to Windows Clipboard automatically"},
		},
	},
	"6": {
		ID:      "6",
		Section: "Output",
		Key:     "include_container_title_in_filename",
		Title:   "Include Container Title in Saved Filename",
		Info:    "Uses the container's title in the .txt filename (e.g. 'Show_Name_1080p_mega_nz.txt').",
		Warning: "If NO, files save as 'decrypted_mega_nz.txt' and WILL OVERWRITE previous containers from the same host!",
		Options: []SettingOption{
			{"true", "YES", "Save as '<Container_Title>_<Host>.txt' (Recommended - Prevents Overwriting!)"},
			{"false", "NO", "Save as 'decrypted_<Host>.txt' (Warning: Overwrites previous file for same host!)"},
		},
	},
	"7": {
		ID:      "7",
		Section: "Session",
		Key:     "keep_session_across_containers",
		Title:   "Session Persistence Across Containers ('N')",
		Info:    "Filecrypt uses a temporary session cookie (PHPSESSID) that only lives while the browser stays open.",
		Warning: "If KEEP ALIVE, pressing 'N' keeps the browser open so solved Captchas stay remembered! (Closing browser resets Captchas).",
		Options: []SettingOption{
			{"true", "KEEP ALIVE", "Keep Browser Open & Remember Solved Captchas During Session (Recommended)"},
			{"false", "RESET EACH", "Clear Cookies & Start Fresh on Every New Container (Forces New Captcha)"},
		},
	},
	"8": {
		ID:      "8",
		Section: "Behavior",
		Key:     "skip_offline_links",
		Title:   "Skip Offline / Dead Links",
		Info:    "Controls whether rows marked with a Red Offline icon on Filecrypt are skipped.",
		Warning: "Sometimes Filecrypt falsely marks working links as offline. Keep NO if you want every link.",
		Options: []SettingOption{
			{"false", "NO", "Decrypt All Links for Selected Host (Recommended - Even if marked red/offline)"},
			{"true", "YES", "Automatically Skip Links Marked Offline (Red Status Icon)"},
		},
	},
}

var DefaultConfigMap = map[string]map[string]string{
	"General": {
		"browser":         "chrome",
		"executable_path": "",
		"debug_mode":      "false",
	},
	"Output": {
		"save_to_file":                        "true",
		"save_directory":                      "./decrypted_links",
		"copy_to_clipboard":                   "false",
		"include_container_title_in_filename": "true",
	},
	"Session": {
		"keep_session_across_containers": "true",
		"profile_directory":              "./chrome_filecrypt_profile",
	},
	"Behavior": {
		"resolve_mode":       "invisible",
		"skip_offline_links": "false",
		"request_delay":      "0.2",
	},
}

type AppConfig struct {
	mu   sync.RWMutex
	data map[string]map[string]string
}

func NewAppConfig() *AppConfig {
	c := &AppConfig{data: make(map[string]map[string]string)}
	for sec, kv := range DefaultConfigMap {
		c.data[sec] = make(map[string]string)
		for k, v := range kv {
			c.data[sec][k] = v
		}
	}
	return c
}

func (c *AppConfig) HasSection(sec string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.data[sec]
	return ok
}

func (c *AppConfig) HasOption(sec, key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if m, ok := c.data[sec]; ok {
		_, ok2 := m[key]
		return ok2
	}
	return false
}

func (c *AppConfig) Get(sec, key, fallback string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if m, ok := c.data[sec]; ok {
		if v, ok2 := m[key]; ok2 {
			return v
		}
	}
	return fallback
}

func (c *AppConfig) GetBool(sec, key string, fallback bool) bool {
	val := strings.ToLower(strings.TrimSpace(c.Get(sec, key, "")))
	switch val {
	case "true", "yes", "1", "on":
		return true
	case "false", "no", "0", "off":
		return false
	default:
		return fallback
	}
}

func (c *AppConfig) GetFloat(sec, key string, fallback float64) float64 {
	val := strings.TrimSpace(c.Get(sec, key, ""))
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return f
}

func (c *AppConfig) Set(sec, key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.data[sec]; !ok {
		c.data[sec] = make(map[string]string)
	}
	c.data[sec][key] = val
}

func writeCleanConfigFile(cfg *AppConfig) {
	content := fmt.Sprintf(`# ====================================================================
# FILECRYPT DECRYPTOR %s - CONFIGURATION FILE
# Author     : %s
# Repository : %s
# License    : %s
# Tip: You can edit these settings safely inside the app by pressing 'S'!
# ====================================================================

[General]
# Allowed: chrome, msedge, brave, chromium  (Recommended: chrome)
browser = %s

# Optional full path to custom browser .exe (Leave empty for default installation)
executable_path = %s

# Allowed: true, false  (Recommended: false for clean UI)
debug_mode = %s

[Output]
# Allowed: true, false  (Set to false if you only want to copy/paste links without saving .txt files to disk)
save_to_file = %s

# Folder where decrypted .txt files are saved (when save_to_file = true)
save_directory = %s

# Allowed: true, false  (Recommended: false so your clipboard isn't overwritten unexpectedly)
copy_to_clipboard = %s

# Allowed: true, false  (Recommended: true! If false, 'decrypted_mega_nz.txt' gets overwritten by new containers!)
include_container_title_in_filename = %s

[Session]
# Allowed: true, false  (Recommended: true - Keeps browser open when pressing 'N' so solved Captchas are remembered!)
# Note: Closing the browser window always ends Filecrypt's temporary PHPSESSID session cookie.
keep_session_across_containers = %s

# Folder where browser profile is stored
profile_directory = %s

[Behavior]
# Allowed: invisible, single_tab  (Recommended: invisible for ZERO tab flashing!)
resolve_mode = %s

# Allowed: true, false  (Recommended: false in case Filecrypt falsely reports a link offline)
skip_offline_links = %s

# Delay in seconds between links (Allowed: 0.0 to 10.0, Recommended: 0.2)
request_delay = %s
`,
		AppVersion,
		AppAuthor,
		AppRepoURL,
		AppLicense,
		cfg.Get("General", "browser", "chrome"),
		cfg.Get("General", "executable_path", ""),
		cfg.Get("General", "debug_mode", "false"),
		cfg.Get("Output", "save_to_file", "true"),
		cfg.Get("Output", "save_directory", "./decrypted_links"),
		cfg.Get("Output", "copy_to_clipboard", "false"),
		cfg.Get("Output", "include_container_title_in_filename", "true"),
		cfg.Get("Session", "keep_session_across_containers", "true"),
		cfg.Get("Session", "profile_directory", "./chrome_filecrypt_profile"),
		cfg.Get("Behavior", "resolve_mode", "invisible"),
		cfg.Get("Behavior", "skip_offline_links", "false"),
		cfg.Get("Behavior", "request_delay", "0.2"),
	)
	_ = os.WriteFile(ConfigPath, []byte(content), 0644)
}

func resetConfigToDefaults() *AppConfig {
	cfg := NewAppConfig()
	writeCleanConfigFile(cfg)
	return cfg
}

func parseRawINI(path string) (*AppConfig, error) {
	rawBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &AppConfig{data: make(map[string]map[string]string)}
	lines := strings.Split(string(rawBytes), "\n")
	currentSec := ""
	for idx, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("line %d: malformed section header '%s'", idx+1, line)
			}
			currentSec = strings.TrimSpace(line[1 : len(line)-1])
			if _, ok := cfg.data[currentSec]; !ok {
				cfg.data[currentSec] = make(map[string]string)
			}
			continue
		}
		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 || currentSec == "" {
			return nil, fmt.Errorf("line %d: invalid syntax '%s'", idx+1, line)
		}
		k := strings.TrimSpace(line[:eqIdx])
		v := strings.TrimSpace(line[eqIdx+1:])
		cfg.data[currentSec][k] = v
	}
	return cfg, nil
}

func validateAndLoadConfig() *AppConfig {
	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		return resetConfigToDefaults()
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		var errs []string
		cfg, parseErr := parseRawINI(ConfigPath)
		if parseErr != nil {
			errs = append(errs, fmt.Sprintf("File syntax corrupted: %v", parseErr))
		}

		if len(errs) == 0 {
			migrated := false
			if cfg.HasSection("Session") && !cfg.HasOption("Session", "keep_session_across_containers") {
				cfg.Set("Session", "keep_session_across_containers", "true")
				migrated = true
			}
			if cfg.HasSection("Output") && !cfg.HasOption("Output", "save_to_file") {
				cfg.Set("Output", "save_to_file", "true")
				migrated = true
			}
			if migrated {
				writeCleanConfigFile(cfg)
			}

			sectionsOrder := []string{"General", "Output", "Session", "Behavior"}
			for _, sec := range sectionsOrder {
				keys := DefaultConfigMap[sec]
				if !cfg.HasSection(sec) {
					errs = append(errs, fmt.Sprintf("Missing section: [%s]", sec))
					continue
				}
				for k := range keys {
					if !cfg.HasOption(sec, k) {
						errs = append(errs, fmt.Sprintf("Missing setting: [%s] %s", sec, k))
					}
				}
			}

			for _, id := range []string{"1", "2", "3", "4", "5", "6", "7", "8"} {
				meta := SettingsMetadata[id]
				sec, k := meta.Section, meta.Key
				if cfg.HasOption(sec, k) {
					val := strings.ToLower(strings.TrimSpace(cfg.Get(sec, k, "")))
					var allowed []string
					isValid := false
					for _, opt := range meta.Options {
						allowed = append(allowed, opt.Value)
						if val == opt.Value {
							isValid = true
						}
					}
					if !isValid {
						errs = append(errs, fmt.Sprintf("[%s] %s = '%s' is invalid! (Allowed: %s)", sec, k, val, strings.Join(allowed, ", ")))
					} else {
						cfg.Set(sec, k, val)
					}
				}
			}

			if cfg.HasOption("Behavior", "request_delay") {
				rawDelay := strings.TrimSpace(cfg.Get("Behavior", "request_delay", ""))
				dVal, err := strconv.ParseFloat(rawDelay, 64)
				if err != nil {
					errs = append(errs, fmt.Sprintf("[Behavior] request_delay = '%s' is not a valid number (Recommended: 0.2)", rawDelay))
				} else if dVal < 0 || dVal > 30 {
					errs = append(errs, fmt.Sprintf("[Behavior] request_delay = '%s' must be between 0.0 and 30.0", rawDelay))
				}
			}

			if cfg.HasOption("General", "executable_path") {
				exeP := strings.TrimSpace(cfg.Get("General", "executable_path", ""))
				if exeP != "" {
					if _, err := os.Stat(exeP); os.IsNotExist(err) {
						errs = append(errs, fmt.Sprintf("[General] executable_path = '%s' does not exist on this PC!", exeP))
					}
				}
			}
		}

		if len(errs) == 0 {
			return cfg
		}

		clearScreen()
		setWindowTitle("Config Validator Warning")
		var lines []string
		lines = append(lines, Bold+AccentAmber+"Your config.ini has corrupted or invalid settings:"+Reset)
		lines = append(lines, "")
		for _, e := range errs {
			lines = append(lines, "  "+AccentCoral+"•"+Reset+" "+TextMain+e+Reset)
		}
		lines = append(lines, "")
		lines = append(lines, "  "+Bold+AccentTeal+"[R]"+Reset+" "+TextMain+"Reset to Recommended Default Settings "+AccentGreen+"(Recommended)"+Reset)
		lines = append(lines, "  "+Bold+AccentTeal+"[F]"+Reset+" "+TextMain+"Fix Only Invalid Values & Keep My Other Settings"+Reset)
		lines = append(lines, "  "+Bold+AccentTeal+"[E]"+Reset+" "+TextMain+"Open config.ini in Notepad to Fix Manually"+Reset)

		printPanel(lines, Bold+AccentCoral+"Config Validator Warning"+Reset, AccentCoral, 1, 2)

		fmt.Printf("\n  %sChoose an option [R / F / E] (default: R): %s", AccentTeal, Reset)
		inp, _ := reader.ReadString('\n')
		choice := strings.ToUpper(strings.TrimSpace(inp))

		if choice == "E" {
			if runtime.GOOS == "windows" {
				_ = exec.Command("notepad.exe", ConfigPath).Start()
			}
			fmt.Printf("  %sPress Enter after saving config.ini in Notepad...%s", AccentAmber, Reset)
			_, _ = reader.ReadString('\n')
			continue
		} else if choice == "F" && cfg != nil {
			fixedCfg := NewAppConfig()
			for sec, keys := range DefaultConfigMap {
				for k, defaultVal := range keys {
					if cfg.HasOption(sec, k) {
						fixedCfg.Set(sec, k, strings.TrimSpace(cfg.Get(sec, k, defaultVal)))
					} else {
						fixedCfg.Set(sec, k, defaultVal)
					}
				}
			}
			for _, meta := range SettingsMetadata {
				sec, k := meta.Section, meta.Key
				val := strings.ToLower(fixedCfg.Get(sec, k, ""))
				valid := false
				for _, opt := range meta.Options {
					if val == opt.Value {
						valid = true
						break
					}
				}
				if !valid {
					fixedCfg.Set(sec, k, DefaultConfigMap[sec][k])
				}
			}
			dVal, err := strconv.ParseFloat(fixedCfg.Get("Behavior", "request_delay", "0.2"), 64)
			if err != nil || dVal < 0 || dVal > 30 {
				fixedCfg.Set("Behavior", "request_delay", DefaultConfigMap["Behavior"]["request_delay"])
			}
			exeP := fixedCfg.Get("General", "executable_path", "")
			if exeP != "" {
				if _, err := os.Stat(exeP); os.IsNotExist(err) {
					fixedCfg.Set("General", "executable_path", "")
				}
			}
			writeCleanConfigFile(fixedCfg)
			return fixedCfg
		} else {
			return resetConfigToDefaults()
		}
	}
}

func isDebug(cfg *AppConfig) bool {
	if cfg == nil {
		return false
	}
	return cfg.GetBool("General", "debug_mode", false)
}

func logMsg(msg string, level string, cfg *AppConfig) {
	if level == "DEBUG" && (cfg == nil || !isDebug(cfg)) {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	col := TextMain
	switch level {
	case "INFO":
		col = AccentTeal
	case "SUCCESS":
		col = AccentGreen
	case "WARN":
		col = AccentAmber
	case "ERROR":
		col = AccentCoral
	case "DEBUG":
		col = TextMuted
	}
	fmt.Printf("  %s%s%s  %s%s%-7s%s  %s%s%s\n",
		TextMuted, timestamp, Reset,
		Bold, col, level, Reset,
		TextMain, msg, Reset)
}

// ==============================================================================
// SYSTEM HELPERS & KEYBOARD INPUT READERS (WITH INSTANT PASTE DRAIN)
// ==============================================================================

func copyTextToClipboard(text string) bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("clip")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return false
		}
		if err := cmd.Start(); err != nil {
			return false
		}
		_, _ = io.WriteString(stdin, text)
		_ = stdin.Close()
		return cmd.Wait() == nil
	}
	return false
}

func openFolderInExplorer(folderPath string) {
	_ = os.MkdirAll(folderPath, 0755)
	if runtime.GOOS == "windows" {
		_ = exec.Command("explorer", folderPath).Start()
	}
}

func openURLInBrowser(targetURL string) {
	if runtime.GOOS == "windows" {
		_ = exec.Command("explorer", targetURL).Start()
	}
}

func readTextWithEsc(prompt string) (string, bool) {
	fmt.Print(prompt)

	if runtime.GOOS != "windows" {
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line), true
	}

	drainKeyboardBuffer()

	var chars []rune
	for {
		for kbhit() {
			ch := getwch()

			if ch == 0x03 {
				fmt.Printf("\n  %sExited cleanly.%s\n", AccentAmber, Reset)
				os.Exit(0)
			}

			if ch == 0x1b {
				fmt.Printf(" %s(Cancelled)%s\n", AccentAmber, Reset)
				return "", false
			}

			if ch == 0x00 || ch == 0xe0 {
				if kbhit() {
					getwch()
				}
				continue
			}

			if ch == '\r' || ch == '\n' {
				fmt.Print("\n")
				return strings.TrimSpace(string(chars)), true
			}

			if ch == 0x08 || ch == 0x7f {
				if len(chars) > 0 {
					chars = chars[:len(chars)-1]
					fmt.Print("\b \b")
				}
				continue
			}

			if unicode.IsPrint(ch) {
				chars = append(chars, ch)
				fmt.Print(string(ch))
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
}

// readTextWithEscAndAbort reads a line of text while supporting Esc cancellation and
// automatically aborting if the browser tab is closed while waiting for input.
func readTextWithEscAndAbort(prompt string, abortCheckFn func() bool) (string, bool) {
	fmt.Print(prompt)

	if runtime.GOOS != "windows" {
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line), true
	}

	drainKeyboardBuffer()

	var chars []rune
	for {
		if abortCheckFn != nil && abortCheckFn() {
			fmt.Print("\n")
			return "", false
		}
		for kbhit() {
			ch := getwch()

			if ch == 0x03 {
				fmt.Printf("\n  %sExited cleanly.%s\n", AccentAmber, Reset)
				os.Exit(0)
			}

			if ch == 0x1b {
				fmt.Printf(" %s(Cancelled)%s\n", AccentAmber, Reset)
				return "", false
			}

			if ch == 0x00 || ch == 0xe0 {
				if kbhit() {
					getwch()
				}
				continue
			}

			if ch == '\r' || ch == '\n' {
				fmt.Print("\n")
				return strings.TrimSpace(string(chars)), true
			}

			if ch == 0x08 || ch == 0x7f {
				if len(chars) > 0 {
					chars = chars[:len(chars)-1]
					fmt.Print("\b \b")
				}
				continue
			}

			if unicode.IsPrint(ch) {
				chars = append(chars, ch)
				fmt.Print(string(ch))
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// parseLinkSelectionInput parses inputs like "1", "1, 3, 5", "2-6", "1, 3-5, 8", or "A" (all)
// into a deduplicated 1-based list of link numbers within [1..maxNum].
func parseLinkSelectionInput(input string, maxNum int) ([]int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, errors.New("empty selection")
	}
	if strings.EqualFold(trimmed, "A") || strings.EqualFold(trimmed, "ALL") {
		res := make([]int, maxNum)
		for i := 1; i <= maxNum; i++ {
			res[i-1] = i
		}
		return res, nil
	}

	normalized := strings.ReplaceAll(trimmed, ";", ",")
	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})

	seen := make(map[int]bool)
	var result []int

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if strings.Contains(tok, "-") {
			parts := strings.SplitN(tok, "-", 2)
			start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 != nil || err2 != nil || start < 1 || end < 1 || start > maxNum || end > maxNum {
				return nil, fmt.Errorf("invalid range '%s' (valid: 1-%d)", tok, maxNum)
			}
			if start > end {
				start, end = end, start
			}
			for n := start; n <= end; n++ {
				if !seen[n] {
					seen[n] = true
					result = append(result, n)
				}
			}
		} else {
			num, err := strconv.Atoi(tok)
			if err != nil || num < 1 || num > maxNum {
				return nil, fmt.Errorf("invalid link number '%s' (valid: 1-%d)", tok, maxNum)
			}
			if !seen[num] {
				seen[num] = true
				result = append(result, num)
			}
		}
	}

	if len(result) == 0 {
		return nil, errors.New("no valid link numbers selected")
	}
	return result, nil
}

func waitForSingleKey(prompt string, validLetters []string, maxNum int, abortCheckFn func() bool, allowEsc bool) string {
	fmt.Print(prompt)

	if runtime.GOOS != "windows" {
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		return strings.ToUpper(strings.TrimSpace(line))
	}

	drainKeyboardBuffer()

	validSet := make(map[string]bool)
	for _, l := range validLetters {
		validSet[strings.ToUpper(l)] = true
	}

	buffer := ""
	for {
		if abortCheckFn != nil && abortCheckFn() {
			fmt.Print("\n")
			return ""
		}

		if kbhit() {
			ch := getwch()

			if ch == 0x03 {
				fmt.Printf("\n  %sExited cleanly.%s\n", AccentAmber, Reset)
				os.Exit(0)
			}

			if ch == 0x1b && allowEsc {
				fmt.Printf("%sEsc%s\n", AccentAmber, Reset)
				return "ESC"
			}

			if ch == 0x00 || ch == 0xe0 {
				if kbhit() {
					getwch()
				}
				continue
			}

			if ch == 0x08 || ch == 0x7f {
				if len(buffer) > 0 {
					buffer = buffer[:len(buffer)-1]
					fmt.Print("\b \b")
				}
				continue
			}

			if ch == '\r' || ch == '\n' {
				if buffer != "" {
					fmt.Print("\n")
					return buffer
				}
				continue
			}

			chUp := strings.ToUpper(string(ch))
			if buffer == "" && validSet[chUp] {
				fmt.Printf("%s%s%s\n", Bold+AccentTeal, chUp, Reset)
				return chUp
			}

			if maxNum > 0 && unicode.IsDigit(ch) {
				buffer += string(ch)
				fmt.Print(string(ch))

				valTimes10, _ := strconv.Atoi(buffer + "0")
				if valTimes10 > maxNum {
					fmt.Print("\n")
					return buffer
				}

				moreTyped := false
				for i := 0; i < 9; i++ {
					time.Sleep(50 * time.Millisecond)
					if kbhit() {
						moreTyped = true
						break
					}
				}
				if !moreTyped {
					fmt.Print("\n")
					return buffer
				}
			}
		}

		time.Sleep(40 * time.Millisecond)
	}
}

// ==============================================================================
// INTERACTIVE MENUS: ABOUT/LICENSE & SETTINGS
// ==============================================================================

func showAboutLicenseScreen(cfg *AppConfig) {
	for {
		printAppHeader("App / About & License", cfg, true, "")

		aboutLines := []string{
			Bold + AccentTeal + AppTitle + " " + AppVersion + Reset,
			"",
			TextMuted + "Author     :" + Reset + "  " + Bold + TextMain + AppAuthor + Reset + "  " + TextMuted + "(" + AppGithubProfile + ")" + Reset,
			TextMuted + "Repository :" + Reset + "  " + Bold + AccentLavender + AppRepoURL + Reset,
			TextMuted + "License    :" + Reset + "  " + Bold + AccentGreen + "GNU General Public License v3.0 (GPL-3.0)" + Reset,
			TextMuted + "Copyright  :" + Reset + "  " + TextMain + "Copyright © " + AppYear + " " + AppAuthor + Reset,
			"",
			TextMuted + "This program is free software: you can redistribute it and/or modify it under the terms of" + Reset,
			TextMuted + "the GNU General Public License as published by the Free Software Foundation, either version 3" + Reset,
			TextMuted + "of the License, or (at your option) any later version." + Reset,
			"",
			TextMuted + "This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;" + Reset,
			TextMuted + "without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE." + Reset,
			TextMuted + "See the GNU General Public License for more details:" + Reset,
			TextMuted + AppLicenseURL + Reset,
			"",
			"  " + Bold + AccentTeal + "[G]" + Reset + " Open GitHub Repository    " +
				Bold + AccentMauve + "[L]" + Reset + " View Repository License    " +
				Bold + AccentGreen + "[Esc]" + Reset + " Back to Home",
		}

		printPanel(aboutLines, "", BorderSubtle, 1, 2)

		key := waitForSingleKey(
			fmt.Sprintf("\n  %sPress [G] GitHub Repo, [L] License, or [Esc] to return: %s", AccentTeal, Reset),
			[]string{"G", "L", "B", "Q", "I"},
			0, nil, true,
		)
		if key == "G" {
			openURLInBrowser(AppRepoURL)
			continue
		} else if key == "L" {
			openURLInBrowser(AppLicenseURL)
			continue
		}
		break
	}
}

func chooseSettingOptionSubmenu(cfg *AppConfig, settingID string, parentPath string) string {
	meta := SettingsMetadata[settingID]
	currentVal := strings.ToLower(strings.TrimSpace(cfg.Get(meta.Section, meta.Key, "")))

	printAppHeader(fmt.Sprintf("%s / %s", parentPath, meta.Title), cfg, true, "")

	infoLines := []string{
		Bold + TextMain + meta.Title + Reset,
		TextMuted + meta.Info + Reset,
		AccentAmber + "Note: " + meta.Warning + Reset,
	}
	printPanel(infoLines, "", BorderSubtle, 0, 2)

	cols := []tableCol{
		{Header: "Key", Width: 6, Justify: "center", NoWrap: true},
		{Header: "Option", Width: 24, Justify: "left", NoWrap: true},
		{Header: "Description", Width: 49, Justify: "left", NoWrap: false},
		{Header: "Status", Width: 14, Justify: "right", NoWrap: true},
	}

	var rows []tableRow
	for idx, opt := range meta.Options {
		isCur := opt.Value == currentVal
		keyCell := fmt.Sprintf("%s[%d]%s", Bold+AccentTeal, idx+1, Reset)
		labelCell := TextMain + opt.Label + Reset
		if isCur {
			labelCell = Bold + TextMain + opt.Label + Reset
		}
		descCell := TextMuted + opt.Desc + Reset
		statusCell := ""
		if isCur {
			statusCell = Bold + AccentGreen + "● CURRENT" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{keyCell, labelCell, descCell, statusCell}})
	}

	rows = append(rows, tableRow{IsSection: true})
	rows = append(rows, tableRow{Cells: []string{
		Bold + AccentAmber + "[Esc]" + Reset,
		TextMuted + "Cancel" + Reset,
		TextMuted + "Return to Settings without changing" + Reset,
		"",
	}})

	tblLines := renderTable(cols, rows, true, UIWidth-4)
	printPanel(tblLines, "", BorderSubtle, 0, 1)

	choice := waitForSingleKey(
		fmt.Sprintf("\n  %sSelect option [1-%d, Esc]: %s", AccentTeal, len(meta.Options), Reset),
		[]string{"B", "Q"},
		len(meta.Options), nil, true,
	)

	if idx, err := strconv.Atoi(choice); err == nil && idx >= 1 && idx <= len(meta.Options) {
		selected := meta.Options[idx-1]
		cfg.Set(meta.Section, meta.Key, selected.Value)
		writeCleanConfigFile(cfg)
		return fmt.Sprintf("%s✓ Updated '%s' to%s %s%s%s",
			AccentGreen, meta.Title, Reset, Bold+TextMain, selected.Label, Reset)
	}
	return ""
}

func interactiveSettingsMenu(cfg *AppConfig, parentPath string) {
	statusBanner := ""
	settingsPath := fmt.Sprintf("%s / Settings", parentPath)

	for {
		printAppHeader(settingsPath, cfg, true, "")

		if statusBanner != "" {
			printPanel([]string{"  " + statusBanner}, "", AccentGreen, 0, 1)
		}

		cols := []tableCol{
			{Header: "Key", Width: 6, Justify: "center", NoWrap: true},
			{Header: "Setting", Width: 33, Justify: "left", NoWrap: true},
			{Header: "Current Value", Width: 17, Justify: "left", NoWrap: true},
			{Header: "Notes / Recommendation", Width: 0, Justify: "left", NoWrap: true},
		}

		var rows []tableRow

		// 1. Browser
		bVal := strings.ToUpper(cfg.Get("General", "browser", "chrome"))
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[1]" + Reset,
			TextMain + "Web Browser" + Reset,
			Bold + AccentLavender + bVal + Reset,
			TextMuted + "Chrome, Edge, Brave, Chromium" + Reset,
		}})

		// 2. Debug
		dbgOn := cfg.GetBool("General", "debug_mode", false)
		dbgVal := Bold + TextMain + "OFF" + Reset
		dbgNote := TextMuted + "Clean UI (Recommended)" + Reset
		if dbgOn {
			dbgVal = Bold + AccentAmber + "ON" + Reset
			dbgNote = AccentAmber + "Verbose logs active" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[2]" + Reset,
			TextMain + "Debug Logging Mode" + Reset,
			dbgVal,
			dbgNote,
		}})

		// 3. Resolve Mode
		invis := cfg.Get("Behavior", "resolve_mode", "invisible") == "invisible"
		resVal := Bold + AccentLavender + "SINGLE TAB" + Reset
		resNote := TextMuted + "One Dark Background Tab" + Reset
		if invis {
			resVal = Bold + AccentTeal + "INVISIBLE" + Reset
			resNote = TextMuted + "Zero-Flash In-Memory (Recommended)" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[3]" + Reset,
			TextMain + "Link Resolution Method" + Reset,
			resVal,
			resNote,
		}})

		// 4. Save to .txt File
		saveFileOn := cfg.GetBool("Output", "save_to_file", true)
		saveFileVal := Bold + AccentAmber + "NO" + Reset
		saveFileNote := AccentAmber + "Console & Clipboard only (No files on disk)" + Reset
		if saveFileOn {
			saveFileVal = Bold + AccentGreen + "YES" + Reset
			saveFileNote = TextMuted + "Save .txt files to disk (Recommended)" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[4]" + Reset,
			TextMain + "Save URLs to .txt File" + Reset,
			saveFileVal,
			saveFileNote,
		}})

		// 5. Clipboard
		clipOn := cfg.GetBool("Output", "copy_to_clipboard", false)
		clipVal := Bold + TextMain + "NO" + Reset
		clipNote := TextMuted + "Manual copy via [C] (Recommended)" + Reset
		if clipOn {
			clipVal = Bold + AccentGreen + "YES" + Reset
			clipNote = AccentAmber + "Auto-copies & overwrites Clipboard" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[5]" + Reset,
			TextMain + "Auto-Copy URLs to Clipboard" + Reset,
			clipVal,
			clipNote,
		}})

		// 6. Container Title in Filename
		titleOn := cfg.GetBool("Output", "include_container_title_in_filename", true)
		titleVal := Bold + AccentCoral + "NO" + Reset
		titleNote := AccentCoral + "Warning: Overwrites previous host file!" + Reset
		if titleOn {
			titleVal = Bold + AccentGreen + "YES" + Reset
			titleNote = TextMuted + "'<Title>_<Host>.txt' (Recommended)" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[6]" + Reset,
			TextMain + "Include Container Title in File" + Reset,
			titleVal,
			titleNote,
		}})

		// 7. Session Mode
		keepOn := cfg.GetBool("Session", "keep_session_across_containers", true)
		keepVal := Bold + AccentAmber + "RESET EACH" + Reset
		keepNote := AccentAmber + "Forces new Captcha on every URL" + Reset
		if keepOn {
			keepVal = Bold + AccentGreen + "KEEP ALIVE" + Reset
			keepNote = TextMuted + "Remembers solved Captchas while open" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[7]" + Reset,
			TextMain + "Session Mode on New Container" + Reset,
			keepVal,
			keepNote,
		}})

		// 8. Skip Offline
		skipOn := cfg.GetBool("Behavior", "skip_offline_links", false)
		skipVal := Bold + TextMain + "NO" + Reset
		skipNote := TextMuted + "Decrypts all links (Recommended)" + Reset
		if skipOn {
			skipVal = Bold + AccentAmber + "YES" + Reset
			skipNote = AccentAmber + "Skips links marked Red" + Reset
		}
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[8]" + Reset,
			TextMain + "Skip Offline / Dead Links" + Reset,
			skipVal,
			skipNote,
		}})

		// 9. Save Directory
		saveDir := cfg.Get("Output", "save_directory", "./decrypted_links")
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentTeal + "[9]" + Reset,
			TextMain + "Change Save Directory" + Reset,
			AccentTeal + saveDir + Reset,
			TextMuted + "Output folder for .txt files" + Reset,
		}})

		rows = append(rows, tableRow{IsSection: true})
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentAmber + "[R]" + Reset,
			TextMain + "Reset All Settings" + Reset,
			"",
			TextMuted + "Restore recommended defaults" + Reset,
		}})
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentMauve + "[E]" + Reset,
			TextMain + "Edit config.ini in Notepad" + Reset,
			"",
			TextMuted + "Open raw configuration file" + Reset,
		}})
		rows = append(rows, tableRow{Cells: []string{
			Bold + AccentGreen + "[Esc]" + Reset,
			TextMain + "Back to " + parentPath + Reset,
			"",
			"",
		}})

		tblLines := renderTable(cols, rows, false, UIWidth-4)
		printPanel(tblLines, "", BorderSubtle, 0, 1)

		key := waitForSingleKey(
			fmt.Sprintf("\n  %sSelect a setting [1-9, R, E, Esc]: %s", AccentTeal, Reset),
			[]string{"R", "E", "B", "Q"},
			9, nil, true,
		)

		if key == "ESC" || key == "B" || key == "Q" || key == "" {
			break
		} else if _, ok := SettingsMetadata[key]; ok {
			msg := chooseSettingOptionSubmenu(cfg, key, settingsPath)
			if msg != "" {
				statusBanner = msg
			}
		} else if key == "9" {
			printAppHeader(fmt.Sprintf("%s / Save Directory", settingsPath), cfg, true, "")
			absSaveDir, _ := filepath.Abs(filepath.Join(AppDir, saveDir))
			dirPanel := []string{
				TextMuted + "Current Folder:" + Reset + "  " + Bold + AccentTeal + absSaveDir + Reset,
				TextMuted + "Instructions  :" + Reset + "  Type or paste a new folder path and press " + Bold + AccentGreen + "Enter" + Reset + ", or press " + Bold + AccentAmber + "Esc" + Reset + " to cancel.",
			}
			printPanel(dirPanel, "", BorderSubtle, 1, 2)
			newDir, ok := readTextWithEsc(fmt.Sprintf("\n  %sNew folder path (or Esc to cancel): %s", AccentTeal, Reset))
			if ok && newDir != "" {
				cfg.Set("Output", "save_directory", newDir)
				writeCleanConfigFile(cfg)
				statusBanner = fmt.Sprintf("%s✓ Save directory updated to:%s %s%s%s", AccentGreen, Reset, Bold+TextMain, newDir, Reset)
			} else {
				statusBanner = TextMuted + "Save directory change cancelled." + Reset
			}
		} else if key == "R" {
			printAppHeader(fmt.Sprintf("%s / Reset Defaults", settingsPath), cfg, true, "")
			confirm := waitForSingleKey(
				fmt.Sprintf("\n  %sReset ALL settings to recommended defaults? [Y / N / Esc]: %s", AccentAmber, Reset),
				[]string{"Y", "N"},
				0, nil, true,
			)
			if confirm == "Y" {
				newCfg := resetConfigToDefaults()
				for sec, kv := range newCfg.data {
					for k, v := range kv {
						cfg.Set(sec, k, v)
					}
				}
				statusBanner = AccentGreen + "✓ All settings restored to recommended defaults!" + Reset
			}
		} else if key == "E" {
			writeCleanConfigFile(cfg)
			if runtime.GOOS == "windows" {
				_ = exec.Command("notepad.exe", ConfigPath).Start()
			}
			fmt.Printf("\n  %sPress Enter after saving config.ini in Notepad...%s", AccentAmber, Reset)
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
			validated := validateAndLoadConfig()
			for sec, kv := range validated.data {
				for k, v := range kv {
					cfg.Set(sec, k, v)
				}
			}
			statusBanner = AccentGreen + "✓ Reloaded & validated config.ini from Notepad!" + Reset
		}
	}
}

// ==============================================================================
// BROWSER SCRIPTS & SESSION ENGINE (GO-ROD NATIVE CDP)
// ==============================================================================

const AntiAdInitScript = `
(() => {
    const applyDarkBg = () => {
        if (document.documentElement && (!location.href || location.href === 'about:blank')) {
            document.documentElement.style.backgroundColor = '#0d0f14';
            if (document.body) document.body.style.backgroundColor = '#0d0f14';
        }
    };
    applyDarkBg();

    window.__ALLOW_POPUPS__ = false;
    window.__CAPTURED_LINK_URL__ = null;

    const origOpen = window.open;
    window.open = function(url, target, features) {
        if (window.__CAPTURE_MODE__) {
            window.__CAPTURED_LINK_URL__ = url;
            return null;
        }
        if (!window.__ALLOW_POPUPS__) {
            return null;
        }
        return origOpen.call(window, url, target, features);
    };

    const cleanAds = () => {
        applyDarkBg();
        document.querySelectorAll('#ddownload-rect, [id*="-rect"], [class*="sponsor"]').forEach(el => {
            el.remove();
        });

        document.querySelectorAll('*').forEach(el => {
            if (el.shadowRoot) {
                if (el.shadowRoot.querySelector('.rc-link, .rc-ad, a[href*="/Link/"]')) {
                    el.remove();
                }
            }
        });

        const isCaptchaPage = !!document.querySelector('form#cform, .protection');
        if (isCaptchaPage && !document.querySelector('table tbody tr td.status')) {
            document.querySelectorAll('a[href*="/Link/"], a[target="_blank"]').forEach(a => {
                if (!a.href.includes('cutcaptcha')) {
                    a.remove();
                }
            });
        }
    };

    document.addEventListener('DOMContentLoaded', () => {
        cleanAds();
        const obs = new MutationObserver(cleanAds);
        obs.observe(document.documentElement, { childList: true, subtree: true });
    });
    setInterval(cleanAds, 200);
})();
`

type RowInfo struct {
	Index            int    `json:"index"`
	RowID            string `json:"rowid"`
	Filename         string `json:"filename"`
	Host             string `json:"host"`
	Size             string `json:"size"`
	IsOnline         bool   `json:"is_online"`
	HasValidRealFile bool   `json:"has_valid_real_file"`
	DirectLink       string `json:"direct_link"`
}

type ContainerInfo struct {
	Title string    `json:"title"`
	Rows  []RowInfo `json:"rows"`
}

type SessionHolder struct {
	mu              sync.Mutex
	Launcher        *launcher.Launcher
	ProfileDir      string
	Browser         *rod.Browser
	KeepalivePage   *rod.Page
	BrowserName     string
	Phase           string
	MainContainerID proto.TargetTargetID
	CaptureLinkCh   chan string
	UserAgent       string
}

func (s *SessionHolder) SetPhase(phase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Phase = phase
}

func (s *SessionHolder) GetPhase() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Phase
}

func (s *SessionHolder) SetCaptureChan(ch chan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CaptureLinkCh = ch
}

func (s *SessionHolder) EmitCapturedLink(u string) {
	s.mu.Lock()
	ch := s.CaptureLinkCh
	s.mu.Unlock()
	if ch != nil {
		select {
		case ch <- u:
		default:
		}
	}
}

func safeClosePage(p *rod.Page) {
	if p == nil {
		return
	}
	defer func() { _ = recover() }()
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	_ = p.Context(ctx).Close()
}

func isBrowserProcessName(exeName string) bool {
	name := strings.ToLower(strings.TrimSpace(exeName))
	return name == "chrome.exe" || name == "msedge.exe" || name == "brave.exe" || name == "chromium.exe"
}

// killProcessTreeNative terminates a root browser PID and all of its child/descendant
// browser processes using pure Go standard library syscalls (zero taskkill.exe or
// powershell.exe spawned) while verifying process executable names to prevent PID-reuse accidents.
func killProcessTreeNative(rootPID int) {
	if runtime.GOOS != "windows" || rootPID <= 0 {
		return
	}
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil || snapshot == syscall.InvalidHandle {
		return
	}
	defer syscall.CloseHandle(snapshot)

	childrenMap := make(map[uint32][]uint32)
	procNames := make(map[uint32]string)

	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := syscall.Process32First(snapshot, &pe); err == nil {
		for {
			exeName := syscall.UTF16ToString(pe.ExeFile[:])
			procNames[pe.ProcessID] = exeName
			childrenMap[pe.ParentProcessID] = append(childrenMap[pe.ParentProcessID], pe.ProcessID)
			if err := syscall.Process32Next(snapshot, &pe); err != nil {
				break
			}
		}
	}

	// If rootPID is still alive in the process table, verify it is actually a browser executable
	// so we never terminate an unrelated system or user process due to Windows PID reuse.
	if rootName, exists := procNames[uint32(rootPID)]; exists {
		if !isBrowserProcessName(rootName) {
			return
		}
	}

	var toKill []uint32
	queue := []uint32{uint32(rootPID)}
	visited := make(map[uint32]bool)
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if visited[curr] {
			continue
		}
		visited[curr] = true
		if name, ok := procNames[curr]; ok && isBrowserProcessName(name) {
			toKill = append(toKill, curr)
		}
		for _, child := range childrenMap[curr] {
			if !visited[child] {
				queue = append(queue, child)
			}
		}
	}

	for i := len(toKill) - 1; i >= 0; i-- {
		if p, err := os.FindProcess(int(toKill[i])); err == nil {
			_ = p.Kill()
		}
	}
}

func cleanStaleProfileLocks(profileDir string) {
	if profileDir == "" {
		return
	}
	pidFile := filepath.Join(profileDir, ".browser.pid")
	if rawPID, err := os.ReadFile(pidFile); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(rawPID))); err == nil && pid > 0 {
			killProcessTreeNative(pid)
		}
		_ = os.Remove(pidFile)
	}
	for _, lockName := range []string{"lockfile", "SingletonLock", "SingletonCookie", "SingletonSocket"} {
		_ = os.Remove(filepath.Join(profileDir, lockName))
	}
}

func safeCloseBrowser(s *SessionHolder) {
	if s == nil {
		return
	}
	s.mu.Lock()
	b := s.Browser
	l := s.Launcher
	profDir := s.ProfileDir
	s.Browser = nil
	s.Launcher = nil
	s.KeepalivePage = nil
	s.mu.Unlock()

	if b != nil {
		func() {
			defer func() { _ = recover() }()
			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
			defer cancel()
			_ = proto.BrowserClose{}.Call(b.Context(ctx))
			_ = b.Context(ctx).Close()
		}()
	}

	if l != nil {
		if pid := l.PID(); pid > 0 {
			killProcessTreeNative(pid)
		}
		func() {
			defer func() { _ = recover() }()
			l.Kill()
		}()
	}

	time.Sleep(150 * time.Millisecond)
	cleanStaleProfileLocks(profDir)
}

func isBrowserAlive(s *SessionHolder) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	b := s.Browser
	s.mu.Unlock()
	if b == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	pages, err := b.Context(ctx).Pages()
	return err == nil && len(pages) > 0
}

func isPageClosed(b *rod.Browser, p *rod.Page) bool {
	if b == nil || p == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	pages, err := b.Context(ctx).Pages()
	if err != nil {
		return true
	}
	for _, pg := range pages {
		if pg.TargetID == p.TargetID {
			return false
		}
	}
	return true
}

func resolveBrowserExecutable(choice string, customExe string) string {
	if customExe != "" {
		if _, err := os.Stat(customExe); err == nil {
			return customExe
		}
	}

	pf := os.Getenv("ProgramFiles")
	pfx86 := os.Getenv("ProgramFiles(x86)")
	localApp := os.Getenv("LocalAppData")

	var candidates []string

	switch choice {
	case "brave":
		candidates = []string{
			filepath.Join(pf, `BraveSoftware\Brave-Browser\Application\brave.exe`),
			filepath.Join(pfx86, `BraveSoftware\Brave-Browser\Application\brave.exe`),
			filepath.Join(localApp, `BraveSoftware\Brave-Browser\Application\brave.exe`),
		}
	case "msedge":
		candidates = []string{
			filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(pfx86, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(localApp, `Microsoft\Edge\Application\msedge.exe`),
		}
	case "chromium":
		candidates = []string{
			filepath.Join(localApp, `Chromium\Application\chrome.exe`),
			filepath.Join(pf, `Chromium\Application\chrome.exe`),
			filepath.Join(pfx86, `Chromium\Application\chrome.exe`),
			filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(pfx86, `Google\Chrome\Application\chrome.exe`),
		}
	default:
		candidates = []string{
			filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(pfx86, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(localApp, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(pfx86, `Microsoft\Edge\Application\msedge.exe`),
		}
	}

	for _, c := range candidates {
		if c != "" {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	if path, has := launcher.LookPath(); has {
		return path
	}
	return ""
}

func attachInitScriptToPage(p *rod.Page) {
	if p == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = proto.PageAddScriptToEvaluateOnNewDocument{Source: AntiAdInitScript}.Call(p.Context(ctx))
}

func ensureBrowserSession(session *SessionHolder, cfg *AppConfig) (*rod.Browser, *rod.Page, error) {
	desiredBrowser := strings.ToLower(strings.TrimSpace(cfg.Get("General", "browser", "chrome")))
	keepSess := cfg.GetBool("Session", "keep_session_across_containers", true)

	if isBrowserAlive(session) && session.BrowserName != desiredBrowser {
		safeCloseBrowser(session)
	}

	if !isBrowserAlive(session) {
		customExe := strings.TrimSpace(cfg.Get("General", "executable_path", ""))
		exePath := resolveBrowserExecutable(desiredBrowser, customExe)
		if exePath == "" {
			return nil, nil, errors.New("no compatible browser (Chrome, Edge, Brave, Chromium) found on this PC")
		}

		relProfile := strings.TrimSpace(cfg.Get("Session", "profile_directory", "./chrome_filecrypt_profile"))
		profileDir, _ := filepath.Abs(filepath.Join(AppDir, relProfile))

		logMsg(fmt.Sprintf("Launching Browser (%s)...", exePath), "INFO", cfg)

		newLauncher := func() *launcher.Launcher {
			return launcher.New().
				Leakless(false).
				Bin(exePath).
				UserDataDir(profileDir).
				Headless(false).
				Delete("no-sandbox").
				Delete("enable-automation").
				Set("disable-blink-features", "AutomationControlled").
				Set("disable-background-mode").
				Set("disable-backgrounding-occluded-windows").
				Set("no-first-run").
				Set("no-default-browser-check").
				Set("start-maximized")
		}

		cleanStaleProfileLocks(profileDir)

		l := newLauncher()
		controlURL, err := l.Launch()
		if err != nil {
			logMsg("Waiting for previous browser session to finish closing and retrying...", "INFO", cfg)
			cleanStaleProfileLocks(profileDir)
			time.Sleep(600 * time.Millisecond)
			l = newLauncher()
			controlURL, err = l.Launch()
			if err != nil {
				return nil, nil, err
			}
		}

		if pid := l.PID(); pid > 0 {
			_ = os.MkdirAll(profileDir, 0755)
			_ = os.WriteFile(filepath.Join(profileDir, ".browser.pid"), []byte(strconv.Itoa(pid)), 0644)
		}

		b := rod.New().ControlURL(controlURL).NoDefaultDevice()
		if err := b.Connect(); err != nil {
			return nil, nil, err
		}

		pages, _ := b.Pages()
		var keepalive *rod.Page
		if len(pages) > 0 {
			keepalive = pages[0]
		} else {
			keepalive, err = b.Page(proto.TargetCreateTarget{URL: "about:blank"})
			if err != nil {
				return nil, nil, err
			}
		}
		attachInitScriptToPage(keepalive)
		_ = keepalive.Navigate("about:blank")
		_, _ = keepalive.Eval(`() => {
			document.title = 'Keep-Alive (Do Not Close)';
			if (document.documentElement) document.documentElement.style.backgroundColor = '#0d0f14';
		}`)

		if uaRes, err := keepalive.Eval(`() => navigator.userAgent`); err == nil {
			session.UserAgent = uaRes.Value.Str()
		}

		waitTargetCreated := b.EachEvent(func(e *proto.TargetTargetCreated) {
			if e.TargetInfo.Type == proto.TargetTargetInfoTypePage {
				go func(targetID proto.TargetTargetID) {
					if pg, err := b.PageFromTarget(targetID); err == nil {
						attachInitScriptToPage(pg)
					}
				}(e.TargetInfo.TargetID)
			}
		})
		go waitTargetCreated()

		router := b.HijackRequests()
		adKeywords := []string{
			"popads", "propeller", "adsterra", "exoclick", "doubleclick",
			"syndication", "ddownload.com/pricing", "werbemanager", "0pixel.php",
			"this-is-not-tracking.php", "bla3.php", "meh.php", "muh2.php",
		}

		_ = router.Add("*", "", func(h *rod.Hijack) {
			reqURL := h.Request.URL().String()
			urlLower := strings.ToLower(reqURL)

			for _, kw := range adKeywords {
				if strings.Contains(urlLower, kw) {
					h.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
					return
				}
			}

			phase := session.GetPhase()

			if strings.Contains(urlLower, "/link/") {
				session.EmitCapturedLink(reqURL)
			}

			switch phase {
			case "LOAD_CONTAINER":
				if strings.Contains(urlLower, "/link/") || strings.Contains(urlLower, "/go/") {
					h.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
					return
				}
				if h.Request.Type() == proto.NetworkResourceTypeDocument {
					parsed, err := url.Parse(urlLower)
					if err == nil && parsed.Host != "" {
						host := parsed.Host
						if !strings.Contains(host, "filecrypt.") &&
							!strings.Contains(host, "cutcaptcha.") &&
							!strings.Contains(host, "cloudflare.") {
							h.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
							return
						}
					}
				}

			case "HARVEST_LINKS":
				// During Step 1/2 (clicking buttons on the container tab to harvest /Link/ URLs),
				// respond with 204 No Content to document navigations so the container tab stays locked.
				// During Step 2/2 (RESOLVE_LINKS), this lock is disabled so Single Tab Mode's workerPage
				// can freely navigate through /Link/ -> /Go/ without getting a 204!
				if h.Request.Type() == proto.NetworkResourceTypeDocument {
					if strings.Contains(urlLower, "/link/") || strings.Contains(urlLower, "/go/") || strings.Contains(urlLower, "/container/") {
						h.Response.SetHeader("Content-Type", "text/plain")
						h.Response.Payload().ResponseCode = 204
						return
					}
				}
			}

			h.ContinueRequest(&proto.FetchContinueRequest{})
		})
		go router.Run()

		session.mu.Lock()
		session.Launcher = l
		session.ProfileDir = profileDir
		session.Browser = b
		session.KeepalivePage = keepalive
		session.BrowserName = desiredBrowser
		session.mu.Unlock()
	} else {
		if !keepSess {
			logMsg("Resetting session cookies for new container per Settings...", "INFO", cfg)
			_ = session.Browser.SetCookies(nil)
		} else {
			logMsg("Re-using active browser session (solved Captchas stay remembered!)...", "INFO", cfg)
		}
	}

	return session.Browser, session.KeepalivePage, nil
}

// ==============================================================================
// CONTAINER EXTRACTION & LINK RESOLUTION
// ==============================================================================

const FreezeAndExtractJS = `() => {
    let highestId = window.setTimeout(() => {}, 0);
    for (let i = 0; i <= highestId; i++) {
        window.clearTimeout(i);
        window.clearInterval(i);
    }
    window.setTimeout = function() { return 0; };
    window.setInterval = function() { return 0; };

    const h2 = document.querySelector('.container h2, .box h2, h2');
    const containerTitle = h2 ? h2.innerText.trim() : 'container';

    const rows = Array.from(document.querySelectorAll('table tbody tr'));
    const parsed = [];

    rows.forEach((tr, idx) => {
        tr.setAttribute('data-pw-index', idx);

        const titleTd = tr.querySelector('td[title]');
        let filename = titleTd ? titleTd.getAttribute('title').trim() : 'unknown';

        let host = 'unknown';
        if (titleTd) {
            const hostSpan = titleTd.querySelector('span, a');
            if (hostSpan && hostSpan.innerText.trim()) {
                host = hostSpan.innerText.trim().toLowerCase();
            } else {
                const lines = titleTd.innerText.split('\n').map(s => s.trim()).filter(Boolean);
                if (lines.length > 1) host = lines[lines.length - 1].toLowerCase();
            }
        }

        const tds = Array.from(tr.querySelectorAll('td'));
        let size = 'unknown';
        for (const td of tds) {
            const txt = td.innerText.trim();
            if (/^\d+(\.\d+)?\s*(KB|MB|GB|TB|B|Bytes)$/i.test(txt)) {
                size = txt;
                break;
            }
        }

        const statusTd = tr.querySelector('td.status');
        let hasExplicitOfflineIcon = false;
        if (statusTd) {
            if (statusTd.querySelector('i.offline, .offline, [class*="offline"], [title*="offline" i], .status_2')) {
                hasExplicitOfflineIcon = true;
            } else if (/\boffline\b/i.test(statusTd.className)) {
                hasExplicitOfflineIcon = true;
            }
        }

        let isTinyDeadPlaceholder = (
            filename === 'undefined' ||
            filename === 'unknown' ||
            /^\d+\s*(b|bytes)$/i.test(size)
        );
        let hasValidRealFile = (
            !isTinyDeadPlaceholder &&
            /^\d+(\.\d+)?\s*(KB|MB|GB|TB)$/i.test(size)
        );

        let isOnline = !(hasExplicitOfflineIcon || isTinyDeadPlaceholder);

        const btn = tr.querySelector('button, a.download, td:last-child a, td:last-child button');
        if (btn) {
            btn.setAttribute('data-pw-btn', idx);

            let directLink = "";
            const allEls = [tr, btn, ...Array.from(tr.querySelectorAll('*'))];
            for (const el of allEls) {
                if (!el.attributes) continue;
                for (const attr of Array.from(el.attributes)) {
                    const val = attr.value || '';
                    const match = val.match(/(?:https?:\/\/[^'"\s]+)?\/Link\/[A-Za-z0-9]+\.html/i);
                    if (match) {
                        directLink = match[0];
                        break;
                    }
                }
                if (directLink) break;
            }

            parsed.push({
                index: idx,
                rowid: tr.getAttribute('data-rowid') || '',
                filename: filename,
                host: host,
                size: size,
                is_online: isOnline,
                has_valid_real_file: hasValidRealFile,
                direct_link: directLink
            });
        }
    });

    return {
        title: containerTitle,
        rows: parsed
    };
}`

func freezePageAndExtractContainerInfo(page *rod.Page, cfg *AppConfig) (*ContainerInfo, error) {
	logMsg("Freezing page timers and parsing container title + table...", "DEBUG", cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	res, err := page.Context(ctx).Eval(FreezeAndExtractJS)
	if err != nil {
		return nil, err
	}

	var info ContainerInfo
	if err := res.Value.Unmarshal(&info); err != nil {
		return nil, err
	}

	byHost := make(map[string][]int)
	for i, r := range info.Rows {
		byHost[r.Host] = append(byHost[r.Host], i)
	}

	for _, indices := range byHost {
		onlineCount := 0
		for _, idx := range indices {
			if info.Rows[idx].IsOnline {
				onlineCount++
			}
		}
		if len(indices) > 1 && float64(onlineCount)/float64(len(indices)) >= 0.75 {
			for _, idx := range indices {
				if !info.Rows[idx].IsOnline && info.Rows[idx].HasValidRealFile {
					info.Rows[idx].IsOnline = true
				}
			}
		}
	}

	return &info, nil
}

func resolveRelativeURL(baseStr, refStr string) string {
	baseU, err1 := url.Parse(baseStr)
	refU, err2 := url.Parse(refStr)
	if err1 != nil || err2 != nil {
		return refStr
	}
	return baseU.ResolveReference(refU).String()
}

func pageCurrentURL(p *rod.Page) string {
	if p == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	info, err := p.Context(ctx).Info()
	if err != nil || info == nil {
		return ""
	}
	return info.URL
}

func harvestIntermediateLinkURL(
	session *SessionHolder,
	b *rod.Browser,
	mainPage *rod.Page,
	keepalivePage *rod.Page,
	row RowInfo,
	cfg *AppConfig,
) (string, error) {
	if checkCancelKeyPressed() {
		return "", ErrUserCancelled
	}
	if isPageClosed(b, mainPage) {
		return "", ErrBrowserClosed
	}

	mainURL := pageCurrentURL(mainPage)
	if row.DirectLink != "" {
		fullLink := resolveRelativeURL(mainURL, row.DirectLink)
		logMsg(fmt.Sprintf("Found direct /Link/ URL in DOM: %s", fullLink), "DEBUG", cfg)
		return fullLink, nil
	}

	session.SetPhase("HARVEST_LINKS")
	captureCh := make(chan string, 4)
	session.SetCaptureChan(captureCh)
	defer func() {
		session.SetCaptureChan(nil)
		session.SetPhase("IDLE")
	}()

	defer func() {
		if !isPageClosed(b, mainPage) {
			evalCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			_, _ = mainPage.Context(evalCtx).Eval(`() => { window.__CAPTURE_MODE__ = false; }`)
			cancel()
		}
		ctxP, cancelP := context.WithTimeout(context.Background(), 1200*time.Millisecond)
		if pages, err := b.Context(ctxP).Pages(); err == nil {
			for _, pg := range pages {
				if pg.TargetID != mainPage.TargetID && pg.TargetID != keepalivePage.TargetID {
					safeClosePage(pg)
				}
			}
		}
		cancelP()
	}()

	evalCtx, evalCancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, _ = mainPage.Context(evalCtx).Eval(`() => { window.__CAPTURE_MODE__ = true; window.__CAPTURED_LINK_URL__ = null; }`)
	evalCancel()

	btnSelector := fmt.Sprintf("[data-pw-btn='%d']", row.Index)
	elCtx, elCancel := context.WithTimeout(context.Background(), 3*time.Second)
	el, err := mainPage.Context(elCtx).Element(btnSelector)
	if err != nil {
		elCancel()
		logMsg(fmt.Sprintf("Could not locate button for row %d: %v", row.Index, err), "DEBUG", cfg)
		return "", nil
	}
	_ = el.ScrollIntoView()
	_, _ = el.Eval(`() => this.click()`)
	elCancel()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if checkCancelKeyPressed() {
			return "", ErrUserCancelled
		}
		if isPageClosed(b, mainPage) {
			return "", ErrBrowserClosed
		}
		select {
		case captured := <-captureCh:
			if strings.Contains(strings.ToLower(captured), "/link/") {
				full := resolveRelativeURL(mainURL, captured)
				logMsg(fmt.Sprintf("Harvested intermediate /Link/ URL: %s", full), "DEBUG", cfg)
				return full, nil
			}
		default:
		}

		pollCtx, pollCancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		jsRes, err := mainPage.Context(pollCtx).Eval(`() => window.__CAPTURED_LINK_URL__`)
		pollCancel()
		if err == nil {
			jsCaptured := jsRes.Value.Str()
			if jsCaptured != "" && strings.Contains(strings.ToLower(jsCaptured), "/link/") {
				full := resolveRelativeURL(mainURL, jsCaptured)
				logMsg(fmt.Sprintf("Harvested intermediate /Link/ URL: %s", full), "DEBUG", cfg)
				return full, nil
			}
		}
		time.Sleep(80 * time.Millisecond)
	}

	logMsg(fmt.Sprintf("Could not harvest /Link/ URL for row %d (timeout)", row.Index), "DEBUG", cfg)
	return "", nil
}

func isExternalTarget(targetURL string) bool {
	if targetURL == "" {
		return false
	}
	lower := strings.ToLower(targetURL)
	for _, prefix := range []string{"about:", "chrome", "data:", "blob:"} {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	u, err := url.Parse(targetURL)
	if err != nil || u.Host == "" {
		return false
	}
	domain := strings.ToLower(u.Host)
	for _, ig := range []string{"filecrypt.", "cutcaptcha.", "cloudflare."} {
		if strings.Contains(domain, ig) {
			return false
		}
	}
	return true
}

var (
	goLinkRegex  = regexp.MustCompile(`(?i)(?:https?://[^'"\s]+)?/Go/[A-Za-z0-9]+\.html`)
	extHrefRegex = regexp.MustCompile(`(?i)(?:location\.href\s*=\s*|url=)['"]?(https?://[^'"\s>]+)`)
)

func resolveLinkInvisibly(
	ctx context.Context,
	session *SessionHolder,
	b *rod.Browser,
	mainPage *rod.Page,
	linkURL string,
	cfg *AppConfig,
) (string, error) {
	currentURL := linkURL
	referer := pageCurrentURL(mainPage)

	ckCtx, ckCancel := context.WithTimeout(ctx, 2*time.Second)
	cookies, err := b.Context(ckCtx).GetCookies()
	ckCancel()
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: sharedHTTPTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var cookieParts []string
	for _, ck := range cookies {
		cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", ck.Name, ck.Value))
	}
	cookieHeader := strings.Join(cookieParts, "; ")

	for hop := 0; hop < 6; hop++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		logMsg(fmt.Sprintf("[Invisible Hop %d] GET %s", hop+1, currentURL), "DEBUG", cfg)

		req, err := http.NewRequestWithContext(ctx, "GET", currentURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Referer", referer)
		if session.UserAgent != "" {
			req.Header.Set("User-Agent", session.UserAgent)
		}
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		if cookieHeader != "" && strings.Contains(strings.ToLower(req.URL.Host), "filecrypt.") {
			req.Header.Set("Cookie", cookieHeader)
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		_ = resp.Body.Close()

		if resp.StatusCode == 301 || resp.StatusCode == 302 || resp.StatusCode == 303 || resp.StatusCode == 307 || resp.StatusCode == 308 {
			loc := resp.Header.Get("Location")
			if loc != "" {
				nextURL := resolveRelativeURL(currentURL, loc)
				logMsg(fmt.Sprintf("[Invisible Location Header] %s", nextURL), "DEBUG", cfg)
				if isExternalTarget(nextURL) {
					return nextURL, nil
				}
				referer = currentURL
				currentURL = nextURL
				continue
			}
		}

		body := string(bodyBytes)

		if goMatch := goLinkRegex.FindString(body); goMatch != "" {
			nextURL := resolveRelativeURL(currentURL, goMatch)
			logMsg(fmt.Sprintf("[Invisible Parsed /Go/ URL] %s", nextURL), "DEBUG", cfg)
			referer = currentURL
			currentURL = nextURL
			continue
		}

		if extMatch := extHrefRegex.FindStringSubmatch(body); len(extMatch) > 1 {
			candidate := extMatch[1]
			if isExternalTarget(candidate) {
				return candidate, nil
			}
		}

		break
	}

	return "", nil
}

func formatResolvedItemMarkup(indexNum, totalNum int, filename, finalURL string, success bool) string {
	idxStr := fmt.Sprintf("%s[%02d/%02d]%s", AccentTeal, indexNum, totalNum, Reset)
	col := AccentCoral
	if success {
		col = AccentGreen
	}
	return fmt.Sprintf("  %s %s%s%s\n         %s╰──›%s %s%s%s",
		idxStr, TextMain, filename, Reset,
		TextMuted, Reset,
		Bold+col, finalURL, Reset)
}

func printResolvedItem(indexNum, totalNum int, filename, finalURL string, success bool) {
	fmt.Println(formatResolvedItemMarkup(indexNum, totalNum, filename, finalURL, success))
}

type WorkerHolder struct {
	Page *rod.Page
}

func resolveLinkSmart(
	session *SessionHolder,
	b *rod.Browser,
	mainPage *rod.Page,
	workerHolder *WorkerHolder,
	linkURL string,
	filename string,
	indexNum int,
	totalNum int,
	cfg *AppConfig,
) (string, error) {
	if checkCancelKeyPressed() {
		return "", ErrUserCancelled
	}
	if isPageClosed(b, mainPage) {
		return "", ErrBrowserClosed
	}

	session.SetPhase("RESOLVE_LINKS")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var abortReason error
	var abortMu sync.Mutex
	setAbortReason := func(err error) {
		abortMu.Lock()
		if abortReason == nil {
			abortReason = err
		}
		abortMu.Unlock()
		cancel()
	}
	getAbortReason := func() error {
		abortMu.Lock()
		defer abortMu.Unlock()
		return abortReason
	}

	go func() {
		ticker := time.NewTicker(75 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if checkCancelKeyPressed() {
					setAbortReason(ErrUserCancelled)
					return
				}
				if isPageClosed(b, mainPage) {
					setAbortReason(ErrBrowserClosed)
					return
				}
			}
		}
	}()

	resolveMode := strings.ToLower(strings.TrimSpace(cfg.Get("Behavior", "resolve_mode", "invisible")))

	if resolveMode == "invisible" {
		finalURL, err := resolveLinkInvisibly(ctx, session, b, mainPage, linkURL, cfg)
		if r := getAbortReason(); r != nil {
			return "", r
		}
		if err == nil && finalURL != "" {
			if !strings.Contains(strings.ToLower(finalURL), "mega.nz") || strings.Contains(finalURL, "#") {
				printResolvedItem(indexNum, totalNum, filename, finalURL, true)
				return finalURL, nil
			}
			logMsg("Invisible resolver needed DOM fallback, using persistent dark background tab...", "DEBUG", cfg)
		} else if err != nil {
			logMsg(fmt.Sprintf("Invisible resolver note: %v", err), "DEBUG", cfg)
		}
	}

	if r := getAbortReason(); r != nil {
		return "", r
	}

	if workerHolder.Page == nil || isPageClosed(b, workerHolder.Page) {
		createCtx, createCancel := context.WithTimeout(ctx, 4*time.Second)
		wp, err := b.Context(createCtx).Page(proto.TargetCreateTarget{URL: "about:blank"})
		createCancel()
		if err != nil {
			if r := getAbortReason(); r != nil {
				return "", r
			}
			printResolvedItem(indexNum, totalNum, filename, "[ERROR - Could not open background resolver tab]", false)
			return "", nil
		}
		wp = wp.Context(context.Background())
		attachInitScriptToPage(wp)
		_ = proto.NetworkEnable{}.Call(wp.Context(ctx))
		workerHolder.Page = wp
		actCtx, actCancel := context.WithTimeout(ctx, 1*time.Second)
		_, _ = mainPage.Context(actCtx).Activate()
		actCancel()
	}

	workerPage := workerHolder.Page
	_ = proto.NetworkEnable{}.Call(workerPage.Context(ctx))

	finalCh := make(chan string, 4)
	var once sync.Once
	deliver := func(u string) {
		once.Do(func() {
			finalCh <- u
		})
	}

	waitEvents := workerPage.Context(ctx).EachEvent(
		func(e *proto.NetworkResponseReceived) {
			status := e.Response.Status
			if status == 301 || status == 302 || status == 303 || status == 307 || status == 308 {
				for k, v := range e.Response.Headers {
					if strings.EqualFold(k, "location") && v.Str() != "" {
						fullLoc := resolveRelativeURL(e.Response.URL, v.Str())
						logMsg(fmt.Sprintf("[HTTP %d Location Header] %s", status, fullLoc), "DEBUG", cfg)
						if isExternalTarget(fullLoc) {
							deliver(fullLoc)
						}
					}
				}
			}
		},
		func(e *proto.NetworkRequestWillBeSent) {
			if e.RedirectResponse != nil {
				status := e.RedirectResponse.Status
				for k, v := range e.RedirectResponse.Headers {
					if strings.EqualFold(k, "location") && v.Str() != "" {
						fullLoc := resolveRelativeURL(e.RedirectResponse.URL, v.Str())
						logMsg(fmt.Sprintf("[Redirect %d Location] %s", status, fullLoc), "DEBUG", cfg)
						if isExternalTarget(fullLoc) {
							deliver(fullLoc)
						}
					}
				}
			}
			if isExternalTarget(e.Request.URL) && e.Type == proto.NetworkResourceTypeDocument {
				deliver(e.Request.URL)
			}
		},
		func(e *proto.PageFrameNavigated) {
			if e.Frame.ParentID == "" {
				if isExternalTarget(e.Frame.URL) {
					deliver(e.Frame.URL)
				}
			}
		},
	)
	go waitEvents()

	mainReferer := pageCurrentURL(mainPage)
	actCtx, actCancel := context.WithTimeout(ctx, 1*time.Second)
	_, _ = mainPage.Context(actCtx).Activate()
	actCancel()

	go func() {
		_, _ = proto.PageNavigate{
			URL:      linkURL,
			Referrer: mainReferer,
		}.Call(workerPage.Context(ctx))
	}()

	go func() {
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cur := pageCurrentURL(workerPage)
				if isExternalTarget(cur) {
					deliver(cur)
					return
				}
			}
		}
	}()

	select {
	case finalURL := <-finalCh:
		if strings.Contains(strings.ToLower(finalURL), "mega.nz") && !strings.Contains(finalURL, "#") {
			for i := 0; i < 15; i++ {
				if ctx.Err() != nil {
					break
				}
				cur := pageCurrentURL(workerPage)
				if strings.Contains(cur, "#") {
					finalURL = cur
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		_ = proto.PageStopLoading{}.Call(workerPage.Context(stopCtx))
		stopCancel()

		printResolvedItem(indexNum, totalNum, filename, finalURL, true)
		return finalURL, nil

	case <-ctx.Done():
		if r := getAbortReason(); r != nil {
			if workerHolder.Page != nil {
				safeClosePage(workerHolder.Page)
				workerHolder.Page = nil
			}
			return "", r
		}
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_ = proto.PageStopLoading{}.Call(workerPage.Context(stopCtx))
		stopCancel()
		printResolvedItem(indexNum, totalNum, filename, "[TIMEOUT - Failed to resolve within 20s]", false)
		return "", nil
	}
}

func waitForTableOrCaptcha(b *rod.Browser, keepalivePage *rod.Page, cfg *AppConfig) (*rod.Page, error) {
	announcedCaptcha := false
	drainKeyboardBuffer()

	for i := 0; i < 1200; i++ {
		if checkCancelKeyPressed() {
			logMsg("Cancelled by user ([Esc] / [X]). Returning to Home...", "WARN", cfg)
			return nil, ErrUserCancelled
		}

		ctxP, cancelP := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		pages, err := b.Context(ctxP).Pages()
		cancelP()
		if err != nil || len(pages) == 0 || (len(pages) == 1 && pages[0].TargetID == keepalivePage.TargetID) {
			logMsg("Filecrypt tab was closed by user.", "WARN", cfg)
			return nil, ErrBrowserClosed
		}

		for _, p := range pages {
			if p.TargetID == keepalivePage.TargetID {
				continue
			}

			if checkCancelKeyPressed() {
				logMsg("Cancelled by user ([Esc] / [X]). Returning to Home...", "WARN", cfg)
				return nil, ErrUserCancelled
			}

			u := strings.ToLower(pageCurrentURL(p))
			if u != "" && !strings.HasPrefix(u, "about:") && !strings.HasPrefix(u, "chrome:") {
				parsed, err := url.Parse(u)
				if err == nil && parsed.Host != "" {
					domain := parsed.Host
					if !strings.Contains(domain, "filecrypt.") &&
						!strings.Contains(domain, "cutcaptcha.") &&
						!strings.Contains(domain, "cloudflare.") {
						logMsg(fmt.Sprintf("Closed hijacked Ad Tab: %s", u), "DEBUG", cfg)
						safeClosePage(p)
						continue
					}
				}
			}

			evalCtx, evalCancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
			res, err := p.Context(evalCtx).Eval(`() => {
				const tbl = document.querySelectorAll('table tbody tr td.status, table tbody tr td[title]').length;
				const cap = document.querySelectorAll('form#cform, .protection').length ||
				            (document.body && document.body.innerText.includes('Security Check') ? 1 : 0);
				return { tbl, cap };
			}`)
			evalCancel()
			if err == nil {
				tblCount := res.Value.Get("tbl").Int()
				capCount := res.Value.Get("cap").Int()

				if tblCount > 0 {
					if !announcedCaptcha {
						logMsg("Active Session Cookie Detected! Captcha skipped & table loaded directly.", "SUCCESS", cfg)
					} else {
						logMsg("Captcha solved! Download links table loaded.", "SUCCESS", cfg)
					}
					for w := 0; w < 8; w++ {
						if checkCancelKeyPressed() {
							return nil, ErrUserCancelled
						}
						time.Sleep(100 * time.Millisecond)
					}
					return p, nil
				}

				if !announcedCaptcha && capCount > 0 {
					setWindowTitle("Waiting for Captcha in Browser...")
					logMsg("Security Check detected! Please solve the Captcha in Browser...", "INFO", cfg)
					announcedCaptcha = true
				}
			}
		}

		for w := 0; w < 5; w++ {
			if checkCancelKeyPressed() {
				logMsg("Cancelled by user ([Esc] / [X]). Returning to Home...", "WARN", cfg)
				return nil, ErrUserCancelled
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil, errors.New("timed out waiting for container table")
}

var (
	nonAlphaNumRegex      = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	nonTitleCharsRegex    = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	containerIDFromURLReg = regexp.MustCompile(`(?i)/Container/([A-Za-z0-9]+)`)
)

type ExtractedURL struct {
	Filename string
	URL      string
}

type DisplayBatchItem struct {
	Filename string
	URL      string
	Success  bool
}

type HostBatch struct {
	Host  string
	Items []DisplayBatchItem
}

func saveHostURLs(hostName, containerTitle, containerURL string, extractedURLs []ExtractedURL, cfg *AppConfig) string {
	if len(extractedURLs) == 0 {
		return ""
	}

	if !cfg.GetBool("Output", "save_to_file", true) {
		return fmt.Sprintf("%s✓ Extracted %d URLs for [%s]%s %s(File saving disabled — Press [C] to copy)%s",
			Bold+AccentGreen, len(extractedURLs), hostName, Reset,
			TextMuted, Reset)
	}

	saveDir, _ := filepath.Abs(filepath.Join(AppDir, cfg.Get("Output", "save_directory", "./decrypted_links")))
	useTitle := cfg.GetBool("Output", "include_container_title_in_filename", true)

	_ = os.MkdirAll(saveDir, 0755)

	cleanHost := strings.Trim(nonAlphaNumRegex.ReplaceAllString(hostName, "_"), "_")
	var filename string
	if useTitle && containerTitle != "" {
		cleanTitle := strings.Trim(nonTitleCharsRegex.ReplaceAllString(containerTitle, "_"), "_")
		if strings.EqualFold(cleanTitle, "container") {
			if m := containerIDFromURLReg.FindStringSubmatch(containerURL); len(m) > 1 {
				cleanTitle = "container_" + m[1]
			}
		}
		if len(cleanTitle) > 60 {
			cleanTitle = cleanTitle[:60]
		}
		filename = fmt.Sprintf("%s_%s.txt", cleanTitle, cleanHost)
	} else {
		filename = fmt.Sprintf("decrypted_%s.txt", cleanHost)
	}

	outputPath := filepath.Join(saveDir, filename)

	var urlList []string
	for _, item := range extractedURLs {
		urlList = append(urlList, item.URL)
	}
	rawURLsText := strings.Join(urlList, "\n")
	_ = os.WriteFile(outputPath, []byte(rawURLsText+"\n"), 0644)

	return fmt.Sprintf("%s✓ Saved %d URLs for [%s]%s %s──›%s %s%s%s",
		Bold+AccentGreen, len(extractedURLs), hostName, Reset,
		TextMuted, Reset,
		AccentTeal, outputPath, Reset)
}

// ==============================================================================
// INLINE PROGRESS BAR (STEP 1/2 PREPARING LINKS)
// ==============================================================================

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func renderProgressBarLine(completed, total, frameIdx int) {
	if total <= 0 {
		total = 1
	}
	pct := (completed * 100) / total
	barWidth := 38
	filled := (completed * barWidth) / total
	if filled > barWidth {
		filled = barWidth
	}

	spin := AccentTeal + spinnerFrames[frameIdx%len(spinnerFrames)] + Reset
	if completed >= total {
		spin = AccentGreen + "✓" + Reset
	}

	barColor := AccentTeal
	if completed >= total {
		barColor = AccentGreen
	}

	bar := barColor + strings.Repeat("━", filled) + Reset + BorderSubtle + strings.Repeat("━", barWidth-filled) + Reset
	fmt.Printf("\r  %s   %sStep 1/2 (Preparing Links)%s %s %s%3d%%%s",
		spin, TextMain, Reset, bar, AccentTeal, pct, Reset)
}

// ==============================================================================
// INDIVIDUAL LINK SELECTION SUBMENU ([L])
// ==============================================================================

func promptIndividualLinkSelection(
	b *rod.Browser,
	activeContainerPage *rod.Page,
	containerTitle string,
	hostList []string,
	hostsMap map[string][]RowInfo,
	decryptedRowURLs map[int]string,
	cfg *AppConfig,
) ([]RowInfo, bool) {
	var orderedRows []RowInfo
	for _, h := range hostList {
		orderedRows = append(orderedRows, hostsMap[h]...)
	}
	totalLinks := len(orderedRows)
	if totalLinks == 0 {
		return nil, false
	}

	const pageSize = 25
	totalPages := (totalLinks + pageSize - 1) / pageSize
	currentPage := 1
	warningMsg := ""

	for !isPageClosed(b, activeContainerPage) {
		breadcrumb := "App / Container / Select Individual Links"
		if totalPages > 1 {
			breadcrumb = fmt.Sprintf("App / Container / Select Individual Links (Page %d of %d)", currentPage, totalPages)
		}
		printAppHeader(
			breadcrumb,
			cfg,
			!isDebug(cfg),
			fmt.Sprintf("Select Links — %s", containerTitle),
		)

		renderContainerInfoPanel(containerTitle, totalLinks, len(hostList), "", 0)

		if warningMsg != "" {
			printPanel([]string{"  " + warningMsg}, "", AccentAmber, 0, 1)
			warningMsg = ""
		}

		// totalInnerWidth = UIWidth - 4 = 104. With 4 columns, separator overhead = (3*4)-1 = 11,
		// so column widths sum to 93 (6 + 16 + 59 + 12 = 93) giving 59 full chars for Filename!
		const filenameColWidth = 59
		cols := []tableCol{
			{Header: "Key", Width: 6, Justify: "center", NoWrap: true},
			{Header: "Host Provider", Width: 16, Justify: "left", NoWrap: true},
			{Header: "Filename", Width: filenameColWidth, Justify: "left", NoWrap: true},
			{Header: "Status", Width: 12, Justify: "right", NoWrap: true},
		}

		startIdx := (currentPage - 1) * pageSize
		endIdx := startIdx + pageSize
		if endIdx > totalLinks {
			endIdx = totalLinks
		}

		var tblRows []tableRow
		prevHost := ""
		for idx := startIdx; idx < endIdx; idx++ {
			r := orderedRows[idx]
			linkNum := idx + 1
			if prevHost != "" && prevHost != r.Host {
				tblRows = append(tblRows, tableRow{IsSection: true})
			}
			prevHost = r.Host

			keyStr := fmt.Sprintf("%s[%d]%s", Bold+AccentTeal, linkNum, Reset)
			hostStyle := TextMain
			fileStyle := TextMain
			if !r.IsOnline {
				hostStyle = TextMuted
				fileStyle = TextMuted
			}

			fileCell := truncateMiddlePlain(r.Filename, filenameColWidth, fileStyle)

			var statusStr string
			if _, done := decryptedRowURLs[r.Index]; done {
				statusStr = Bold + AccentGreen + "✓ Decrypted" + Reset
			} else if r.IsOnline {
				statusStr = AccentGreen + "● Online" + Reset
			} else {
				statusStr = AccentCoral + "○ Offline" + Reset
			}

			tblRows = append(tblRows, tableRow{Cells: []string{
				keyStr,
				hostStyle + r.Host + Reset,
				fileCell,
				statusStr,
			}})
		}

		tblRows = append(tblRows, tableRow{IsSection: true})
		if currentPage < totalPages {
			nextStart := currentPage*pageSize + 1
			nextEnd := (currentPage + 1) * pageSize
			if nextEnd > totalLinks {
				nextEnd = totalLinks
			}
			tblRows = append(tblRows, tableRow{Cells: []string{
				Bold + AccentAmber + "[N]" + Reset,
				Bold + AccentAmber + "Next Page ▶" + Reset,
				fmt.Sprintf("%sView links %d–%d (Page %d of %d)%s",
					AccentAmber, nextStart, nextEnd, currentPage+1, totalPages, Reset),
				"",
			}})
		}
		if currentPage > 1 {
			prevStart := (currentPage-2)*pageSize + 1
			prevEnd := (currentPage - 1) * pageSize
			tblRows = append(tblRows, tableRow{Cells: []string{
				Bold + AccentLavender + "[P]" + Reset,
				Bold + AccentLavender + "◀ Previous Page" + Reset,
				fmt.Sprintf("%sView links %d–%d (Page %d of %d)%s",
					AccentLavender, prevStart, prevEnd, currentPage-1, totalPages, Reset),
				"",
			}})
		}
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentGreen + "[A]" + Reset,
			Bold + TextMain + "Decrypt All Links" + Reset,
			fmt.Sprintf("%sExtract all %d links across all hosts%s", TextMuted, totalLinks, Reset),
			"",
		}})
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentAmber + "[Esc]" + Reset,
			TextMuted + "Back" + Reset,
			TextMuted + "Return to Host Selection without decrypting" + Reset,
			"",
		}})

		panelTitle := ""
		panelBorder := BorderSubtle
		if totalPages > 1 {
			navHint := "Press [N] for Next Page"
			if currentPage == totalPages {
				navHint = "Press [P] for Previous Page"
			} else if currentPage > 1 {
				navHint = "Press [N] Next Page  /  [P] Previous Page"
			}
			panelTitle = fmt.Sprintf("%sPAGE %d OF %d%s %s(Showing Links %d–%d of %d)%s  %s│%s  %s%s%s",
				Bold+AccentAmber, currentPage, totalPages, Reset,
				TextMain, startIdx+1, endIdx, totalLinks, Reset,
				BorderSubtle, Reset,
				Bold+AccentTeal, navHint, Reset)
			panelBorder = BorderAccent
		}

		printPanel(renderTable(cols, tblRows, true, UIWidth-4), panelTitle, panelBorder, 0, 1)

		fmt.Printf("  %sTip: Enter link numbers like%s %s1%s %sor%s %s1, 3, 5%s %sor%s %s2-6%s %sor%s %sA%s %s(or press%s %s[Esc]%s %sto return).%s\n",
			TextMuted, Reset,
			Bold+AccentTeal, Reset,
			TextMuted, Reset,
			Bold+AccentTeal, Reset,
			TextMuted, Reset,
			Bold+AccentTeal, Reset,
			TextMuted, Reset,
			Bold+AccentGreen, Reset,
			TextMuted, Reset,
			Bold+AccentAmber, Reset,
			TextMuted, Reset)
		if totalPages > 1 {
			fmt.Printf("  %sInfo: You can type any link number%s %s1–%d%s %sfrom any page without flipping pages first.%s\n",
				TextMuted, Reset,
				Bold+TextMain, totalLinks, Reset,
				TextMuted, Reset)
		}

		var promptParts []string
		promptParts = append(promptParts, fmt.Sprintf("1-%d", totalLinks))
		if currentPage < totalPages {
			promptParts = append(promptParts, "N=Next Page")
		}
		if currentPage > 1 {
			promptParts = append(promptParts, "P=Prev Page")
		}
		promptParts = append(promptParts, "A", "Esc")
		promptHint := strings.Join(promptParts, ", ")

		inp, ok := readTextWithEscAndAbort(
			fmt.Sprintf("\n  %sSelect link(s) to decrypt [%s]: %s", AccentTeal, promptHint, Reset),
			func() bool { return isPageClosed(b, activeContainerPage) },
		)
		if !ok || isPageClosed(b, activeContainerPage) {
			return nil, false
		}
		if inp == "" || strings.EqualFold(inp, "B") || strings.EqualFold(inp, "ESC") {
			return nil, false
		}
		if totalPages > 1 && (strings.EqualFold(inp, "N") || strings.EqualFold(inp, "NEXT")) {
			if currentPage < totalPages {
				currentPage++
			} else {
				warningMsg = fmt.Sprintf("%s! You are already on the last page (Page %d of %d). Press [P] for Previous Page.%s", AccentAmber, currentPage, totalPages, Reset)
			}
			continue
		}
		if totalPages > 1 && (strings.EqualFold(inp, "P") || strings.EqualFold(inp, "PREV")) {
			if currentPage > 1 {
				currentPage--
			} else {
				warningMsg = fmt.Sprintf("%s! You are already on the first page (Page 1 of %d). Press [N] for Next Page.%s", AccentAmber, totalPages, Reset)
			}
			continue
		}

		indices, err := parseLinkSelectionInput(inp, totalLinks)
		if err != nil {
			warningMsg = fmt.Sprintf("%s! Invalid selection: %v. Try e.g. 1 or 1,3,5 or 2-4 or A.%s", AccentAmber, err, Reset)
			continue
		}

		var chosen []RowInfo
		for _, num := range indices {
			chosen = append(chosen, orderedRows[num-1])
		}
		return chosen, true
	}

	return nil, false
}

// ==============================================================================
// SINGLE CONTAINER WORKFLOW
// ==============================================================================

func processSingleContainer(containerURL string, cfg *AppConfig, session *SessionHolder) string {
	printAppHeader(
		"App / Container / Loading & Security Check",
		cfg,
		!isDebug(cfg),
		"Loading Container...",
	)
	printCancelTip("Home")
	drainKeyboardBuffer()

	b, keepalivePage, err := ensureBrowserSession(session, cfg)
	if err != nil {
		logMsg(fmt.Sprintf("Could not launch browser: %v", err), "ERROR", cfg)
		logMsg("Tip: Make sure another instance of the browser profile isn't already open, or press 'S' to switch browsers.", "WARN", cfg)
		fmt.Printf("\n  %sPress Enter to return to main menu...%s", AccentAmber, Reset)
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return "NEW_CONTAINER"
	}

	if checkCancelKeyPressed() {
		return "CANCELLED_TO_HOME"
	}

	if pages, err := b.Pages(); err == nil {
		for _, pOld := range pages {
			if pOld.TargetID != keepalivePage.TargetID {
				safeClosePage(pOld)
			}
		}
	}

	session.SetPhase("LOAD_CONTAINER")

	filecryptPage, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		safeCloseBrowser(session)
		return "BROWSER_CLOSED"
	}
	attachInitScriptToPage(filecryptPage)

	logMsg(fmt.Sprintf("Opening Container: %s", containerURL), "INFO", cfg)
	navCtx, navCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if _, err := (proto.PageNavigate{URL: containerURL}).Call(filecryptPage.Context(navCtx)); err != nil {
		logMsg(fmt.Sprintf("Initial navigation note: %v", err), "DEBUG", cfg)
	}
	navCancel()

	activeContainerPage, waitErr := waitForTableOrCaptcha(b, keepalivePage, cfg)
	if activeContainerPage == nil {
		if errors.Is(waitErr, ErrUserCancelled) {
			if pages, err := b.Pages(); err == nil {
				for _, pOther := range pages {
					if pOther.TargetID != keepalivePage.TargetID {
						safeClosePage(pOther)
					}
				}
			}
			return "CANCELLED_TO_HOME"
		}
		safeCloseBrowser(session)
		return "BROWSER_CLOSED"
	}

	if pages, err := b.Pages(); err == nil {
		for _, pOther := range pages {
			if pOther.TargetID != activeContainerPage.TargetID && pOther.TargetID != keepalivePage.TargetID {
				safeClosePage(pOther)
			}
		}
	}

	info, err := freezePageAndExtractContainerInfo(activeContainerPage, cfg)
	if err != nil || info == nil || len(info.Rows) == 0 {
		logMsg("No individual download rows found! (This container may only offer DLC / Click'n'Load).", "ERROR", cfg)
		fmt.Printf("\n  %sPress Enter to return to main menu...%s", AccentAmber, Reset)
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		safeClosePage(activeContainerPage)
		return "NEW_CONTAINER"
	}

	containerTitle := info.Title
	rows := info.Rows

	hostsMap := make(map[string][]RowInfo)
	var hostList []string
	for _, r := range rows {
		if _, exists := hostsMap[r.Host]; !exists {
			hostList = append(hostList, r.Host)
		}
		hostsMap[r.Host] = append(hostsMap[r.Host], r)
	}

	session.mu.Lock()
	session.Phase = "IDLE"
	session.MainContainerID = activeContainerPage.TargetID
	session.mu.Unlock()

	decryptedStatus := make(map[string][2]int)
	decryptedRowURLs := make(map[int]string)
	harvestedCache := make(map[int]string)
	workerHolder := &WorkerHolder{}
	var lastActionBanners []string
	var lastDecryptedBatches []HostBatch

	type hostJob struct {
		Host string
		Rows []RowInfo
	}

	for !isPageClosed(b, activeContainerPage) {
		session.SetPhase("IDLE")

		printAppHeader(
			"App / Container / Host Selection",
			cfg,
			!isDebug(cfg),
			containerTitle,
		)

		renderContainerInfoPanel(containerTitle, len(rows), len(hostsMap), "", 0)

		if len(lastDecryptedBatches) > 0 {
			var batchLines []string
			for bIdx, batch := range lastDecryptedBatches {
				if bIdx > 0 {
					batchLines = append(batchLines, "")
				}
				batchLines = append(batchLines, fmt.Sprintf("%sHost: %s (%d links)%s",
					Bold+AccentLavender, batch.Host, len(batch.Items), Reset))
				totB := len(batch.Items)
				for idxB, item := range batch.Items {
					formatted := formatResolvedItemMarkup(idxB+1, totB, item.Filename, item.URL, item.Success)
					batchLines = append(batchLines, strings.Split(formatted, "\n")...)
				}
			}
			title := Bold + AccentTeal + "Extracted Links (Press [C] to copy all URLs to clipboard)" + Reset
			printPanel(batchLines, title, BorderAccent, 0, 1)
		}

		if len(lastActionBanners) > 0 {
			var bLines []string
			bannerBorder := AccentGreen
			for _, bMsg := range lastActionBanners {
				bLines = append(bLines, "  "+bMsg)
				if strings.Contains(bMsg, AccentCoral) {
					bannerBorder = AccentCoral
				} else if strings.Contains(bMsg, AccentAmber) && bannerBorder != AccentCoral {
					bannerBorder = AccentAmber
				}
			}
			printPanel(bLines, "", bannerBorder, 0, 1)
		}

		cols := []tableCol{
			{Header: "Key", Width: 6, Justify: "center", NoWrap: true},
			{Header: "Host Provider", Width: 22, Justify: "left", NoWrap: true},
			{Header: "Total Links", Width: 13, Justify: "left", NoWrap: true},
			{Header: "Availability", Width: 0, Justify: "left", NoWrap: true},
			{Header: "Decryption Status", Width: 22, Justify: "right", NoWrap: true},
		}

		var tblRows []tableRow
		for idx, h := range hostList {
			hRows := hostsMap[h]
			totalH := len(hRows)
			onlineH := 0
			for _, r := range hRows {
				if r.IsOnline {
					onlineH++
				}
			}
			offlineH := totalH - onlineH

			var healthStr string
			if offlineH == 0 {
				healthStr = fmt.Sprintf("%s● %d Online%s", AccentGreen, onlineH, Reset)
			} else if onlineH == 0 {
				healthStr = fmt.Sprintf("%s○ %d Offline%s", AccentCoral, offlineH, Reset)
			} else {
				healthStr = fmt.Sprintf("%s● %d Online%s %s/%s %s%d Offline%s",
					AccentGreen, onlineH, Reset, TextMuted, Reset, AccentCoral, offlineH, Reset)
			}

			statusBadge := TextMuted + "Pending" + Reset
			if st, ok := decryptedStatus[h]; ok {
				if st[0] >= st[1] {
					statusBadge = fmt.Sprintf("%s✓ Decrypted (%d/%d)%s", Bold+AccentGreen, st[0], st[1], Reset)
				} else {
					statusBadge = fmt.Sprintf("%s◐ Partial (%d/%d)%s", Bold+AccentAmber, st[0], st[1], Reset)
				}
			}

			hostStyle := TextMuted
			if onlineH > 0 {
				hostStyle = Bold + TextMain
			}

			tblRows = append(tblRows, tableRow{Cells: []string{
				fmt.Sprintf("%s[%d]%s", Bold+AccentTeal, idx+1, Reset),
				hostStyle + h + Reset,
				fmt.Sprintf("%s%d links%s", TextMain, totalH, Reset),
				healthStr,
				statusBadge,
			}})
		}

		allDone := len(hostList) > 0
		for _, h := range hostList {
			st, ok := decryptedStatus[h]
			if !ok || st[0] < st[1] {
				allDone = false
				break
			}
		}
		allDoneBadge := ""
		if allDone {
			allDoneBadge = Bold + AccentGreen + "✓ All Complete" + Reset
		}

		tblRows = append(tblRows, tableRow{IsSection: true})
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentGreen + "[A]" + Reset,
			Bold + TextMain + "Decrypt All Hosts" + Reset,
			fmt.Sprintf("%s%d links%s", TextMuted, len(rows), Reset),
			"",
			allDoneBadge,
		}})
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentLavender + "[L]" + Reset,
			TextMain + "Select Specific Links" + Reset,
			fmt.Sprintf("%s%d links%s", TextMuted, len(rows), Reset),
			TextMuted + "Pick specific files (e.g. 1, 3, 4-6)" + Reset,
			"",
		}})
		if len(lastDecryptedBatches) > 0 {
			tblRows = append(tblRows, tableRow{Cells: []string{
				Bold + AccentLavender + "[C]" + Reset,
				TextMain + "Copy Shown URLs" + Reset,
				"",
				TextMuted + "Copy extracted links to Clipboard" + Reset,
				"",
			}})
		}
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentTeal + "[N]" + Reset,
			TextMain + "New Container URL" + Reset,
			"",
			TextMuted + "Keeps session alive" + Reset,
			"",
		}})
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentAmber + "[O]" + Reset,
			TextMain + "Open Output Folder" + Reset,
			"",
			TextMuted + "View saved .txt files" + Reset,
			"",
		}})
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentMauve + "[S]" + Reset,
			TextMain + "Application Settings" + Reset,
			"",
			TextMuted + "Browser, Debug, Save/Copy" + Reset,
			"",
		}})
		tblRows = append(tblRows, tableRow{Cells: []string{
			Bold + AccentCoral + "[Q]" + Reset,
			TextMain + "Quit Application" + Reset,
			"",
			"",
			"",
		}})

		printPanel(renderTable(cols, tblRows, false, UIWidth-4), "", BorderSubtle, 0, 1)

		validKeys := []string{"A", "L", "N", "O", "S", "Q"}
		promptKeys := fmt.Sprintf("1-%d, A, L, N, O, S, Q", len(hostList))
		if len(lastDecryptedBatches) > 0 {
			validKeys = append(validKeys, "C")
			promptKeys = fmt.Sprintf("1-%d, A, L, C, N, O, S, Q", len(hostList))
		}

		choice := waitForSingleKey(
			fmt.Sprintf("\n  %sPress key [%s]: %s", AccentTeal, promptKeys, Reset),
			validKeys,
			len(hostList),
			func() bool { return isPageClosed(b, activeContainerPage) },
			false,
		)

		if choice == "" || isPageClosed(b, activeContainerPage) {
			safeCloseBrowser(session)
			return "BROWSER_CLOSED"
		}

		if choice == "Q" {
			safeCloseBrowser(session)
			return "QUIT_APP"
		} else if choice == "N" {
			safeClosePage(workerHolder.Page)
			safeClosePage(activeContainerPage)
			return "NEW_CONTAINER"
		} else if choice == "C" && len(lastDecryptedBatches) > 0 {
			var urlsToCopy []string
			for _, batch := range lastDecryptedBatches {
				for _, item := range batch.Items {
					if item.Success {
						urlsToCopy = append(urlsToCopy, item.URL)
					}
				}
			}
			if len(urlsToCopy) > 0 && copyTextToClipboard(strings.Join(urlsToCopy, "\n")) {
				lastActionBanners = []string{
					fmt.Sprintf("%s✓ Copied %d decrypted URLs to Windows Clipboard!%s", Bold+AccentGreen, len(urlsToCopy), Reset),
				}
			} else {
				lastActionBanners = []string{
					fmt.Sprintf("%s! Could not copy URLs to clipboard.%s", AccentCoral, Reset),
				}
			}
			continue
		} else if choice == "O" {
			saveDir, _ := filepath.Abs(filepath.Join(AppDir, cfg.Get("Output", "save_directory", "./decrypted_links")))
			openFolderInExplorer(saveDir)
			lastActionBanners = []string{
				fmt.Sprintf("%s✓ Opened folder in Explorer:%s %s%s%s", AccentGreen, Reset, AccentTeal, saveDir, Reset),
			}
			continue
		} else if choice == "S" {
			interactiveSettingsMenu(cfg, "App / Container")
			continue
		}

		var jobsToProcess []hostJob
		isCustomLinkSelection := false

		if choice == "L" {
			chosenRows, ok := promptIndividualLinkSelection(
				b,
				activeContainerPage,
				containerTitle,
				hostList,
				hostsMap,
				decryptedRowURLs,
				cfg,
			)
			if isPageClosed(b, activeContainerPage) {
				safeCloseBrowser(session)
				return "BROWSER_CLOSED"
			}
			if !ok || len(chosenRows) == 0 {
				continue
			}
			isCustomLinkSelection = true
			groupedByHost := make(map[string][]RowInfo)
			var selectedHostOrder []string
			for _, r := range chosenRows {
				if _, exists := groupedByHost[r.Host]; !exists {
					selectedHostOrder = append(selectedHostOrder, r.Host)
				}
				groupedByHost[r.Host] = append(groupedByHost[r.Host], r)
			}
			for _, h := range selectedHostOrder {
				jobsToProcess = append(jobsToProcess, hostJob{
					Host: h,
					Rows: groupedByHost[h],
				})
			}
		} else if choice == "A" {
			for _, h := range hostList {
				jobsToProcess = append(jobsToProcess, hostJob{
					Host: h,
					Rows: hostsMap[h],
				})
			}
		} else if idx, err := strconv.Atoi(choice); err == nil && idx >= 1 && idx <= len(hostList) {
			h := hostList[idx-1]
			jobsToProcess = []hostJob{{
				Host: h,
				Rows: hostsMap[h],
			}}
		} else {
			lastActionBanners = []string{
				fmt.Sprintf("%s! Invalid key '%s'. Please try again.%s", AccentAmber, choice, Reset),
			}
			continue
		}

		skipOffline := cfg.GetBool("Behavior", "skip_offline_links", false)
		reqDelay := cfg.GetFloat("Behavior", "request_delay", 0.2)
		copyClip := cfg.GetBool("Output", "copy_to_clipboard", false)
		lastActionBanners = nil
		lastDecryptedBatches = nil
		var allBatchURLsForClipboard []string

		userCancelledBatch := false
		browserClosedDuringBatch := false

		for _, job := range jobsToProcess {
			if userCancelledBatch || browserClosedDuringBatch {
				break
			}
			if isPageClosed(b, activeContainerPage) {
				browserClosedDuringBatch = true
				break
			}

			targetHost := job.Host
			selectedRows := job.Rows

			// Apply skip_offline_links only on bulk host selection (when the user explicitly
			// picked specific links via [L], respect their exact selection).
			if skipOffline && !isCustomLinkSelection {
				var filtered []RowInfo
				for _, r := range selectedRows {
					if r.IsOnline {
						filtered = append(filtered, r)
					}
				}
				selectedRows = filtered
				if len(selectedRows) == 0 {
					lastActionBanners = append(lastActionBanners,
						fmt.Sprintf("%s! Skipped [%s] (all links marked Offline).%s", AccentAmber, targetHost, Reset))
					continue
				}
			}

			var extractedForHost []ExtractedURL
			var displayItemsForHost []DisplayBatchItem

			printAppHeader(
				fmt.Sprintf("App / Container / Decrypting (%s)", targetHost),
				cfg,
				!isDebug(cfg),
				fmt.Sprintf("Decrypting %s (%d links)...", targetHost, len(selectedRows)),
			)
			renderContainerInfoPanel(containerTitle, 0, 0, targetHost, len(selectedRows))
			printCancelTip("Host Selection")
			drainKeyboardBuffer()

			type harvestedItem struct {
				RowIndex int
				Filename string
				Size     string
				LinkURL  string
			}
			var harvestedLinks []harvestedItem

			if !isDebug(cfg) {
				renderProgressBarLine(0, len(selectedRows), 0)
				for idx, row := range selectedRows {
					if checkCancelKeyPressed() {
						userCancelledBatch = true
						break
					}
					if isPageClosed(b, activeContainerPage) {
						browserClosedDuringBatch = true
						break
					}
					// If this row was already fully decrypted earlier in the session,
					// we don't even need to re-harvest its intermediate /Link/ URL.
					if _, alreadyDecrypted := decryptedRowURLs[row.Index]; alreadyDecrypted {
						harvestedLinks = append(harvestedLinks, harvestedItem{row.Index, row.Filename, row.Size, "CACHED"})
						renderProgressBarLine(idx+1, len(selectedRows), idx+1)
						continue
					}
					linkURL, cached := harvestedCache[row.Index]
					if !cached {
						var hErr error
						linkURL, hErr = harvestIntermediateLinkURL(session, b, activeContainerPage, keepalivePage, row, cfg)
						if errors.Is(hErr, ErrUserCancelled) {
							userCancelledBatch = true
							break
						}
						if errors.Is(hErr, ErrBrowserClosed) {
							browserClosedDuringBatch = true
							break
						}
						if linkURL != "" {
							harvestedCache[row.Index] = linkURL
						} else {
							time.Sleep(200 * time.Millisecond)
						}
					}
					if linkURL != "" {
						harvestedLinks = append(harvestedLinks, harvestedItem{row.Index, row.Filename, row.Size, linkURL})
					}
					renderProgressBarLine(idx+1, len(selectedRows), idx+1)
				}
				fmt.Print("\n")
			} else {
				for _, row := range selectedRows {
					if checkCancelKeyPressed() {
						userCancelledBatch = true
						break
					}
					if isPageClosed(b, activeContainerPage) {
						browserClosedDuringBatch = true
						break
					}
					if _, alreadyDecrypted := decryptedRowURLs[row.Index]; alreadyDecrypted {
						harvestedLinks = append(harvestedLinks, harvestedItem{row.Index, row.Filename, row.Size, "CACHED"})
						continue
					}
					linkURL, cached := harvestedCache[row.Index]
					if !cached {
						var hErr error
						linkURL, hErr = harvestIntermediateLinkURL(session, b, activeContainerPage, keepalivePage, row, cfg)
						if errors.Is(hErr, ErrUserCancelled) {
							userCancelledBatch = true
							break
						}
						if errors.Is(hErr, ErrBrowserClosed) {
							browserClosedDuringBatch = true
							break
						}
					}
					if linkURL != "" {
						harvestedCache[row.Index] = linkURL
						harvestedLinks = append(harvestedLinks, harvestedItem{row.Index, row.Filename, row.Size, linkURL})
					}
				}
			}

			if userCancelledBatch {
				lastActionBanners = append(lastActionBanners,
					fmt.Sprintf("%s! Decryption cancelled by user ([Esc] / [X]). Returned to Host Selection.%s", AccentAmber, Reset))
				break
			}
			if browserClosedDuringBatch || isPageClosed(b, activeContainerPage) {
				browserClosedDuringBatch = true
				break
			}

			fmt.Println()
			for i, item := range harvestedLinks {
				if checkCancelKeyPressed() {
					userCancelledBatch = true
					break
				}
				if isPageClosed(b, activeContainerPage) {
					browserClosedDuringBatch = true
					break
				}
				// Reuse already-resolved URL if decrypted earlier in this session
				if cachedFinal, ok := decryptedRowURLs[item.RowIndex]; ok && cachedFinal != "" {
					printResolvedItem(i+1, len(harvestedLinks), item.Filename, cachedFinal, true)
					extractedForHost = append(extractedForHost, ExtractedURL{item.Filename, cachedFinal})
					displayItemsForHost = append(displayItemsForHost, DisplayBatchItem{item.Filename, cachedFinal, true})
					allBatchURLsForClipboard = append(allBatchURLsForClipboard, cachedFinal)
					continue
				}

				setWindowTitle(fmt.Sprintf("[%02d/%02d] Decrypting %s — %s", i+1, len(harvestedLinks), targetHost, containerTitle))
				finalURL, rErr := resolveLinkSmart(
					session,
					b,
					activeContainerPage,
					workerHolder,
					item.LinkURL,
					item.Filename,
					i+1,
					len(harvestedLinks),
					cfg,
				)
				if errors.Is(rErr, ErrUserCancelled) {
					userCancelledBatch = true
					break
				}
				if errors.Is(rErr, ErrBrowserClosed) {
					browserClosedDuringBatch = true
					break
				}
				if finalURL != "" {
					decryptedRowURLs[item.RowIndex] = finalURL
					extractedForHost = append(extractedForHost, ExtractedURL{item.Filename, finalURL})
					displayItemsForHost = append(displayItemsForHost, DisplayBatchItem{item.Filename, finalURL, true})
					allBatchURLsForClipboard = append(allBatchURLsForClipboard, finalURL)
				} else {
					displayItemsForHost = append(displayItemsForHost, DisplayBatchItem{
						item.Filename,
						"[TIMEOUT - Failed to resolve within 20s]",
						false,
					})
				}
				if reqDelay > 0 {
					sleepSteps := int(reqDelay * 20)
					if sleepSteps < 1 {
						sleepSteps = 1
					}
					stepDur := time.Duration(reqDelay*float64(time.Second)) / time.Duration(sleepSteps)
					for s := 0; s < sleepSteps; s++ {
						if checkCancelKeyPressed() {
							userCancelledBatch = true
							break
						}
						time.Sleep(stepDur)
					}
					if userCancelledBatch {
						break
					}
				}
			}

			if len(displayItemsForHost) > 0 {
				lastDecryptedBatches = append(lastDecryptedBatches, HostBatch{
					Host:  targetHost,
					Items: displayItemsForHost,
				})
			}

			if len(extractedForHost) > 0 {
				totalHostRows := len(hostsMap[targetHost])
				// Build cumulative decrypted URLs for this host in container order so partial
				// selections via [L] merge into the host's .txt file instead of overwriting it.
				var cumulativeForHost []ExtractedURL
				for idxRow, r := range hostsMap[targetHost] {
					if u, ok := decryptedRowURLs[r.Index]; ok && u != "" {
						cumulativeForHost = append(cumulativeForHost, ExtractedURL{r.Filename, u})
						hostsMap[targetHost][idxRow].IsOnline = true
					}
				}
				bannerMsg := saveHostURLs(targetHost, containerTitle, containerURL, cumulativeForHost, cfg)
				if bannerMsg != "" {
					lastActionBanners = append(lastActionBanners, bannerMsg)
				}
				decryptedStatus[targetHost] = [2]int{len(cumulativeForHost), totalHostRows}
			} else if !userCancelledBatch && !browserClosedDuringBatch {
				lastActionBanners = append(lastActionBanners,
					fmt.Sprintf("%s! No final URLs were extracted for [%s].%s", AccentCoral, targetHost, Reset))
			}

			if userCancelledBatch {
				lastActionBanners = append(lastActionBanners,
					fmt.Sprintf("%s! Decryption cancelled by user ([Esc] / [X]). Returned to Host Selection.%s", AccentAmber, Reset))
				break
			}
		}

		// Perform automatic clipboard copy once across the entire batch so multi-host selections
		// via [L] or [A] copy all extracted URLs together instead of overwriting per host.
		if copyClip && len(allBatchURLsForClipboard) > 0 {
			if copyTextToClipboard(strings.Join(allBatchURLsForClipboard, "\n")) {
				lastActionBanners = append(lastActionBanners,
					fmt.Sprintf("%s✓ Auto-copied %d decrypted URL(s) to Windows Clipboard!%s", Bold+AccentMauve, len(allBatchURLsForClipboard), Reset))
			}
		}

		if workerHolder.Page != nil {
			safeClosePage(workerHolder.Page)
			workerHolder.Page = nil
		}

		if browserClosedDuringBatch {
			break
		}
	}

	safeCloseBrowser(session)
	return "BROWSER_CLOSED"
}

// ==============================================================================
// MAIN APPLICATION ENTRYPOINT
// ==============================================================================

func main() {
	initWindowsConsole()

	exePath, err := os.Executable()
	if err == nil {
		AppDir = filepath.Dir(exePath)
	} else {
		AppDir, _ = os.Getwd()
	}
	if strings.Contains(strings.ToLower(AppDir), "go-build") {
		AppDir, _ = os.Getwd()
	}
	ConfigPath = filepath.Join(AppDir, "config.ini")

	sessionHolder := &SessionHolder{
		Phase: "LOAD_CONTAINER",
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		safeCloseBrowser(sessionHolder)
		fmt.Printf("\n  %sExited cleanly.%s\n", AccentAmber, Reset)
		os.Exit(0)
	}()
	defer safeCloseBrowser(sessionHolder)

	homeWarning := ""

	for {
		cfg := validateAndLoadConfig()
		printAppHeader("App / Home", cfg, !isDebug(cfg), "Home")

		if homeWarning != "" {
			printPanel([]string{"  " + homeWarning}, "", AccentAmber, 0, 1)
			homeWarning = ""
		}

		renderSettingsPreviewPanel(cfg)

		var sessState string
		if isBrowserAlive(sessionHolder) {
			sessState = Bold + AccentGreen + "● Active" + Reset + " " + TextMuted + "(Browser Open — Solved Captchas Remembered)" + Reset
		} else {
			sessState = TextMuted + "○ Standby (Launches when you enter a Container URL)" + Reset
		}

		homeLines := []string{
			TextMuted + "Browser Session:" + Reset + "  " + sessState,
			"",
			"  • Paste a " + Bold + AccentTeal + "Filecrypt Container URL" + Reset + " below and press " + Bold + AccentGreen + "Enter" + Reset,
			"  • Quick Commands:  " + Bold + AccentMauve + "[S]" + Reset + " Settings    " +
				Bold + AccentAmber + "[O]" + Reset + " Open Saved Folder    " +
				Bold + AccentLavender + "[I]" + Reset + " About / License    " +
				Bold + AccentCoral + "[Q]" + Reset + " Quit App",
		}
		printPanel(homeLines, "", BorderSubtle, 1, 2)

		userInp, ok := readTextWithEsc(
			fmt.Sprintf("\n  %sContainer URL or Command [S / O / I / Q]: %s", AccentTeal, Reset),
		)
		if !ok || userInp == "" {
			continue
		}

		cmd := strings.ToUpper(userInp)
		if cmd == "Q" {
			break
		} else if cmd == "S" {
			interactiveSettingsMenu(cfg, "App")
			continue
		} else if cmd == "I" {
			showAboutLicenseScreen(cfg)
			continue
		} else if cmd == "O" {
			saveDir, _ := filepath.Abs(filepath.Join(AppDir, cfg.Get("Output", "save_directory", "./decrypted_links")))
			openFolderInExplorer(saveDir)
			continue
		}

		if !strings.HasPrefix(strings.ToLower(userInp), "http://") && !strings.HasPrefix(strings.ToLower(userInp), "https://") {
			userInp = "https://" + userInp
		}

		parsedInp, err := url.Parse(userInp)
		if err != nil || !strings.Contains(parsedInp.Host, ".") || len(parsedInp.Host) < 4 {
			homeWarning = fmt.Sprintf("%s! '%s' is not a valid Container URL or command (S, O, I, Q).%s", AccentAmber, userInp, Reset)
			continue
		}

		action := processSingleContainer(userInp, cfg, sessionHolder)
		if action == "QUIT_APP" {
			break
		} else if action == "CANCELLED_TO_HOME" {
			homeWarning = fmt.Sprintf("%s! Container loading cancelled by user ([Esc] / [X]).%s", AccentAmber, Reset)
		} else if action == "BROWSER_CLOSED" {
			homeWarning = fmt.Sprintf("%s! Browser tab or window was closed. Returned to Home.%s", AccentAmber, Reset)
		}
	}
}
