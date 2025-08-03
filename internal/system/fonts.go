package system

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"vervet/internal/models"

	"github.com/flopp/go-findfont"
	"golang.org/x/image/font/sfnt"
)

type FontService struct {
	logger *slog.Logger
}

func NewFontService(logger *slog.Logger) FontService {
	return FontService{
		logger: logger,
	}
}

func (fs FontService) GetInstalledFonts() []models.Font {
	fontFiles := findfont.List()
	fonts := make([]models.Font, 0, len(fontFiles))

	for _, fontFile := range fontFiles {
		font, err := fs.processFont(fontFile)
		if err != nil {
			fs.logger.Warn("failed to process font", slog.Any("error", err), slog.String("fontFile", fontFile))
			continue
		}

		exists := false
		for _, f := range fonts {
			if f.Family == font.Family {
				exists = true
			}
		}
		if !exists {
			fonts = append(fonts, *font)
		}
	}

	sort.SliceStable(
		fonts,
		func(i, j int) bool {
			return fonts[i].Family < fonts[j].Family
		}, )

	return fonts
}

func (fs FontService) processFont(fontFile string) (*models.Font, error) {
	file, err := os.Open(fontFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open font file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	fontCollection, err := sfnt.ParseCollectionReaderAt(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load font collection: %w", err)
	}

	font, err := fontCollection.Font(0)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	family, err := getFontFamily(font)
	if err != nil {
		return nil, fmt.Errorf("failed to get font family: %w", err)
	}

	isMono := isFixedWidth(font)

	return &models.Font{
		Family:       strings.TrimSpace(family),
		IsFixedWidth: isMono,
	}, nil
}

func isFixedWidth(font *sfnt.Font) bool {
	pt := font.PostTable()
	return pt.IsFixedPitch
}

func getFontFamily(font *sfnt.Font) (string, error) {
	return font.Name(nil, sfnt.NameIDFamily)
}
