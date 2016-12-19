/*===== Copyright (c) 2016 Yves Beaudoin - All rights reserved - MIT LICENSE (MIT) - Email: webpraxis@gmail.com ================
 *  Package:
 *      mosaic
 *  Overview:
 *      package for creating a mosaic of an image using Truchet tiles.
 *  Function:
 *      Truchet(inFile, outFile string, tileSide uint)
 *         Creates a mosaic of an image using Truchet tiles.
 *  Remarks:
 *      None.
 *  History: v1.0.0 - December 17, 2016 - Original release.
 *============================================================================================================================*/
package mosaic

import(
    "fmt"
    "github.com/nfnt/resize"
    "image"
    "image/color"
    "image/draw"
    "log"
    "math"
    "os"
    "runtime"
    "strings"
    "image/gif"
    "image/jpeg"
    "image/png"
    "path/filepath"
    "time"
)
/*Exported -------------------------------------------------------------------------------------------------------------------*/
func Truchet(inFile, outFile string, tileSide uint) {
/*         Purpose : Creates a mosaic of an image using Truchet tiles.
 *       Arguments : inFile   = path of the image file to be processed.
 *                   outFile  = path for the resulting mosaic file.
 *                   tileSide = side length in pixels for the square area occupied by a tile.
 *         Returns : None.
 * Externals -  In : truchetTiler
 * Externals - Out : None.
 *       Functions : halt, isBackground, readImage, saveImage, tileSelector, updateProgressBar
 *         Remarks : Supported image formats are GIF, JPEG and PNG.
 *         History : v1.0.0 - December 15, 2016 - Original release.
 */
    type avgColor64 struct {
            R, G, B, A uint64
    }
    var(
        start     = time.Now()                        //record start of execution
        tileArea  = uint64(tileSide * tileSide)
        tileCount int
        title     = `Processing "` + inFile + `"`
        x1        float64
        x2        float64
    )
    //Read the source image
    os.Stdout.WriteString("Initializing...")
    imgData := readImage(inFile)
    //Set the dimensions of the mosaic
    imgRect      := imgData.Bounds()
    numCols      := uint(math.Ceil(float64(imgRect.Dx()) / float64(tileSide)))
    numRows      := uint(math.Ceil(float64(imgRect.Dy()) / float64(tileSide)))
    numTiles     := int(numCols * numRows)
    mosaicWidth  := numCols * tileSide
    mosaicHeight := numRows * tileSide
    //Resize the source to the mosaic dimensions
    imgData = resize.Resize(mosaicWidth, mosaicHeight, imgData, resize.Lanczos3)
    imgRect = imgData.Bounds()
    //Convert the resized image to NRGBA64
    mosaic := image.NewNRGBA64(imgRect)
    draw.Draw(mosaic, imgRect, imgData, imgRect.Min, draw.Src)
    os.Stdout.WriteString("\r")
    //Overlay the resized image with Truchet tiles in a top-down, left to right fashion
    for rowTop := imgRect.Min.Y; rowTop < imgRect.Max.Y; rowTop += int(tileSide) {        //repeat for each image row
        rowBottom := rowTop + int(tileSide) - 1
        //define the end y ordinates of the tile's interior boundary
        y1     := float64(rowTop)
        y2     := float64(rowBottom)
        yDelta := y2 - y1
        for colLeft := imgRect.Min.X; colLeft < imgRect.Max.X; colLeft += int(tileSide) { // repeat for each image column
            colRight := colLeft + int(tileSide) - 1
            tileCount++
            updateProgressBar(title, tileCount, numTiles)
            //select a tile shape in the range [0,3]
            tileNo := tileSelector(mosaic, colLeft, colRight, rowTop, rowBottom)
            //define the end x ordinates of the tile's interior boundary
            if tileNo % 2 == 0 {
                x1 = float64(colLeft)
                x2 = float64(colRight)
            } else {
                x1 = float64(colRight)
                x2 = float64(colLeft)
            }
            xDelta := x2 - x1
            //get the average color of the tile and background areas
            bgAvg   := avgColor64{}
            tileAvg := avgColor64{}
            tilePts := uint64(0)
            for y := y1; y <= y2; y++ {                                                   //  repeat for each tile row
                lambda    := (y - y1) / yDelta
                xBoundary := x1 + lambda * xDelta
                xx1       := math.Min(x1, x2)
                xx2       := math.Max(x1, x2)
                for x := xx1; x <= xx2; x++ {                                             //   repeat for each tile column
                    r, g, b, a := mosaic.NRGBA64At(int(x), int(y)).RGBA()
                    bgAvg.R += uint64(r)
                    bgAvg.G += uint64(g)
                    bgAvg.B += uint64(b)
                    bgAvg.A += uint64(a)
                    if isBackground(tileNo, x, xBoundary) { continue }
                    tileAvg.R += uint64(r)
                    tileAvg.G += uint64(g)
                    tileAvg.B += uint64(b)
                    tileAvg.A += uint64(a)
                    tilePts++
                }                                                                         //   until all tile columns processed
            }                                                                             //  until all tile rows processed
            bgAvg.R   /= tileArea
            bgAvg.G   /= tileArea
            bgAvg.B   /= tileArea
            bgAvg.A   /= tileArea
            tileAvg.R /= tilePts
            tileAvg.G /= tilePts
            tileAvg.B /= tilePts
            tileAvg.A /= tilePts
            //overlay the tile
            tileRect  := image.Rect(colLeft, rowTop, colRight + 1, rowBottom + 1)
            bgColor   := color.NRGBA64{uint16(bgAvg.R), uint16(bgAvg.G), uint16(bgAvg.B), uint16(bgAvg.A)}
            tileColor := color.NRGBA64{uint16(tileAvg.R), uint16(tileAvg.G), uint16(tileAvg.B), uint16(tileAvg.A)}
            tiler     := truchetTiler{tileNo, tileColor, tileRect, bgColor, x1, xDelta, y1, yDelta}
            draw.Draw(mosaic, tileRect, &tiler, image.Pt(colLeft, rowTop), draw.Src)
        }                                                                                 // until all image columns processed
    }                                                                                     //until all image rows processed
    //Write the mosaic
    saveImage(outFile, mosaic)
    fmt.Printf(`Mosaic "%s" created in %s` + "\n\n", outFile, time.Since(start))
}
/*Private  -------------------------------------------------------------------------------------------------------------------*/
const _progressBarLen = 50
////File ops
func readImage(file string) (imgData image.Image) {
    fh, err := os.Open(file)
    if err != nil { halt("os.Open - " + err.Error()) }
    imgData, _, err = image.Decode(fh)
    if err != nil { halt("image.Decode - " + err.Error()) }
    fh.Close()
    return
} //end func readImage
func saveImage(file string, imgData image.Image) {
    os.Stdout.WriteString(`Encoding to "` + file + `"...`)
    fh, err := os.Create(file)
    if err != nil { halt("os.Create - " + err.Error()) }
    switch(strings.ToLower(filepath.Ext(file))) {
        case ".gif":
            if err := gif.Encode(fh, imgData, nil); err != nil { halt("gif.Encode - " + err.Error()) }
        case ".jpg", ".jpeg":
            var o jpeg.Options
            o.Quality = 100
            if err := jpeg.Encode(fh, imgData, &o); err != nil { halt("jpeg.Encode - " + err.Error()) }
        case ".png":
            var enc png.Encoder
            enc.CompressionLevel = png.BestCompression
            if err := enc.Encode(fh, imgData); err != nil { halt("png.Encode - " + err.Error()) }
        default:
            halt("Unsupported image format")
    }
    if err := fh.Sync();  err != nil { halt("fh.Sync - " + err.Error()) }
    if err := fh.Close(); err != nil { halt("fh.Close - " + err.Error()) }
    os.Stdout.WriteString("\r")
    return
} //func saveImage
////Reporting
func halt(msg string) {
    pc, _, _, ok := runtime.Caller(1)
    details      := runtime.FuncForPC(pc)
    if ok && details != nil {
        log.Fatalln(fmt.Sprintf("\a%s: %s", details.Name(), msg))
    }
    log.Fatalln("\alsystems: FATAL ERROR!")
} //end func halt
func updateProgressBar(title string, current, total int) {
    //code derived from Graham King's post "Pretty command line / console output on Unix in Python and Go Lang"
    //(http://www.darkcoding.net/software/pretty-command-line-console-output-on-unix-in-python-and-go-lang/)
    prefix := fmt.Sprintf("%s: %d / %d ", title, current, total)
    amount := int(0.1 + float32(_progressBarLen) * float32(current) / float32(total))
    remain := _progressBarLen - amount
    bar    := strings.Repeat("\u2588", amount) + strings.Repeat("\u2591", remain)
    os.Stdout.WriteString(prefix + bar + "\r")
    if current == total { os.Stdout.WriteString(strings.Repeat(" ", len(prefix) + _progressBarLen) + "\r") }
    os.Stdout.Sync()
    return
} //end func updateProgressBar
////Tile ops
func isBackground(tileNo int, x, xBoundary float64) bool {
    return ( tileNo < 2 && x > xBoundary ) || ( tileNo > 1 && x < xBoundary )
} //end func isBackground
func makeMetric(center color.NRGBA64) func(pixel color.NRGBA64) (dist int64) {
    cr, cg, cb, ca := int64(center.R), int64(center.G), int64(center.B), int64(center.A)
    return func(pixel color.NRGBA64) (dist int64) {
            diff := int64(pixel.R) - cr; dist  = diff * diff
            diff  = int64(pixel.G) - cg; dist += diff * diff
            diff  = int64(pixel.B) - cb; dist += diff * diff
            diff  = int64(pixel.A) - ca; dist += diff * diff
            return
           }
} //end func makeMetric
func tileSelector(imgData *image.NRGBA64, colLeft, colRight, rowTop, rowBottom int) (tileNo int) {
    metric := makeMetric(imgData.NRGBA64At(int(float64(colLeft + colRight)/2.), int(float64(rowTop + rowBottom)/2.)))
    //tile 0: center pixel and bottom left one
    worstMetric := metric(imgData.NRGBA64At(colLeft, rowBottom))
    tileNo       = 0
    //tile 1: center pixel and top left one
    dist := metric(imgData.NRGBA64At(colLeft, rowTop))
    if dist > worstMetric { worstMetric = dist; tileNo = 1 }
    //tile 2: center pixel and top right one
    dist  = metric(imgData.NRGBA64At(colRight, rowTop))
    if dist > worstMetric { worstMetric = dist; tileNo = 2 }
    //tile 3: center pixel and bottom right one
    dist  = metric(imgData.NRGBA64At(colRight, rowBottom))
    if dist > worstMetric { tileNo = 3 }
    return
} //end func tileSelector
type truchetTiler struct {
    tileNo    int
    tileColor color.NRGBA64
    tileRect  image.Rectangle
    bgColor   color.NRGBA64
    x1        float64
    xDelta    float64
    y1        float64
    yDelta    float64
}
func (t *truchetTiler) ColorModel() color.Model {
    return color.NRGBA64Model
}
func (t *truchetTiler) Bounds() image.Rectangle {
    return t.tileRect
}
func (t *truchetTiler) At(x, y int) color.Color {
    lambda    := (float64(y) - t.y1) / t.yDelta
    xBoundary := t.x1 + lambda * t.xDelta
    if isBackground(t.tileNo, float64(x), xBoundary) { return t.bgColor }
    return t.tileColor
}
//===== Copyright (c) 2016 Yves Beaudoin - All rights reserved - MIT LICENSE (MIT) - Email: webpraxis@gmail.com ================
//end of Package mosaic
