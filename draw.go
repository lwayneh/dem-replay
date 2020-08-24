package main

import (
	"fmt"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/golang/geo/r3"
	ocom "github.com/lwayneh/dem-replay/common"
	"github.com/lwayneh/dem-replay/match"
	part "github.com/lwayneh/dem-replay/particle"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	meta "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"
	"golang.org/x/image/colornames"
)

const (
	radiusPlayer float64 = 10
	radiusSmoke  float64 = 25
)

var (
	colorTerror  = colornames.Darkorange
	colorCounter = colornames.Dodgerblue
	colorMoney   = colornames.Forestgreen
	rifleLine    = pixel.V(13, 13)
	awpLine      = pixel.V(20, 20)
	shotLength   = pixel.V(1000, 1000)
	halfSecond   = float64(0.5)
	flashed      = time.Duration(halfSecond) * time.Second
	teamOne      ocom.Clan
	teamTwo      ocom.Clan
	playerNames  map[string]string
)

func drawPlayer(imd *imdraw.IMDraw, canvas *pixelgl.Canvas, player *ocom.Player, game *match.Match, txt *text.Text, mat *pixel.Matrix) {
	txt.Clear()
	imd.SetMatrix(pixel.IM)
	flash := imdraw.New(nil)
	teamOne, teamTwo = match.GetTeamTags()
	var color color.RGBA
	if player.Team == common.TeamTerrorists {
		color = colorTerror
	} else {
		color = colorCounter
	}

	if player.Health > 0 {
		pos := player.LastAlivePosition
		viewAngle := degreeToRad(float64(player.ViewDirectionX)) - .7
		imd.Color = color
		exact := position(&pos, game)
		imd.Push(exact)
		imd.Circle(radiusPlayer, 1.5)
		fmt.Fprintln(txt, shortName(player, teamOne, teamTwo))

		txtMat := pixel.IM.Scaled(txt.Dot, .25)
		txtMat = txtMat.Moved(exact.Sub(txt.Dot))
		txt.Draw(canvas, txtMat)

		if player.FlashRemaining > flashed {
			fade := (player.FlashRemaining.Seconds() * 255) / 5.5
			flash.SetColorMask(pixel.Alpha(fade / 255))
			flash.Push(exact)
			flash.Circle(radiusPlayer-1, 0)
			flash.SetMatrix(pixel.IM.Moved(exact))
			flash.Draw(canvas)

		}

		drawWeapon(imd, player, viewAngle, exact, canvas)
		if player.IsDefusing || player.IsPlanting {
			imd.Color = colornames.Turquoise
			imd.Push(exact)
			imd.Circle(radiusPlayer, 0)
		}

		for _, w := range player.Weapons() {
			if w.Type == common.EqBomb {
				imd.Color = colornames.Salmon
				imd.Push(exact)
				imd.Circle(radiusPlayer-5, 0)
			}
		}

	}
	drawTimer(txt, canvas, game.States[curFrame].Timer)

}

func degreeToRad(degree float64) (rad float64) {
	return degree * (math.Pi / 180)
}

func drawBomb(sprites map[string]*pixel.Sprite, bomb *common.Bomb, match *match.Match, canvas *pixelgl.Canvas) {
	var bombSprite *pixel.Sprite
	pos := bomb.Position()
	exact := position(&pos, match)

	if match.States[curFrame].Timer.Phase == ocom.PhasePlanted {
		bombSprite = sprites["bombRed"]
		bombSprite.Draw(canvas, pixel.IM.Scaled(pixel.ZV, .5).Moved(exact))
	} else {
		if bomb.Carrier == nil {
			bombSprite = sprites["bombCarry"]
			bombSprite.Draw(canvas, pixel.IM.Scaled(pixel.ZV, .5).Moved(exact))
		}
	}

}

func drawKills(match *match.Match, sprites map[string]*pixel.Sprite, txt *text.Text, canvas *pixelgl.Canvas, overview *pixel.Sprite) {
	feedX := float64(mapXOffset) + overview.Frame().W() + 3
	feedV := pixel.V(feedX, 400)
	txt.Orig = feedV
	killMat := pixel.IM.Scaled(txt.Orig, .25)
	kills := match.Killfeed[curFrame]
	txt.LineHeight = txt.Atlas().LineHeight() * 2

	for _, kill := range kills {
		attacker := playerFromName(kill.KillerName, match)
		victim := playerFromName(kill.VictimName, match)
		attackerName := shortName(&attacker, teamOne, teamTwo)
		victimName := shortName(&victim, teamOne, teamTwo)
		weapon := kill.Weapon
		dot := txt.Dot
		fmt.Fprintln(txt, attackerName)
		txt.Dot = dot.Add(pixel.V(700, 0))
		fmt.Fprintln(txt, victimName)
		dot.Y = (dot.Y * .25) + 300

		if sprites[weapon] != nil {

			if weapon == "Knife" {
				sprites[weapon].Draw(canvas, pixel.IM.Scaled(pixel.ZV, .3).Moved(dot.Add(pixel.V(135, 0))))
			} else {
				sprites[weapon].Draw(canvas, pixel.IM.Scaled(pixel.ZV, .6).Moved(dot.Add(pixel.V(135, 0))))
			}
		}
	}

	txt.Draw(canvas, killMat)
	txt.Clear()
}

func drawInfoBars(match *match.Match, canvas *pixelgl.Canvas, sprites map[string]*pixel.Sprite, txtInfo *text.Text) {
	imdInfo := imdraw.New(nil)
	var cts, ts []ocom.Player
	for _, player := range match.States[curFrame].Players {
		if player.Team == common.TeamCounterTerrorists {
			cts = append(cts, player)

		} else {
			ts = append(ts, player)
		}
	}
	sort.Slice(cts, func(i, j int) bool { return cts[i].SteamID64 < cts[j].SteamID64 })
	sort.Slice(ts, func(i, j int) bool { return ts[i].SteamID64 < ts[j].SteamID64 })
	drawInfoBar(imdInfo, cts, canvas, sprites, txtInfo, colorCounter)
	drawInfoBar(imdInfo, ts, canvas, sprites, txtInfo, colorTerror)

}

func drawInfoBar(imd *imdraw.IMDraw, players []ocom.Player, canvas *pixelgl.Canvas, sprites map[string]*pixel.Sprite,
	txt *text.Text, color color.Color) {
	var pos pixel.Vec
	kit := sprites["defuser"]
	armor := sprites["armor"]
	helmet := sprites["armor_helmet"]
	ctPos := pixel.V(0, canvas.Bounds().Max.Y)
	tPos := pixel.V(mapXOffset+float64(mapOverviewWidth), canvas.Bounds().Max.Y-mapYOffset)
	if len(players) > 0 {
		if players[0].Team == common.TeamCounterTerrorists {
			pos = ctPos
		} else {
			pos = tPos
		}
	}
	txt.Orig = pixel.V(0, canvas.Bounds().Max.Y-50)
	infoMat := pixel.IM
	imd.Color = color
	txt.Color = color
	infoMat = infoMat.Scaled(pos, .3)
	txt.Dot.X = pos.X
	txt.Dot = txt.Dot.Add(pixel.V(0, -10))
	txt.LineHeight = txt.Atlas().LineHeight() * 1.5
	var yOffset float64
	dot := txt.Dot
	for _, player := range players {

		if player.Health > 0 {
			hpX := pos.X + float64(player.Health)*(mapXOffset/100)
			imd.Push(pixel.V(hpX, canvas.Bounds().Max.Y-yOffset))
			imd.Push(pixel.V(pos.X, canvas.Bounds().Max.Y-yOffset))
			imd.Color = color
			imd.Line(3)
			txt.Color = colornames.Ghostwhite
			fmt.Fprintln(txt, shortName(&player, teamOne, teamTwo))
			txt.Dot = dot.Add(pixel.V(800, 0))
			if player.Health < 75 && player.Health >= 50 {
				txt.Color = colornames.Yellow
			}
			if player.Health < 50 && player.Health >= 30 {
				txt.Color = colornames.Darkorange
			}
			if player.Health < 30 {
				txt.Color = colornames.Red
			}
			fmt.Fprintln(txt, "HP:", player.Health)
			txt.LineHeight = txt.Atlas().LineHeight() * 1.5
			txt.Dot = dot.Add(pixel.V(0, -80))
			txt.Color = colornames.Greenyellow
			fmt.Fprintln(txt, "$", player.Money)
			txt.Dot = dot.Add(pixel.V(0, -160))
			txt.Color = colornames.Ghostwhite
			fmt.Fprintln(txt, "K:", player.Kills, "A:", player.Assists, "D:", player.Deaths)

			if player.Armor > 0 && player.Helmet {
				helmet.Draw(canvas, pixel.IM.Moved(pixel.V(pos.X+170, canvas.Bounds().Max.Y-yOffset-60)))
			} else if player.Armor > 0 {
				armor.Draw(canvas, pixel.IM.Moved(pixel.V(pos.X+170, canvas.Bounds().Max.Y-yOffset-60)))
			}
			if player.Kit {
				kit.Draw(canvas, pixel.IM.Moved(pixel.V(pos.X+135, canvas.Bounds().Max.Y-yOffset-60)))
			}
			var nadeCounter float64
			weapons := player.Weapons()
			sort.Slice(weapons, func(i, j int) bool { return weapons[i].Type < weapons[j].Type })
			for _, w := range weapons {
				if w.Class() == common.EqClassSMG || w.Class() == common.EqClassHeavy || w.Class() == common.EqClassRifle {
					weapon := w.String()
					if sprites[weapon] != nil {
						sprites[weapon].Draw(canvas, pixel.IM.Scaled(pixel.ZV, .6).Moved(pixel.V(pos.X+230, canvas.Bounds().Max.Y-yOffset-40)))
					}
				}
				if w.Class() == common.EqClassPistols {
					weapon := w.String()
					if sprites[weapon] != nil {
						sprites[weapon].Draw(canvas, pixel.IM.Scaled(pixel.ZV, .6).Moved(pixel.V(pos.X+250, canvas.Bounds().Max.Y-yOffset-60)))
					}
				}
				if w.Class() == common.EqClassGrenade {
					var nadeSprite *pixel.Sprite
					switch w.Type {
					case common.EqDecoy:
						nadeSprite = sprites["Decoy Grenade"]
					case common.EqMolotov:
						nadeSprite = sprites["molotov"]
					case common.EqIncendiary:
						nadeSprite = sprites["molotov"]
					case common.EqFlash:
						nadeSprite = sprites["Flashbang"]
					case common.EqSmoke:
						nadeSprite = sprites["Smoke Grenade"]
					case common.EqHE:
						nadeSprite = sprites["HE Grenade"]
					}

					for i := 0; i < player.AmmoLeft[w.AmmoType()]; i++ {
						nadeSprite.Draw(canvas, pixel.IM.Moved(pixel.V(pos.X+10+nadeCounter*35, canvas.Bounds().Max.Y-yOffset-90)))
						nadeCounter++
					}

				}
				if w.Class() == common.EqClassEquipment {
					if w.Type == common.EqBomb {
						sprites["bombCarry"].Draw(canvas, pixel.IM.Scaled(pixel.ZV, .6).Moved(pixel.V(pos.X+170, canvas.Bounds().Max.Y-yOffset-30)))
					}
				}
			}

		}

		if player.Health < 1 {
			txt.Color = pixel.Alpha(.5)
			dot := txt.Dot
			fmt.Fprintln(txt, shortName(&player, teamOne, teamTwo))
			txt.Dot = dot.Add(pixel.V(800, 0))
			fmt.Fprintln(txt, "HP:", player.Health)
			txt.Dot = dot.Add(pixel.V(0, -80))
			fmt.Fprintln(txt, "$", player.Money)
			txt.Dot = dot.Add(pixel.V(0, -160))
			fmt.Fprintln(txt, "K:", player.Kills, "A:", player.Assists, "D:", player.Deaths)
		}
		dot = dot.Add(pixel.V(0, -370))
		txt.Dot = dot

		yOffset += infoBarHeight
	}

	imd.Draw(canvas)
	txt.Draw(canvas, infoMat)
	txt.Clear()
}

func drawWeapon(imd *imdraw.IMDraw, player *ocom.Player, viewAngle float64, exact pixel.Vec, canvas *pixelgl.Canvas) {

	gunline := pixel.V(0, 0)
	imd.SetMatrix(pixel.IM.Rotated(exact, viewAngle))
	var color color.RGBA

	if player.ActiveWeapon.Type.Class() == common.EqClassGrenade {

		switch player.ActiveWeapon.Type {
		case common.EqDecoy:
			color = colornames.Saddlebrown
		case common.EqMolotov:
			color = colornames.Orangered
		case common.EqIncendiary:
			color = colornames.Orangered
		case common.EqFlash:
			color = colornames.Floralwhite
		case common.EqSmoke:
			color = colornames.Darkgray
		case common.EqHE:
			color = colornames.Lawngreen
		}

		imd.Color = color
		imd.Push(exact.Add(pixel.V(radiusPlayer-2, radiusPlayer-2)))
		imd.Circle(3, 0)

	} else {
		imd.EndShape = imdraw.SharpEndShape
		if player.ActiveWeapon.Type == common.EqAWP {
			gunline = awpLine
			imd.Color = colornames.Darkturquoise

		} else {
			gunline = rifleLine
			imd.Color = colornames.Crimson

		}
		imd.Push(exact.Add(pixel.V(radiusPlayer-2, radiusPlayer-2)))
		imd.Push(exact.Add(gunline))
		imd.Line(3.5)
	}

}

func drawTimer(txt *text.Text, canvas *pixelgl.Canvas, timer ocom.Timer) {
	txt.Clear()
	if timer.Phase == ocom.PhaseWarmup {
		fmt.Fprintln(txt, "Warm Up")
	} else {
		minutes := int(timer.TimeRemaining.Minutes())
		seconds := int(timer.TimeRemaining.Seconds()) - 60*minutes
		timeString := fmt.Sprintf("%d:%2d", minutes, seconds)
		var color = colornames.Floralwhite
		if timer.Phase == ocom.PhasePlanted {
			color = colornames.Red
		} else if timer.Phase == ocom.PhaseRestart {
			color = colornames.Greenyellow
		} else {
			color = colornames.Floralwhite
		}
		txt.Color = color
		fmt.Fprintln(txt, timeString)
	}

	winMinX := canvas.Bounds().Min.X
	winMinY := canvas.Bounds().Min.Y
	timerPos := pixel.V(winMinX+10, winMinY+300)
	timeMat := pixel.IM.Scaled(txt.Dot, .4)
	timeMat = timeMat.Moved(timerPos.Sub(txt.Dot))

	txt.Draw(canvas, timeMat)
	txt.Color = colornames.Floralwhite
	txt.Clear()

	// Draw menu icon
	if !loadCtrl {
		leftCorner := canvas.Bounds().Min
		leftOffset := canvas.Bounds().Center().Sub(leftCorner)
		menu := controls["barsHorizontal"]
		menu.SetOffset(pixel.V(-leftOffset.X+20, -leftOffset.Y+20))
		centerMat := pixel.IM.Scaled(pixel.ZV, menu.Scale)
		centerMat = centerMat.Moved(canvas.Bounds().Center())
		centerMat = centerMat.Moved(menu.Offset)
		menu.Sprite.Draw(canvas, centerMat)
	}
}

func drawGrenade(imd *imdraw.IMDraw, grenade *common.GrenadeProjectile, match *match.Match) {
	imd.SetMatrix(pixel.IM)
	//gPath gets a list of positions of the grenade projectile
	gPath := grenade.Trajectory
	//pos gets the last position from the list of positions in gPath
	pos := gPath[len(gPath)-1]

	exact := position(&pos, match)
	var color = colornames.Floralwhite

	switch grenade.WeaponInstance.Type {
	case common.EqDecoy:
		color = colornames.Saddlebrown
	case common.EqMolotov:
		color = colornames.Orangered
	case common.EqIncendiary:
		color = colornames.Orangered
	case common.EqFlash:
		color = colornames.Floralwhite
	case common.EqSmoke:
		color = colornames.Darkgray
	case common.EqHE:
		color = colornames.Lawngreen
	}
	imd.Color = color
	imd.Push(exact.Add(pixel.V(radiusPlayer-2, radiusPlayer-2)))
	imd.Circle(3, 0)
}

func drawGrenadeEffect(effect *ocom.GrenadeEffect, match *match.Match,
	parts map[string]*part.Particles, batches map[string]*pixel.Batch, canvas *pixelgl.Canvas, dt *float64) {
	pos := effect.GrenadeEvent.Position
	exact := position(&pos, match)

	heB := batches["he"]
	heP := parts["he"]
	flashB := batches["flash"]
	flashP := parts["flash"]
	smokeB := batches["smoke"]
	smokeP := parts["smoke"]
	flashB.Clear()
	smokeB.Clear()
	heB.Clear()

	switch effect.GrenadeEvent.GrenadeType {

	case common.EqFlash:
		flashB.SetMatrix(pixel.IM.Scaled(pixel.ZV, .15).Moved(exact))
		flashP.DrawAll(flashB)
		flashB.Draw(canvas)

	case common.EqSmoke:
		smokeB.SetMatrix(pixel.IM.Scaled(pixel.ZV, .15).Moved(exact))
		smokeP.DrawAll(smokeB)
		smokeB.Draw(canvas)

	case common.EqHE:
		heB.SetMatrix(pixel.IM.Scaled(pixel.ZV, .1).Moved(exact))
		heP.DrawAll(heB)
		heB.Draw(canvas)
	}

}

func drawInferno(imd *imdraw.IMDraw, inferno *common.Inferno, match *match.Match, parts map[string]*part.Particles,
	batches map[string]*pixel.Batch, canvas *pixelgl.Canvas, dt *float64) {

	fireB := batches["fire"]
	fireP := parts["fire"]
	fireB.Clear()

	hull := inferno.Fires().ConvexHull2D()
	coordinates := make([]pixel.Vec, 0)

	for _, v := range hull {
		scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(v.X, v.Y)
		scaledXoff := (scaledX + mapXOffset)
		scaledYoff := 1024 - (scaledY + mapYOffset)
		toVec := pixel.V(scaledXoff, scaledYoff)
		coordinates = append(coordinates, toVec)
	}
	center := getPolyCentroid(coordinates)
	/* coordinates = append(coordinates, center)
	for _, fire := range coordinates {
		fireB.SetMatrix(pixel.IM.Scaled(pixel.ZV, .04).Moved(center))
		fireP.DrawAll(fireB)
	} */
	fireB.SetMatrix(pixel.IM.Scaled(pixel.ZV, .2).Moved(center))
	fireP.DrawAll(fireB)
	fireB.Draw(canvas)

}

func position(pos *r3.Vector, match *match.Match) pixel.Vec {
	scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(pos.X, pos.Y)
	exactX := scaledX + mapXOffset
	exactY := 1024 - (scaledY + mapYOffset)
	exact := pixel.V(exactX, exactY)
	return exact
}

func getPolyCentroid(vertices []pixel.Vec) pixel.Vec {

	nVert := float64(len(vertices))
	xSum := 0.0
	ySum := 0.0
	for _, v := range vertices {
		xSum += v.X
		ySum += v.Y
	}
	xAvg := xSum / nVert
	yAvg := ySum / nVert
	centroid := pixel.V(xAvg, yAvg)
	return centroid
}

func indexOf(element pixel.Vec, data []pixel.Vec) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1
}

func drawShot(imd *imdraw.IMDraw, canvas *pixelgl.Canvas, shot *ocom.Shot, match *match.Match) {
	pos := shot.Position
	shooter := position(&pos, match)
	viewAngle := degreeToRad(float64(shot.ViewDirectionX)) - .7
	color := colornames.Floralwhite

	if shot.IsAwpShot {
		color = colornames.Crimson
	}
	imd.Color = color
	target := shooter.Add(shotLength)

	imd.SetMatrix(pixel.IM.Rotated(shooter, viewAngle))
	imd.Push(shooter.Add(pixel.V(radiusPlayer+1, radiusPlayer+1)))
	imd.Push(target)
	imd.Line(1)

}

func drawControls(c *pixelgl.Canvas, match *match.Match, win *pixelgl.Window, spriteList map[string]*pixel.Sprite) {
	leftCorner := c.Bounds().Min
	leftOffset := c.Bounds().Center().Sub(leftCorner)
	win.SetMatrix(pixel.IM)
	play := controls["Play"]
	rewind := controls["Rewind"]
	rewind.SetOffset(pixel.V(-75, 0))
	fastForward := controls["FastForward"]
	fastForward.SetOffset(pixel.V(75, 0))
	menu := controls["barsHorizontal"]
	menu.SetOffset(pixel.V(-leftOffset.X+20, 0))
	var ctrl *ocom.Control

	for i, img := range spriteList {
		var mask color.Color
		if i == "Pause" {
			if paused {
				ctrl = play
			} else {
				continue
			}
		} else {
			ctrl = controls[i]
		}
		if i == "Play" {
			if paused {
				continue
			}
		}

		centerMat := pixel.IM.Scaled(pixel.ZV, ctrl.Scale)
		centerMat = centerMat.Moved(c.Bounds().Center())
		centerMat = centerMat.Moved(pixel.V(0, -5))
		centerMat = centerMat.Moved(ctrl.Offset)

		if ctrl.Status == selected {
			mask = color.RGBA{169, 169, 169, 255}

			img.DrawColorMask(c, centerMat, mask)
		} else {
			img.Draw(c, centerMat)
		}
	}

}

func drawScore(match *match.Match, txt *text.Text, canvas *pixelgl.Canvas) {
	imd := imdraw.New(nil)
	ctScore := match.States[curFrame].TeamCounterTerrorists.Score
	tScore := match.States[curFrame].TeamTerrorists.Score
	ctName := match.States[curFrame].TeamCounterTerrorists.ClanName
	tName := match.States[curFrame].TeamTerrorists.ClanName
	txt.Color = colornames.White
	tScoreString := strconv.Itoa(tScore)
	ctScoreString := strconv.Itoa(ctScore)
	var message string
	message = ctName + ":" + ctScoreString + " - " + tName + ":" + tScoreString
	ctmessage := ctName + ":" + ctScoreString + "  "
	tmessage := "  " + tName + ":" + tScoreString
	topCenter := canvas.Bounds().Center().Add(pixel.V(0, canvas.Bounds().H()/2))
	width := txt.BoundsOf(message).W()
	scoreMin := pixel.V(topCenter.X-(width/4), topCenter.Y-40)
	scoreMax := pixel.V(topCenter.X+(width/4), topCenter.Y-2)
	rect := pixel.R(scoreMin.X, scoreMin.Y, scoreMax.X, scoreMax.Y)
	imd.Color = color.RGBA{85, 90, 99, 230}
	imd.Push(rect.Min, rect.Max)
	imd.Rectangle(0)
	imd.Color = colornames.White
	imd.Push(rect.Min.Add(pixel.V(-1, -1)), rect.Max.Add(pixel.V(1, 1)))
	imd.Rectangle(1)

	scoreMat := pixel.IM.Scaled(txt.Dot, .5)
	txt.Dot = rect.Center()
	txt.Dot.X = txt.Dot.X * 2
	txt.Dot.X -= txt.BoundsOf(message).W() / 2
	nextDot := txt.Dot
	txt.Color = colornames.Dodgerblue
	fmt.Fprintln(txt, ctmessage)
	txt.Dot = nextDot
	txt.Dot.X = nextDot.X + txt.BoundsOf(ctmessage).W()
	txt.Color = colornames.Darkorange
	fmt.Fprintln(txt, tmessage)
	imd.Draw(canvas)
	txt.Draw(canvas, scoreMat)
	txt.Clear()
}

func drawFrameBar(canvas *pixelgl.Canvas, game *match.Match, txt *text.Text) {
	imdRounds := imdraw.New(nil)
	totalFrames := float64(match.TotalFrames)
	currentFrame := float64(curFrame)
	watchedPercent := (currentFrame / (totalFrames / 100))
	imd := imdraw.New(nil)
	minX := canvas.Bounds().Min.X + 3
	minY := canvas.Bounds().H() - 10
	maxX := canvas.Bounds().Max.X - 3
	maxY := canvas.Bounds().H() - 3
	min := pixel.V(minX, minY)
	max := pixel.V(maxX, maxY)
	playBar = pixel.Rect{min, max}
	imd.Color = colornames.Darkgray
	imd.Push(min, max)
	imd.Rectangle(2)
	progress := (playBar.W() / 100) * watchedPercent
	barStart := pixel.V(minX+2, minY+(playBar.H()/2))
	barEnd := pixel.V(progress, maxY-(playBar.H()/2))
	imd.Color = colornames.Red
	imd.Push(barStart, barEnd)
	imd.Line(4)
	imd.Draw(canvas)

	rounds := game.RoundStarts
	var ctScoreOld, tScoreOld, ctScore, tScore int
	var previousState ocom.OverviewState

	for i, round := range rounds {
		roundState := game.States[round]

		if i == 0 {
			ctScoreOld = 0
			tScoreOld = 0
		} else {
			previousState = game.States[rounds[i-1]]
			ctScoreOld = previousState.TeamCounterTerrorists.Score
			tScoreOld = previousState.TeamTerrorists.Score
		}
		ctScore = roundState.TeamCounterTerrorists.Score
		tScore = roundState.TeamTerrorists.Score
		if ctScore > ctScoreOld || tScore > tScoreOld {
			framePos := float64(round) / (totalFrames / 100)
			frameToBar := (playBar.W() / 100) * framePos
			kMin := pixel.V(frameToBar, minY+1)
			kMax := pixel.V(frameToBar, maxY-1)

			if ctScore > ctScoreOld {
				imdRounds.Color = colorCounter
			}
			if tScore > tScoreOld {
				imdRounds.Color = colorTerror
			}
			imdRounds.Push(kMin)
			imdRounds.Push(kMax)
			imdRounds.Line(3)
		}
	}
	imdRounds.Draw(canvas)
}

func shortName(player *ocom.Player, teamOne ocom.Clan, teamTwo ocom.Clan) string {

	if player.ClanName == teamOne.ClanName {
		return strings.Replace(player.Name, teamOne.Tag, "", 1)
	}
	if player.ClanName == teamTwo.ClanName {
		return strings.Replace(player.Name, teamTwo.Tag, "", 1)
	}
	return player.Name
}

func playerFromName(name string, match *match.Match) ocom.Player {
	var player ocom.Player
	players := match.States[curFrame].Players
	for _, p := range players {
		if p.Name == name {
			player = p
		}
	}

	return player
}
