package main
import(
    "github.com/ybeaudoin/go-mosaic"
)
func main() {
    mosaic.Truchet("../images/Ada_Lovelace.jpg", "Ada_Lovelace-Truchet64.gif", 64)
    mosaic.Truchet("../images/Ada_Lovelace.jpg", "Ada_Lovelace-Truchet32.png", 32)
    mosaic.Truchet("../images/Ada_Lovelace.jpg", "Ada_Lovelace-Truchet16.jpg", 16)
}
