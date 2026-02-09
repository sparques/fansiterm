Alright Sparques, let’s turn your lovingly hand-rolled escape zoo into something a future human (or AI archaeologist) can actually use without spelunking the source.

Below is **Markdown-ready documentation** for your **fansiterm private escape sequences**, derived directly from the implementation .
I’ve been pedantic where it matters, explicit about optional arguments, and consistent with the parser’s actual behavior—not what one might *hope* it does.

---

# fansiterm Private Escape Sequences

fansiterm defines a set of **non-standard private escape sequences** prefixed with:

```
ESC /
```

General form:

```
ESC / <mnemonic><arguments> BEL
```

or for image payloads:

```
ESC / <mnemonic><arguments> ESC \
```

Where:

* `ESC` = `\x1b`
* `BEL` = `\a`
* Arguments are comma-separated integers unless otherwise noted
* Colors may be specified as either:

  * `#RRGGBB`
  * `R,G,B`

Coordinates are **pixel-space**, not cell-space, unless explicitly stated.

---

## 🔔 `a` — Bell / Alert

**Mnemonic:** `a` (analogous to ASCII BEL)

```
ESC / a<optional payload> BEL
```

Calls the terminal’s `BellFunc`, passing the payload string (if any).

### Example

```sh
echo -e "\e/abeepbeep\a"
```

---

## 🖼️ `B` — Blit Image

**Mnemonic:** `B`

Draws an image at a location, optionally scaling to a rectangle.

### Forms

#### 1. At cursor (no scaling)

```
ESC / B<pixdata> ESC \
```

#### 2. At pixel position

```
ESC / Bx,y;<pixdata> ESC \
```

#### 3. Fit image into rectangle

```
ESC / Bx1,y1;x2,y2;<pixdata> ESC \
```

* `pixdata` is base64-encoded image data (PNG/JPEG/etc)
* Supported formats depend upon compile-time flags
* Scaling occurs only in the rectangle form
* Cursor advances horizontally by the image width in cells

### Example

```sh
echo -e "\e/B100,100;$(base64 image.png)\e\\"
```

---

## 🧱 `C` — Cell Image Blit

**Mnemonic:** `C`

Places pixel data *inside the current cell*.

### Forms

#### 1. Full cell

```
ESC / C<pixdata> ESC \
```

#### 2. Offset into image

```
ESC / Cx,y;<pixdata> ESC \
```

If decoding fails, raw RGB pixel data is assumed:

* RGB, 1 byte per channel
* Exactly cell-sized

### Example

```sh
echo -e "\e/C0,0;$(base64 cell.rgb)\e\\"
```

---

## 🎨 `F` — Fill Rectangle

**Mnemonic:** `F`

```
ESC / Fx1,y1;x2,y2;<color> BEL
```

Fills a rectangle with a solid color.

Color forms:

* `#RRGGBB`
* `R,G,B`

### Example

```sh
echo -e "\e/F10,10;200,200;#003366\a"
```

---

## 🔁 `I` — Invert Region

**Mnemonic:** `I`

```
ESC / Ix1,y1;x2,y2 BEL
```

Inverts colors within the specified rectangle.

### Example

```sh
echo -e "\e/I0,0;639,479\a"
```

---

## 📏 `L` — Draw Line

**Mnemonic:** `L`

```
ESC / Lx1,y1;x2,y2 BEL
ESC / Lx1,y1;x2,y2;<color> BEL
```

Uses a Bresenham-style raster algorithm.

* Defaults to active foreground color if none specified

### Example

```sh
echo -e "\e/L10,10;300,200;#FF00FF\a"
```

---

## 🎛️ `P` — Palette Definition (Stub)

**Mnemonic:** `P`

```
ESC / Pa<id>;#RRGGBB BEL
ESC / Pp<id>;#RRGGBB BEL
```

* `a` = ANSI palette (0–15)
* `p` = 256-color palette
* Currently parsed but not applied (NOP)

### Example

```sh
echo -e "\e/Pa1;#FF0000\a"
```

---

## 🔴 `R` — Filled Circle

**Mnemonic:** `R`

```
ESC / Rx,y,r BEL
ESC / Rx,y,r;<color> BEL
```

Draws a **filled circle**.

* Defaults to active foreground color
* Uses brute-force radius test (O(r²), gloriously honest)

### Example (your requested form)

```sh
# draw a red, filled circle at 100,100 with radius 10
echo -e "\e/R100,100,10;#FF0000\a"
```

---

## 🟦 `b` — Box (Outline Rectangle)

**Mnemonic:** `b`

```
ESC / bx1,y1;x2,y2 BEL
ESC / bx1,y1;x2,y2;<color> BEL
```

Draws an **unfilled rectangle**.

### Example

```sh
echo -e "\e/b50,50;150,120;0,255,0\a"
```

---

## ⚪ `r` — Circle Outline

**Mnemonic:** `r`

```
ESC / rx,y,r BEL
ESC / rx,y,r;<color> BEL
```

Uses an 8-way symmetric Bresenham circle algorithm.

### Example

```sh
echo -e "\e/r200,200,40;#FFFFFF\a"
```

---

## 💾 `s` — Save Image as Glyph

**Mnemonic:** `s`

```
ESC / s<char>;<pixdata> BEL
```

Maps an image to a Unicode codepoint in the **user tileset**.

* Creates a full-color tile set on demand
* Enables image-backed glyphs

### Example

```sh
echo -e "\e/s@\;$(base64 glyph.png)\a"
```

---

## 📍 `S` — Set Pixel

**Mnemonic:** `S`

```
ESC / Sx,y BEL
ESC / Sx,y;<color> BEL
```

Sets a single pixel.

* Defaults to active foreground color

### Example

```sh
echo -e "\e/S320,240;255,255,0\a"
```

---

## 🧭 `V` — Vector Scroll

**Mnemonic:** `V`

```
ESC / Vx1,y1;x2,y2;dx,dy BEL
```

Scrolls a region by a vector offset.

### Example

```sh
echo -e "\e/V0,0;639,479;0,-10\a"
```

---

## Closing Notes

* All rectangles are canonicalized (`Min`/`Max` swapped if needed)
* Coordinates are relative to render bounds
* Parsing is strict—**argument count matters**
* This protocol is unashamedly pixel-first and hostile to ambiguity
