# MSIX package assets

Required PNG files under `Assets/`:

| File | Size | Purpose |
|---|---|---|
| `StoreLogo.png` | 50×50 | Store listing logo |
| `Square44x44Logo.png` | 44×44 | App list, taskbar |
| `SmallTile.png` (Square71x71) | 71×71 | Small tile |
| `Square150x150Logo.png` | 150×150 | Medium tile |
| `Wide310x150Logo.png` | 310×150 | Wide tile |
| `LargeTile.png` (Square310x310) | 310×310 | Large tile |

Transparent-background PNGs. Source: re-export from `build/appicon.png`.

Regenerate locally:

```bash
cd build/windows/msix/Assets
SRC=../../../appicon.png
magick "$SRC" -resize 50x50    -gravity center -background none -extent 50x50    StoreLogo.png
magick "$SRC" -resize 44x44    -gravity center -background none -extent 44x44    Square44x44Logo.png
magick "$SRC" -resize 71x71    -gravity center -background none -extent 71x71    SmallTile.png
magick "$SRC" -resize 150x150  -gravity center -background none -extent 150x150  Square150x150Logo.png
magick "$SRC" -resize 310x310  -gravity center -background none -extent 310x310  LargeTile.png
magick "$SRC" -resize 150x150  -gravity west   -background none -extent 310x150  Wide310x150Logo.png
```

Store listing assets (NOT in MSIX, uploaded directly in Partner Center):

- 300×300 square logo
- 2480×1200 hero image
- Up to 9 screenshots at 1366×768 or larger
