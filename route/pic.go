package route

import (
	"image/color"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fogleman/gg"
	dataio "github.com/miosolo/readygo/io"
	"golang.org/x/image/font/basicfont"
)

//DrawRoute see the input checkpoints' position as absolute (x,y), draw route, and export
func DrawRoute(cpList []dataio.Checkpoint) (filePath string, errCode int, err error) {
	fontPath := strings.Join([]string{os.Getenv("GOPATH"), "src", "github.com",
		"miosolo", "readygo", "route", "ARIALBI.TTF"}, string(os.PathSeparator))
	folderPath := strings.Join([]string{os.Getenv("GOPATH"), "src", "github.com",
		"miosolo", "readygo", "archive"}, string(os.PathSeparator))
	os.Mkdir(folderPath, os.ModePerm) // ensure the folder exists
	folderPath = strings.Join([]string{folderPath, "route"}, string(os.PathSeparator))
	os.Mkdir(folderPath, os.ModePerm) // ensure the folder exists
	picFilePath := strings.Join([]string{folderPath, ("routepic-" + time.Now().Format("02-Jan-2006-15-04-05") + ".png")}, string(os.PathSeparator))

	minx, miny, maxx, maxy := math.Inf(+1), math.Inf(+1), math.Inf(-1), math.Inf(-1)
	for _, dot := range cpList {
		if dot.Rx < minx {
			minx = dot.Rx
		}
		if dot.Rx > maxx {
			maxx = dot.Rx
		}
		if dot.Ry < miny {
			miny = dot.Ry
		}
		if dot.Ry > maxy {
			maxy = dot.Ry
		}
	}
	maxx = maxx*1.1 + 1
	maxy = maxy*1.1 + 1
	minx = minx*1.1 - 1
	miny = miny*1.1 - 1

	deltax, deltay := maxx-minx, maxy-miny
	var lx, ly int
	if deltax > deltay { // landscape
		lx, ly = 1600, int(1600*deltay/deltax)
	} else { // portaint
		lx, ly = int(1600*deltax/deltay), 1600
	}
	x0, y0 := float64(lx)*math.Abs(minx/deltax), float64(ly)*math.Abs(maxy/deltay) // base point

	dc := gg.NewContext(lx, ly)

	// snippiets
	arrowFromTo := func(x1, y1, x2, y2, w float64) {
		dc.SetLineWidth(w)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()

		rad := math.Atan((y2 - y1) / (x2 - x1)) // rad in [-Pi/2, Pi/2]
		if x2-x1 < 0 {
			rad += math.Pi
		}
		// 60-degree arrow head
		x3 := x2 + 20*math.Cos(rad+5*math.Pi/6)
		y3 := y2 + 20*math.Sin(rad+5*math.Pi/6)
		x4 := x2 + 20*math.Cos(rad-5*math.Pi/6)
		y4 := y2 + 20*math.Sin(rad-5*math.Pi/6)

		dc.DrawLine(x2, y2, x3, y3)
		dc.Stroke()
		dc.DrawLine(x2, y2, x4, y4)
		dc.Stroke()
	}

	getX := func(c dataio.Checkpoint) (x float64) {
		if c.Rx > 0 {
			x = x0 + (float64(lx)-x0)*c.Rx/maxx
		} else {
			x = x0 - x0*c.Rx/maxx
		}
		return
	}

	getY := func(c dataio.Checkpoint) (y float64) {
		if c.Ry > 0 {
			y = y0 - c.Ry/maxy*y0
		} else {
			y = y0 + (float64(ly)-y0)*c.Ry/miny
		}
		return
	}

	if err := dc.LoadFontFace(fontPath, 25); err != nil {
		log.Println(err)
		dc.SetFontFace(basicfont.Face7x13)
	}
	dc.SetColor(color.White)
	dc.DrawRectangle(0, 0, float64(lx), float64(ly))
	dc.Fill()

	dc.SetColor(color.Black)
	arrowFromTo(x0, float64(ly), x0, 0, 5) // Y axis
	arrowFromTo(0, y0, float64(lx), y0, 5) // X axis
	dc.DrawString("Y", x0+50, 50)
	dc.DrawString("X", float64(lx)-50, y0-50)

	var old, new dataio.Checkpoint
	for i := -1; i < len(cpList)-1; i++ { // i start from -1: let the init point be the first new point
		if i >= 0 {
			old = cpList[i]
		}
		new = cpList[i+1]

		if i >= 0 && (getX(old) != getX(new) || getY(old) != getY(new)) {
			arrowFromTo(getX(old), getY(old), getX(new), getY(new), 3)
		}
		if new.IsPortal {
			w := 20.0
			dc.DrawRectangle(getX(new)-w/2, getY(new)-w/2, w, w)
			dc.Fill()
		} else {
			dc.DrawCircle(getX(new), getY(new), 10)
			dc.Fill()
		}
		dc.DrawString(new.Name+"@"+new.Base, getX(new)+10, getY(new)-5)
	}

	err = dc.SavePNG(picFilePath)
	if err != nil {
		log.Println(err)
		return "", http.StatusInternalServerError, err
	}
	return picFilePath, http.StatusOK, nil
}
