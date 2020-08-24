package particle

import (
	"container/list"
	"encoding/csv"
	"image"
	_ "image/png"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"

	"github.com/faiface/pixel"
)

type Particle struct {
	Sprite     *pixel.Sprite
	Pos        pixel.Vec
	Rot, Scale float64
	Color      pixel.RGBA
	Data       interface{}
}

type Particles struct {
	Generate            func(p *Particles) *Particle
	Update              func(dt float64, p *Particle) bool
	SpawnAvg, SpawnDist float64
	Sys                 ParticleSystem
	parts               list.List
	spawnTime           float64
}

func (p *Particles) UpdateAll(dt float64) {
	p.spawnTime -= dt
	for p.spawnTime <= 0 {
		p.parts.PushFront(p.Generate(p))
		p.spawnTime += math.Max(0, p.SpawnAvg+rand.NormFloat64()*p.SpawnDist)
	}

	for e := p.parts.Front(); e != nil; e = e.Next() {
		part := e.Value.(*Particle)
		if !p.Update(dt, part) {
			defer p.parts.Remove(e)
		}
	}
}

func (p *Particles) DrawAll(t pixel.Target) {
	for e := p.parts.Front(); e != nil; e = e.Next() {
		part := e.Value.(*Particle)

		part.Sprite.DrawColorMask(
			t,
			pixel.IM.
				Scaled(pixel.ZV, part.Scale).
				Rotated(pixel.ZV, part.Rot).
				Moved(part.Pos),
			part.Color,
		)

	}
}

type ParticleData struct {
	Vel  pixel.Vec
	Time float64
	Life float64
}

type ParticleSystem struct {
	Sheet      pixel.Picture
	Rects      []pixel.Rect
	Orig       pixel.Vec
	StartColor pixel.RGBA
	ColorVar   float64
	VelBasis   []pixel.Vec
	VelDist    float64

	LifeAvg, LifeDist float64
}

func (ps *ParticleSystem) Generate(xp *Particles) *Particle {
	pd := new(ParticleData)
	for _, base := range ps.VelBasis {
		c := math.Max(0, 1+rand.NormFloat64()*ps.VelDist)
		pd.Vel = pd.Vel.Add(base.Scaled(c))
	}
	pd.Vel = pd.Vel.Scaled(1 / float64(len(ps.VelBasis)))
	pd.Life = math.Max(0, ps.LifeAvg+rand.NormFloat64()*ps.LifeDist)

	p := new(Particle)
	p.Data = pd

	p.Pos = ps.Orig
	p.Scale = 1
	p.Color = pixel.Alpha(1)
	p.Sprite = pixel.NewSprite(ps.Sheet, ps.Rects[rand.Intn(len(ps.Rects))])
	xp.Sys = *ps

	return p
}

func (ps *ParticleSystem) Update(dt float64, p *Particle) bool {
	sd := p.Data.(*ParticleData)
	sd.Time += dt

	frac := sd.Time / sd.Life

	p.Pos = p.Pos.Add(sd.Vel.Scaled(dt))
	p.Scale = 0.5 + frac*1.5

	const (
		fadeIn  = 0.1
		fadeOut = 0.5
	)

	if frac < fadeIn {
		p.Color = pixel.Alpha(math.Pow(frac/fadeIn, 0.75))
	} else if frac >= fadeOut {
		p.Color = pixel.Alpha(math.Pow(1-(frac-fadeOut)/(1-fadeOut), 1.5))
	} else {
		p.Color = pixel.Alpha(1)
	}

	if sd.Time >= sd.Life {
		p.Pos = ps.Orig
	}
	return sd.Time < sd.Life
}

func LoadSpriteSheet(sheetPath, descriptionPath string) (sheet pixel.Picture, rects []pixel.Rect, err error) {
	sheetFile, err := os.Open(sheetPath)
	if err != nil {
		return nil, nil, err
	}
	defer sheetFile.Close()

	sheetImg, _, err := image.Decode(sheetFile)
	if err != nil {
		return nil, nil, err
	}

	sheet = pixel.PictureDataFromImage(sheetImg)

	descriptionFile, err := os.Open(descriptionPath)
	if err != nil {
		return nil, nil, err
	}
	defer descriptionFile.Close()

	description := csv.NewReader(descriptionFile)
	for {
		record, err := description.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		x, _ := strconv.ParseFloat(record[0], 64)
		y, _ := strconv.ParseFloat(record[1], 64)
		w, _ := strconv.ParseFloat(record[2], 64)
		h, _ := strconv.ParseFloat(record[3], 64)

		y = sheet.Bounds().H() - y - h

		rects = append(rects, pixel.R(x, y, x+w, y+h))
	}

	return sheet, rects, nil
}

func LoadSpriteSheetAsMap(sheetPath, descriptionPath string) (sheet pixel.Picture, rects map[string]pixel.Rect, err error) {
	rects = make(map[string]pixel.Rect)
	sheetFile, err := os.Open(sheetPath)
	if err != nil {
		return nil, nil, err
	}
	defer sheetFile.Close()

	sheetImg, _, err := image.Decode(sheetFile)
	if err != nil {
		return nil, nil, err
	}

	sheet = pixel.PictureDataFromImage(sheetImg)

	descriptionFile, err := os.Open(descriptionPath)
	if err != nil {
		return nil, nil, err
	}
	defer descriptionFile.Close()

	description := csv.NewReader(descriptionFile)
	for {
		record, err := description.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		name, _ := record[0], err
		x, _ := strconv.ParseFloat(record[1], 64)
		y, _ := strconv.ParseFloat(record[2], 64)
		w, _ := strconv.ParseFloat(record[3], 64)
		h, _ := strconv.ParseFloat(record[4], 64)

		y = sheet.Bounds().H() - y - h

		rects[name] = pixel.R(x, y, x+w, y+h)
	}

	return sheet, rects, nil
}
