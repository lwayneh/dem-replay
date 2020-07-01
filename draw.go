package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	ocom "github.com/lwayneh/dem-replay/common"
	"github.com/lwayneh/dem-replay/match"
	common "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	meta "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/metadata"

	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	radiusPlayer   int32   = 10
	radiusSmoke    int32   = 25
	killfeedHeight int32   = 15
	shotLength     float64 = 1000
)

var (
	colorTerror       = sdl.Color{252, 176, 12, 255}
	colorCounter      = sdl.Color{89, 206, 200, 255}
	colorMoney        = sdl.Color{0, 255, 0, 255}
	colorBomb         = sdl.Color{255, 0, 0, 255}
	colorEqDecoy      = sdl.Color{102, 34, 0, 255}
	colorEqMolotov    = sdl.Color{255, 153, 0, 255}
	colorEqIncendiary = sdl.Color{255, 153, 0, 255}
	colorInferno      = sdl.Color{255, 153, 0, 100}
	colorEqFlash      = sdl.Color{128, 170, 255, 255}
	colorEqSmoke      = sdl.Color{153, 153, 153, 255}
	colorSmoke        = sdl.Color{153, 153, 153, 100}
	colorEqHE         = sdl.Color{85, 150, 0, 255}
	colorDarkWhite    = sdl.Color{200, 200, 200, 255}
	colorFlashEffect  = sdl.Color{200, 200, 200, 180}
	colorAwpShot      = sdl.Color{255, 50, 0, 255}
	helmetRec         = &sdl.Rect{416, 127, 18, 20}
	armorRec          = &sdl.Rect{176, 166, 19, 20}
	kitRec            = &sdl.Rect{450, 288, 26, 24}
	cartRec           = &sdl.Rect{290, 385, 36, 31}
	mollyRec          = &sdl.Rect{456, 254, 22, 29}
	flashRec          = &sdl.Rect{453, 385, 24, 28}
	smokeRec          = &sdl.Rect{455, 67, 14, 27}
	heRec             = &sdl.Rect{228, 378, 21, 27}
	decoyRec          = &sdl.Rect{396, 413, 25, 30}
)

func drawPlayer(renderer *sdl.Renderer, player *ocom.Player, font *ttf.Font, match *match.Match) {

	var color sdl.Color
	if player.Team == common.TeamTerrorists {
		color = colorTerror
	} else {
		color = colorCounter
	}

	if player.Health > 0 {

		pos := player.LastAlivePosition
		scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(pos.X, pos.Y)
		var scaledXInt int32 = int32(scaledX) + mapXOffset
		var scaledYInt int32 = int32(scaledY) + mapYOffset

		gfx.AACircleColor(renderer, scaledXInt, scaledYInt, radiusPlayer, color)

		DrawString(renderer, cropStringToN(player.Name, 10), color, scaledXInt+10, scaledYInt+10, font)

		viewAngle := -int32(player.ViewDirectionX) // negated because of sdl
		gfx.ArcColor(renderer, scaledXInt, scaledYInt, radiusPlayer+1, viewAngle-20, viewAngle+20, colorDarkWhite)
		gfx.ArcColor(renderer, scaledXInt, scaledYInt, radiusPlayer+2, viewAngle-10, viewAngle+10, colorDarkWhite)
		gfx.ArcColor(renderer, scaledXInt, scaledYInt, radiusPlayer+3, viewAngle-5, viewAngle+5, colorDarkWhite)

		if player.FlashDuration > 0.5 {
			timeSinceFlash := time.Duration(float64(match.States[curFrame].IngameTick-player.FlashTick) / match.TickRate * float64(time.Second))
			// 2+ weird offset because player.FlashDuration is imprecise
			remaining := time.Duration((2+player.FlashDuration)*float32(time.Second)) - timeSinceFlash
			// 2+ weird offset because player.FlashDuration is imprecise
			colorFlashEffect.A = uint8((remaining.Seconds() * 255) / (2 + 5.5))
			gfx.FilledCircleColor(renderer, scaledXInt, scaledYInt, radiusPlayer-5, colorFlashEffect)
		}

		for _, w := range player.Weapons() {
			if w.Type == common.EqBomb {
				gfx.AACircleColor(renderer, scaledXInt, scaledYInt, radiusPlayer-1, colorBomb)
				gfx.AACircleColor(renderer, scaledXInt, scaledYInt, radiusPlayer-2, colorBomb)
			}
		}

		if player.IsDefusing {
			color.A = 200
			gfx.CharacterColor(renderer, scaledXInt-radiusPlayer/4, scaledYInt-radiusPlayer/4, 'D', color)
			color.A = 255
		}
	} else {
		pos := player.LastAlivePosition

		scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(pos.X, pos.Y)
		var scaledXInt int32 = int32(scaledX) + mapXOffset
		var scaledYInt int32 = int32(scaledY) + mapYOffset

		color.A = 150
		gfx.CharacterColor(renderer, scaledXInt, scaledYInt, 'X', color)
		color.A = 255
	}

}

func drawGrenade(renderer *sdl.Renderer, grenade *common.GrenadeProjectile, match *match.Match) {
	//gPath gets a list of positions of the grenade projectile
	gPath := grenade.Trajectory
	//pos gets the last position from the list of positions in gPath
	pos := gPath[len(gPath)-1]

	scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(pos.X, pos.Y)
	var scaledXInt int32 = int32(scaledX) + mapXOffset
	var scaledYInt int32 = int32(scaledY) + mapYOffset
	var color sdl.Color

	switch grenade.WeaponInstance.Type {
	case common.EqDecoy:
		color = colorEqDecoy
	case common.EqMolotov:
		color = colorEqMolotov
	case common.EqIncendiary:
		color = colorEqIncendiary
	case common.EqFlash:
		color = colorEqFlash
	case common.EqSmoke:
		color = colorEqSmoke
	case common.EqHE:
		color = colorEqHE
	}

	gfx.BoxColor(renderer, scaledXInt-2, scaledYInt-3, scaledXInt+2, scaledYInt+3, color)
}

func drawGrenadeEffect(renderer *sdl.Renderer, effect *ocom.GrenadeEffect, match *match.Match) {
	pos := effect.GrenadeEvent.Position

	scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(pos.X, pos.Y)
	var scaledXInt int32 = int32(scaledX) + mapXOffset
	var scaledYInt int32 = int32(scaledY) + mapYOffset

	switch effect.GrenadeEvent.GrenadeType {
	case common.EqFlash:
		gfx.AACircleColor(renderer, scaledXInt, scaledYInt, int32(effect.Lifetime), colorEqFlash)
	case common.EqHE:
		gfx.AACircleColor(renderer, scaledXInt, scaledYInt, int32(effect.Lifetime), colorEqHE)
	case common.EqSmoke:
		// 4.9 is the reference on Inferno for the value for radiusSmoke
		scaledRadiusSmoke := int32(float64(radiusSmoke) * 4.9 / meta.MapNameToMap[match.MapName].Scale)
		gfx.FilledCircleColor(renderer, scaledXInt, scaledYInt, scaledRadiusSmoke, colorSmoke)
		// only draw the outline if the smoke is not fading
		if effect.Lifetime < 15*match.SmokeEffectLifetime/18 {
			gfx.AACircleColor(renderer, scaledXInt, scaledYInt, scaledRadiusSmoke, colorDarkWhite)
		}
		gfx.ArcColor(renderer, scaledXInt, scaledYInt, 10, int32(270+effect.Lifetime*360/match.SmokeEffectLifetime), 630, colorDarkWhite)
	}
}

func drawInferno(renderer *sdl.Renderer, inferno *common.Inferno, match *match.Match) {
	hull := inferno.Fires().ConvexHull2D()
	xCoordinates := make([]int16, 0)
	yCoordinates := make([]int16, 0)

	for _, v := range hull {
		scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(v.X, v.Y)
		scaledXInt := int16(scaledX) + int16(mapXOffset)
		scaledYInt := int16(scaledY) + int16(mapYOffset)
		xCoordinates = append(xCoordinates, scaledXInt)
		yCoordinates = append(yCoordinates, scaledYInt)
	}

	gfx.FilledPolygonColor(renderer, xCoordinates, yCoordinates, colorInferno)
	gfx.AAPolygonColor(renderer, xCoordinates, yCoordinates, colorInferno)
}

func drawBomb(renderer *sdl.Renderer, bomb *common.Bomb, match *match.Match) {
	pos := bomb.Position()
	if bomb.Carrier != nil {
		return
	}

	scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(pos.X, pos.Y)
	var scaledXInt int32 = int32(scaledX) + mapXOffset
	var scaledYInt int32 = int32(scaledY) + mapYOffset

	gfx.BoxColor(renderer, scaledXInt-3, scaledYInt-2, scaledXInt+3, scaledYInt+2, colorBomb)
}

func DrawString(renderer *sdl.Renderer, text string, color sdl.Color, x, y int32, font *ttf.Font) {
	textSurface, err := font.RenderUTF8Blended(text, color)
	if err != nil {
		log.Fatal(err)
	}
	defer textSurface.Free()
	textTexture, err := renderer.CreateTextureFromSurface(textSurface)
	if err != nil {
		log.Fatal(err)
	}
	textTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	defer textTexture.Destroy()
	textRect := &sdl.Rect{
		X: x,
		Y: y,
		W: textSurface.W,
		H: textSurface.H,
	}
	err = renderer.Copy(textTexture, nil, textRect)
	if err != nil {
		log.Fatal(err)
	}
}

func drawInfobars(renderer *sdl.Renderer, match *match.Match, font *ttf.Font, texture *sdl.Texture) {
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
	drawInfobar(renderer, cts, 0, mapYOffset, colorCounter, font, texture)
	drawInfobar(renderer, ts, mapXOffset+mapOverviewWidth, mapYOffset, colorTerror, font, texture)
	drawKillfeed(renderer, match.Killfeed[curFrame], mapXOffset+mapOverviewWidth, mapYOffset+600, font)
	drawTimer(renderer, match.States[curFrame].Timer, 0, mapYOffset+600, font)
}

func drawInfobar(renderer *sdl.Renderer, players []ocom.Player, x, y int32, color sdl.Color, font *ttf.Font, texture *sdl.Texture) {
	var yOffset int32
	for _, player := range players {
		if player.Health > 0 {
			gfx.BoxColor(renderer, x+int32(player.Health)*(mapXOffset/100), yOffset, x, yOffset+5, color)
		}
		if player.Health < 1 {
			color.A = 150
		}
		DrawString(renderer, cropStringToN(player.Name, 20), color, x+85, yOffset+10, font)
		color.A = 255
		DrawString(renderer, fmt.Sprintf("%v", player.Health), color, x+5, yOffset+10, font)
		if player.Armor > 0 && player.Helmet {
			drawImg(renderer, texture, helmetRec, x+5, yOffset+80, 0)
		} else if player.Armor > 0 {
			drawImg(renderer, texture, armorRec, x+5, yOffset+80, 0)
		}
		if player.Kit {
			drawImg(renderer, texture, kitRec, x+30, yOffset+80, 0)
		}
		drawImg(renderer, texture, cartRec, x+2, yOffset+30, -.4)
		DrawString(renderer, fmt.Sprintf("$%v", player.Money), colorMoney, x+25, yOffset+33, font)
		var nadeCounter int32
		weapons := player.Weapons()
		sort.Slice(weapons, func(i, j int) bool { return weapons[i].Type < weapons[j].Type })
		for _, w := range weapons {
			if w.Class() == common.EqClassSMG || w.Class() == common.EqClassHeavy || w.Class() == common.EqClassRifle {
				DrawString(renderer, w.Type.String(), color, x+150, yOffset+30, font)
			}
			if w.Class() == common.EqClassPistols {
				DrawString(renderer, w.Type.String(), color, x+150, yOffset+55, font)
			}
			if w.Class() == common.EqClassGrenade {
				var nadeRect sdl.Rect
				switch w.Type {
				case common.EqDecoy:
					nadeRect = *decoyRec
				case common.EqMolotov:
					nadeRect = *mollyRec
				case common.EqIncendiary:
					nadeRect = *mollyRec
				case common.EqFlash:
					nadeRect = *flashRec
				case common.EqSmoke:
					nadeRect = *smokeRec
				case common.EqHE:
					nadeRect = *heRec
				}

				for i := 0; i < player.AmmoLeft[w.AmmoType()]; i++ {
					drawImg(renderer, texture, &nadeRect, x+150+nadeCounter*35, yOffset+80, 0)
					nadeCounter++
				}

			}
			if w.Class() == common.EqClassEquipment {
				if w.Type == common.EqBomb {
					gfx.BoxColor(renderer, x+50, yOffset+12, x+45+12, yOffset+12+9, colorBomb)
				}
			}
		}
		kdaInfo := fmt.Sprintf("%v / %v / %v", player.Kills, player.Assists, player.Deaths)
		DrawString(renderer, kdaInfo, color, x+5, yOffset+55, font)

		yOffset += infobarElementHeight
	}
}

func drawKillfeed(renderer *sdl.Renderer, killfeed []ocom.Kill, x, y int32, font *ttf.Font) {
	var yOffset int32
	for _, kill := range killfeed {
		var colorKiller, colorVictim sdl.Color
		if kill.KillerTeam == common.TeamCounterTerrorists {
			colorKiller = colorCounter
		} else if kill.KillerTeam == common.TeamTerrorists {
			colorKiller = colorTerror
		} else {
			colorKiller = colorDarkWhite
		}
		if kill.VictimTeam == common.TeamCounterTerrorists {
			colorVictim = colorCounter
		} else {
			colorVictim = colorTerror
		}
		killerName := cropStringToN(kill.KillerName, 10)
		victimName := cropStringToN(kill.VictimName, 10)
		weaponName := cropStringToN(kill.Weapon, 10)
		DrawString(renderer, killerName, colorKiller, x+5, y+yOffset, font)
		DrawString(renderer, weaponName, colorDarkWhite, x+110, y+yOffset, font)
		DrawString(renderer, victimName, colorVictim, x+200, y+yOffset, font)
		yOffset += killfeedHeight
	}
}

func drawTimer(renderer *sdl.Renderer, timer ocom.Timer, x, y int32, font *ttf.Font) {
	if timer.Phase == ocom.PhaseWarmup {
		DrawString(renderer, "Warmup", colorDarkWhite, x+5, y, font)
	} else {
		minutes := int(timer.TimeRemaining.Minutes())
		seconds := int(timer.TimeRemaining.Seconds()) - 60*minutes
		timeString := fmt.Sprintf("%d:%2d", minutes, seconds)
		var color sdl.Color
		if timer.Phase == ocom.PhasePlanted {
			color = colorBomb
		} else if timer.Phase == ocom.PhaseRestart {
			color = colorEqHE
		} else {
			color = colorDarkWhite
		}
		DrawString(renderer, timeString, color, x+5, y, font)
	}
}

func drawShot(renderer *sdl.Renderer, shot *ocom.Shot, match *match.Match) {
	pos := shot.Position
	viewAngleDegrees := -shot.ViewDirectionX // negated because of sdl
	viewAngleRadian := viewAngleDegrees * math.Pi / 180
	color := colorDarkWhite
	if shot.IsAwpShot {
		color = colorAwpShot
	}

	scaledX, scaledY := meta.MapNameToMap[match.MapName].TranslateScale(pos.X, pos.Y)
	scaledX += math.Cos(float64(viewAngleRadian)) * float64(radiusPlayer)
	scaledY += math.Sin(float64(viewAngleRadian)) * float64(radiusPlayer)
	var scaledXInt int32 = int32(scaledX) + mapXOffset
	var scaledYInt int32 = int32(scaledY) + mapYOffset

	targetX := scaledXInt + int32(math.Cos(float64(viewAngleRadian))*shotLength/meta.MapNameToMap[match.MapName].Scale)
	targetY := scaledYInt + int32(math.Sin(float64(viewAngleRadian))*shotLength/meta.MapNameToMap[match.MapName].Scale)

	gfx.AALineColor(renderer, scaledXInt, scaledYInt, targetX, targetY, color)
}

func cropStringToN(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}

	return s
}

func drawImg(renderer *sdl.Renderer, texture *sdl.Texture, rect *sdl.Rect, x, y int32, scale float32) {
	var scaleW, scaleH int32

	if scale != 0 {
		scaleW = rect.W + (int32(float32(rect.W) * scale))
		scaleH = rect.H + (int32(float32(rect.H) * scale))
	}

	if scale == 0 {
		scaleW = rect.W
		scaleH = rect.H
	}

	dstRec := sdl.Rect{x, y, scaleW, scaleH}
	renderer.Copy(texture, rect, &dstRec)

}
