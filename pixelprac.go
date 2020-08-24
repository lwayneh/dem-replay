package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/golang/freetype/truetype"
	ocom "github.com/lwayneh/dem-replay/common"
	"github.com/lwayneh/dem-replay/match"
	game "github.com/lwayneh/dem-replay/match"
	part "github.com/lwayneh/dem-replay/particle"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
)

var (
	paused          bool
	loadCtrl        bool
	staticCtrl      bool = false
	curFrame        int
	spritePath      = "imgSprite.png"
	lastEffect      = 0
	ctrlList        = []string{"Play", "Pause", "Rewind", "FastForward", "barsHorizontal"}
	controls        = make(map[string]*ocom.Control)
	speed           int
	playBar         pixel.Rect
	playPauseCtrl   *ocom.Control
	rewindCtrl      *ocom.Control
	fastForwardCtrl *ocom.Control
	menuCtrl        *ocom.Control

	hover = ocom.Status{
		Name: "Hover",
		Id:   1,
	}

	selected = ocom.Status{
		Name: "Selected",
		Id:   2,
	}
	none = ocom.Status{
		Name: "None",
		Id:   0,
	}
)

// Config contains information the application requires in order to run
type Config struct {
	// Path to font file (.ttf)
	FontPath string

	// Path to overview directory
	OverviewDir string

	// Fallback GOTV Framerate
	FrameRate float64

	// Fallback Gameserver Tickrate
	TickRate float64
}

// DefaultConfig contains standard parameters for the application.
var DefaultConfig = Config{
	FrameRate: -1,
	TickRate:  -1,
}

const (
	fontName          string  = "DejaVuSans.ttf"
	mapOverviewWidth  int32   = 1024
	mapOverviewHeight int32   = 1024
	mapXOffset        float64 = 300
	mapYOffset        float64 = 0
	ctrlBarHeight     float64 = 45
	infoBarHeight     float64 = 110
)

var conf = DefaultConfig

func init() {
	flag.Float64Var(&conf.FrameRate, "framerate", conf.FrameRate, "Fallback GOTV Framerate")
	flag.Float64Var(&conf.TickRate, "tickrate", conf.TickRate, "Fallback Gameserver Tickrate")
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln("trying to get user home directory:", err)
	}
	defaultFontPath := fmt.Sprintf("%v\\dem-replay\\%v", userHomeDir, fontName)
	defaultOverviewDirectory := fmt.Sprintf("%v\\dem-replay\\", userHomeDir)
	flag.StringVar(&conf.FontPath, "fontpath", defaultFontPath, "Path to font file (.ttf)")
	flag.StringVar(&conf.OverviewDir, "overviewdir", defaultOverviewDirectory, "Path to overview directory")
	flag.Parse()
	speed = 5
}

func run() {
	mouseIn := make(chan bool)

	var demoFileName string

	if len(flag.Args()) < 1 {
		demoFileNameB, err := exec.Command("cmd", "/C", "chooser.bat").CombinedOutput()
		if err != nil {
			fmt.Println("Usage: ./dem-replay [path to demo]")
			panic(err)
		}
		demoPath := string(demoFileNameB)
		newPath := strings.ReplaceAll(demoPath, "\\", "/")
		demoFileName = strings.TrimSpace(newPath)

	} else {
		demoFileName = flag.Args()[0]
	}

	cfg := pixelgl.WindowConfig{
		Title:     "Dem-Replay",
		Bounds:    pixel.R(0, 0, 1400, 900),
		Resizable: true,
		VSync:     true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		errorString := fmt.Sprintf("trying to create new window\n%v", err)
		log.Println(errorString)
		time.Sleep(2 * time.Second)
		panic(err)
	}
	win.SetSmooth(true)

	// Load TTF for custom font
	face, err := loadTTF(conf.FontPath, 52)
	if err != nil {
		errorString := fmt.Sprintf("trying to load TTF file:\n%v", err)
		log.Println(errorString)
		time.Sleep(2 * time.Second)
		panic(err)
	}

	// Display welcome/loading message while demo parses
	atlas := text.NewAtlas(face, text.ASCII)
	txt := text.New(win.Bounds().Center(), atlas)
	txt.LineHeight = atlas.LineHeight() * 1.5
	txtInfo := text.New(win.Bounds().Center(), atlas)
	txtInfo.LineHeight = atlas.LineHeight() * 1.5
	txtScore := text.New(win.Bounds().Center(), atlas)
	txtScore.LineHeight = atlas.LineHeight() * 1.5

	win.Clear(colornames.Black)
	lines := []string{
		"We're processing your demo now!",
		"This will take a moment, please wait...",
	}

	go func() {
		for match.Progress < 100 || math.IsNaN(match.Progress) {
			scaleMat := pixel.IM.Scaled(txt.Dot, .4)
			for _, line := range lines {
				txt.Dot.X -= txt.BoundsOf(line).W() / 2
				fmt.Fprintln(txt, line)
			}
			if !math.IsNaN(match.Progress) {
				txt.Dot.X -= 150
				fmt.Fprintf(txt, "%9.f", match.Progress)
				fmt.Fprintln(txt, "%")
			}
			win.Clear(colornames.Black)
			txt.Draw(win, scaleMat)
			win.Update()
			txt.Clear()
		}
		if match.Progress >= 100 {
			runtime.Goexit()
		}
	}()
	// Load & parse match demo
	match, err := game.NewMatch(demoFileName, conf.FrameRate, conf.TickRate)
	if err != nil {
		errorString := fmt.Sprintf("trying to parse demo file:\n%v", err)
		log.Println(errorString)
		panic(err)
	}

	// Re-align text for killfeed
	txt.Clear()

	parts := make(map[string]*part.Particles)
	batches := make(map[string]*pixel.Batch)

	// Load particle sprites
	smokeSheet, smokeRects, err := part.LoadSpriteSheet("blackSmoke.png", "blackSmoke.csv")
	if err != nil {
		panic(err)
	}

	ss := &part.ParticleSystem{
		Rects:    smokeRects,
		Orig:     pixel.ZV,
		VelBasis: []pixel.Vec{pixel.V(-50, 0), pixel.V(20, 100), pixel.V(50, 0)},
		VelDist:  .1,
		LifeAvg:  3,
		LifeDist: .1,
	}

	sp := &part.Particles{
		Generate:  ss.Generate,
		Update:    ss.Update,
		SpawnAvg:  .3,
		SpawnDist: .1,
	}

	parts["smoke"] = sp
	// Load sprite csv data
	sBatch := pixel.NewBatch(&pixel.TrianglesData{}, smokeSheet)
	batches["smoke"] = sBatch

	expSheet, expRects, err := part.LoadSpriteSheet("explosion.png", "explosion.csv")
	if err != nil {
		panic(err)
	}
	heS := &part.ParticleSystem{
		Rects:      expRects,
		Orig:       pixel.ZV,
		VelBasis:   []pixel.Vec{pixel.V(-200, 100), pixel.V(0, 100), pixel.V(200, 100)},
		VelDist:    .05,
		LifeAvg:    1,
		LifeDist:   0.2,
		StartColor: pixel.RGBA{247, 203, 43, 1},
		ColorVar:   .2,
	}

	heP := &part.Particles{
		Generate:  heS.Generate,
		Update:    heS.Update,
		SpawnAvg:  0.3,
		SpawnDist: .1,
	}
	parts["he"] = heP

	// Load sprite csv data
	heBatch := pixel.NewBatch(&pixel.TrianglesData{}, expSheet)
	batches["he"] = heBatch

	// Load particle sprites
	flashSheet, flashRects, err := part.LoadSpriteSheet("flash.png", "flash.csv")
	if err != nil {
		panic(err)
	}

	fs := &part.ParticleSystem{
		Rects:      flashRects,
		Orig:       pixel.ZV,
		VelBasis:   []pixel.Vec{pixel.V(-100, -10), pixel.V(0, 0), pixel.V(100, 10)},
		VelDist:    .7,
		LifeAvg:    2,
		LifeDist:   1,
		StartColor: pixel.RGBA{255, 255, 255, 1},
		ColorVar:   .2,
	}

	fp := &part.Particles{
		Generate:  fs.Generate,
		Update:    fs.Update,
		SpawnAvg:  .3,
		SpawnDist: .02,
	}
	parts["flash"] = fp
	// Load sprite csv data
	fBatch := pixel.NewBatch(&pixel.TrianglesData{}, flashSheet)
	batches["flash"] = fBatch

	// Load particle sprites
	fireSheet, fireRects, err := part.LoadSpriteSheet("fire.png", "fire.csv")
	if err != nil {
		panic(err)
	}

	fires := &part.ParticleSystem{
		Rects:      fireRects,
		Orig:       pixel.ZV,
		VelBasis:   []pixel.Vec{pixel.V(-200, 0), pixel.V(0, -200), pixel.V(200, 0)},
		VelDist:    .02,
		LifeAvg:    3,
		LifeDist:   1,
		StartColor: pixel.RGBA{247, 203, 43, 1},
		ColorVar:   .2,
	}

	firep := &part.Particles{
		Generate:  fires.Generate,
		Update:    fires.Update,
		SpawnAvg:  .3,
		SpawnDist: .05,
	}
	parts["fire"] = firep
	// Load sprite csv data
	fireBatch := pixel.NewBatch(&pixel.TrianglesData{}, fireSheet)
	batches["fire"] = fireBatch

	// Load map overview .jpg
	mapOverview, err := loadPicture(filepath.Join(conf.OverviewDir, fmt.Sprintf("%v.jpg", match.MapName)))
	if err != nil {
		errorString := fmt.Sprintf("Error loading image .jpg \n%v", err)
		log.Println(errorString)
		time.Sleep(2 * time.Second)
		panic(err)
	}

	ctrlSheet, ctrlRects, err := part.LoadSpriteSheetAsMap("controls.png", "controls.csv")
	if err != nil {
		errorString := fmt.Sprintf("Error loading image .png \n%v", err)
		fmt.Printf(errorString)
	}
	ctrlSprites := makeSprites(ctrlSheet, ctrlRects)

	infoImg, infoRects, err := part.LoadSpriteSheetAsMap("infoBar.png", "infoBar.csv")
	if err != nil {
		errorString := fmt.Sprintf("Error loading image .png \n%v", err)
		fmt.Printf(errorString)
	}
	infoSprites := makeAllSprites(infoImg, infoRects)

	createControls(ctrlSprites)
	playPauseCtrl = controls["Play"]
	rewindCtrl = controls["Rewind"]
	fastForwardCtrl = controls["FastForward"]
	menuCtrl = controls["barsHorizontal"]

	mapSprite := pixel.NewSprite(mapOverview, mapOverview.Bounds())
	win.Clear(colornames.Black)
	canvas := pixelgl.NewCanvas(pixel.R(0, 0, 1624, 1024))
	canvas.SetSmooth(true)
	controlBounds := pixel.R(0, 0, win.Bounds().W(), ctrlBarHeight)
	controlCanvas := pixelgl.NewCanvas(controlBounds)
	controlCanvas.SetSmooth(true)

	// Create IMDraw Object for drawing shapes
	imd := imdraw.New(nil)
	imd.Precision = 64

	// BEGIN MAIN GAME LOOP//
	last := time.Now()
	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		for _, v := range parts {
			v.UpdateAll(dt)
		}

		canvas.Clear(colornames.Black)

		imd.Clear()
		win.Clear(colornames.Black)

		frameStart := time.Now()
		handleInputs(win, match)
		go checkMouse(win, controlCanvas, mouseIn, speed, match)

		if paused {
			time.Sleep(32)
			updateGraphics(match, win, txt, canvas, mapSprite, imd, parts, batches, &dt, controlCanvas, ctrlSprites, mouseIn, infoSprites, txtInfo, txtScore)
			//updateWindowTitle(window, match)
			continue
		}

		updateGraphics(match, win, txt, canvas, mapSprite, imd, parts, batches, &dt, controlCanvas, ctrlSprites, mouseIn, infoSprites, txtInfo, txtScore)
		canvas.Clear(color.Alpha{0})
		//updateWindowTitle(window, match)

		var playbackSpeed float64 = 1
		// frameDuration is in ms
		frameDuration := float64(time.Since(frameStart) / 1000000)

		if win.Pressed(pixelgl.KeyW) {
			if win.Pressed(pixelgl.KeyLeftShift) {

				playbackSpeed = 10
				curFrame += 10
			} else {
				playbackSpeed = 5
				curFrame += 10
			}
		}
		if win.Pressed(pixelgl.KeyS) {
			if win.Pressed(pixelgl.KeyLeftShift) {
				playbackSpeed = .1
				curFrame -= 10
			} else {
				playbackSpeed = 0.5
				curFrame -= 5
			}
		}
		delay := (1/playbackSpeed)*(1000/match.FrameRate) - frameDuration
		if delay < 0 {
			delay = 0
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if curFrame < len(match.States)-1 {
			curFrame++
		}

	}
}

func main() {

	pixelgl.Run(run)

}

// Helper for loading TTF font files
func loadTTF(path string, size float64) (font.Face, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	font, err := truetype.Parse(bytes)
	if err != nil {
		return nil, err
	}

	return truetype.NewFace(font, &truetype.Options{
		Size:              size,
		GlyphCacheEntries: 1,
	}), nil
}

// Helper for loading pictures (.png or .jpg)
func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

// Handles all keyboard inputs
func handleInputs(win *pixelgl.Window, match *game.Match) {
	if win.Pressed(pixelgl.KeySpace) {
		resume()
	}

	if win.Pressed(pixelgl.KeyA) {
		if win.Pressed(pixelgl.KeyLeftShift) {
			if curFrame < match.FrameRateRounded*10 {
				curFrame = 0
			} else {
				curFrame -= match.FrameRateRounded * 10
			}
		} else {
			if curFrame < match.FrameRateRounded*5 {
				curFrame = 0
			} else {
				curFrame -= match.FrameRateRounded * 5
			}
		}
	}

	if win.Pressed(pixelgl.KeyD) {
		if win.Pressed(pixelgl.KeyLeftShift) {
			if curFrame+match.FrameRateRounded*10 > len(match.States)-1 {
				curFrame = len(match.States) - 1
			} else {
				curFrame += match.FrameRateRounded * 10
			}
		} else {
			if curFrame+match.FrameRateRounded*5 > len(match.States)-1 {
				curFrame = len(match.States) - 1
			} else {
				curFrame += match.FrameRateRounded * 5
			}
		}
	}

	if win.Pressed(pixelgl.KeyQ) {
		if win.Pressed(pixelgl.KeyLeftShift) {
			set := false
			for i, frame := range match.HalfStarts {
				if curFrame < frame {
					if i > 1 && curFrame < match.HalfStarts[i-1]+match.FrameRateRounded/2 {
						curFrame = match.HalfStarts[i-2]
						set = true
						break
					}
					if i-1 < 0 {
						curFrame = 0
						set = true
						break
					}
					curFrame = match.HalfStarts[i-1]
					set = true
					break
				}
			}
			// not set -> last round of match
			if !set {
				if len(match.HalfStarts) > 1 && curFrame < match.HalfStarts[len(match.HalfStarts)-1]+match.FrameRateRounded/2 {
					curFrame = match.HalfStarts[len(match.HalfStarts)-2]
				} else {
					curFrame = match.HalfStarts[len(match.HalfStarts)-1]
				}
			}
		} else {
			set := false
			for i, frame := range match.RoundStarts {
				if curFrame < frame {
					if i > 1 && curFrame < match.RoundStarts[i-1]+match.FrameRateRounded/2 {
						curFrame = match.RoundStarts[i-2]
						set = true
						break
					}
					if i-1 < 0 {
						curFrame = 0
						set = true
						break
					}
					curFrame = match.RoundStarts[i-1]
					set = true
					break
				}
			}
			// not set -> last round of match
			if !set {
				if len(match.RoundStarts) > 1 && curFrame < match.RoundStarts[len(match.RoundStarts)-1]+match.FrameRateRounded/2 {
					curFrame = match.RoundStarts[len(match.RoundStarts)-2]
				} else {
					curFrame = match.RoundStarts[len(match.RoundStarts)-1]
				}
			}
		}
	}

	if win.Pressed(pixelgl.KeyE) {
		if win.Pressed(pixelgl.KeyLeftShift) {
			for _, frame := range match.HalfStarts {
				if curFrame < frame {
					curFrame = frame
					break
				}
			}
		} else {
			for _, frame := range match.RoundStarts {
				if curFrame < frame {
					curFrame = frame
					break
				}
			}
		}
	}

}

// Updates all graphics each frame
func updateGraphics(match *game.Match, win *pixelgl.Window, txt *text.Text, canvas *pixelgl.Canvas,
	mapSprite *pixel.Sprite, imd *imdraw.IMDraw, parts map[string]*part.Particles,
	batches map[string]*pixel.Batch, dt *float64, control *pixelgl.Canvas,
	sprites map[string]*pixel.Sprite, result <-chan bool, infoSprites map[string]*pixel.Sprite, txtInfo *text.Text, txtScore *text.Text) {
	playerNames = make(map[string]string)
	canvas.Clear(colornames.Black)
	mapSprite.Draw(canvas, pixel.IM.Moved(canvas.Bounds().Center()))

	resizeScale := math.Min(
		win.Bounds().W()/canvas.Bounds().W(),
		win.Bounds().H()/canvas.Bounds().H())
	control.SetBounds(pixel.R(0, 0, win.Bounds().W(), ctrlBarHeight))

	mainMat := pixel.IM.Scaled(pixel.ZV, resizeScale)
	win.SetMatrix(mainMat)
	txt.Clear()
	drawInfoBars(match, canvas, infoSprites, txtInfo)
	drawKills(match, infoSprites, txt, canvas, mapSprite)
	drawScore(match, txtInfo, canvas)
	shots := match.Shots[curFrame]
	for _, shot := range shots {
		drawShot(imd, canvas, &shot, match)
	}

	players := match.States[curFrame].Players
	for _, player := range players {
		txt.Clear()
		drawPlayer(imd, canvas, &player, match, txt, &mainMat)
	}

	effects := match.GrenadeEffects[curFrame]
	for _, effect := range effects {
		drawGrenadeEffect(&effect, match, parts, batches, canvas, dt)
		lastEffect = effect.GrenadeEntityID
	}

	grenades := match.States[curFrame].Grenades
	for _, grenade := range grenades {
		drawGrenade(imd, &grenade, match)
	}

	infernos := match.States[curFrame].Infernos
	for _, inferno := range infernos {
		drawInferno(imd, &inferno, match, parts, batches, canvas, dt)
	}
	bomb := match.States[curFrame].Bomb
	drawBomb(infoSprites, &bomb, match, canvas)

	select {
	case loadCtrl = <-result:
		if loadCtrl {
			imd.Draw(canvas)
			canvas.Draw(win, pixel.IM.Moved(canvas.Bounds().Center()))
			control.Clear(color.RGBA{85, 90, 99, 90})
			drawControls(control, match, win, sprites)
			drawFrameBar(control, match, txt)
			control.Draw(win, pixel.IM.Moved(control.Bounds().Center()))
			canvas.Clear(colornames.Black)
		} else {
			imd.Draw(canvas)
			canvas.Draw(win, pixel.IM.Moved(canvas.Bounds().Center()))
			canvas.Clear(colornames.Black)
		}
	default:
		{
			imd.Draw(canvas)
			canvas.Draw(win, pixel.IM.Moved(canvas.Bounds().Center()))
			canvas.Clear(colornames.Black)
		}
	}

	imd.Clear()
	win.Update()

}

func checkMouse(win *pixelgl.Window, controlCanvas *pixelgl.Canvas, mouseIn chan bool, speed int, match *game.Match) {
	controls["FastForward"].Status = none
	controls["Rewind"].Status = none
	mouse1 := pixelgl.MouseButton1
	if win.MouseInsideWindow() {
		mousePos := win.MousePosition()
		if controlCanvas.Bounds().Contains(mousePos) {
			loadCtrl = true
			if win.Pressed(mouse1) {
				mousePress(mousePos, speed, match, controlCanvas)
			} else if win.JustReleased(mouse1) {
				mouseClicks(mousePos, match, controlCanvas)
			}
		} else {
			if !staticCtrl {
				loadCtrl = false
			}
		}

	} else {
		if !staticCtrl {
			loadCtrl = false
		}
	}
	mouseIn <- loadCtrl

}

func makeSprites(sheet pixel.Picture, rects map[string]pixel.Rect) map[string]*pixel.Sprite {
	sprites := make(map[string]*pixel.Sprite)
	for _, s := range ctrlList {
		name := s
		rect := rects[name]
		newSprite := pixel.NewSprite(sheet, rect)
		sprites[name] = newSprite
	}
	return sprites
}
func makeAllSprites(sheet pixel.Picture, rects map[string]pixel.Rect) map[string]*pixel.Sprite {
	sprites := make(map[string]*pixel.Sprite)
	for n, s := range rects {
		name := n
		rect := s
		newSprite := pixel.NewSprite(sheet, rect)
		sprites[name] = newSprite
	}
	return sprites
}

func createControls(ctrls map[string]*pixel.Sprite) {
	for n, c := range ctrls {
		name := n
		if name != "Pause" {

			ctrl := ocom.Control{
				Name:   name,
				Rect:   c.Frame(),
				Offset: pixel.ZV,
				Scale:  .3,
				Sprite: c,
				Status: none,
			}
			controls[name] = &ctrl
		}

	}

}

func resume() {
	paused = !paused
	play := controls["Play"]
	play.Status = selected
}

func fastForward(speed int, match *game.Match) {
	if curFrame+match.FrameRateRounded+speed > len(match.States)-1 {
		curFrame = len(match.States) - 1
	} else {
		curFrame += match.FrameRateRounded + speed
	}

}

func rewind(speed int, match *game.Match) {
	if curFrame < match.FrameRateRounded+speed {
		curFrame = 0
	} else {
		curFrame -= match.FrameRateRounded + speed
	}
}

func mousePress(mousePos pixel.Vec, speed int, match *game.Match, canvas *pixelgl.Canvas) {
	rewindPos := ctrlPosRect(canvas, rewindCtrl)
	fastforwardPos := ctrlPosRect(canvas, fastForwardCtrl)
	if rewindPos.Contains(mousePos) {
		controls["Rewind"].Status = selected
		rewind(speed, match)
	}
	if fastforwardPos.Contains(mousePos) {
		controls["FastForward"].Status = selected
		fastForward(speed, match)
	}
}

func mouseClicks(mousePos pixel.Vec, game *game.Match, canvas *pixelgl.Canvas) {
	if playBar.Contains(mousePos) {
		totalFramesPerc := float64(match.TotalFrames) / 100
		newFramePerc := mousePos.X / (playBar.W() / 100)
		newFrame := newFramePerc * totalFramesPerc
		curFrame = int(newFrame)
	}
	playPos := ctrlPosRect(canvas, playPauseCtrl)
	menuPos := ctrlPosRect(canvas, menuCtrl)
	//"Play", "Pause", "Rewind", "FastForward", "barsHorizontal"
	if playPos.Contains(mousePos) {
		resume()
	}
	if menuPos.Contains(mousePos) {
		controls["barsHorizontal"].Status = selected
		ctrlToggle()
	}
}

func ctrlPosRect(canvas *pixelgl.Canvas, ctrl *ocom.Control) pixel.Rect {
	center := canvas.Bounds().Center()
	ctrlCenter := center.Add(ctrl.Offset)
	minX := ctrlCenter.X - ((ctrl.Rect.W() / 2) * ctrl.Scale)
	minY := ctrlCenter.Y - ((ctrl.Rect.H() / 2) * ctrl.Scale)
	maxX := ctrlCenter.X + ((ctrl.Rect.W() / 2) * ctrl.Scale)
	maxY := ctrlCenter.Y + ((ctrl.Rect.H() / 2) * ctrl.Scale)
	return pixel.R(minX, minY, maxX, maxY)
}

func ctrlToggle() {
	ctrl := controls["barsHorizontal"]
	if !staticCtrl {
		loadCtrl = true
		staticCtrl = true
		ctrl.Status = selected

	} else {
		staticCtrl = false
		ctrl.Status = none
	}

}
