package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

//go:embed "Bodo Amat.ttf"
var fontFile []byte
var gameFont font.Face

//go:embed hitSound.wav
var hitFile []byte

//go:embed win.wav
var winFile []byte

//go:embed loss.wav
var lossFile []byte

const (
	scrWidth  = 640
	scrHeight = 480
)

type GameState int

const (
	Menu GameState = iota
	Playing
	Quit
)

type Stream int

const (
	HitSound Stream = iota
	Win
	Loss
)

type Paddle struct {
	img    *ebiten.Image
	pos    *image.Point
	speedY int
}

type Ball struct {
	img      *ebiten.Image
	pos      *image.Point
	velocity *image.Point
}

type Game struct {
	state       GameState
	bg          *ebiten.Image
	paddle      *Paddle
	ai          *Paddle
	ball        *Ball
	win         int
	loss        int
	audioCtx    *audio.Context
	audioPlayer *audio.Player
	audioStream [][]byte
}

func init() {
	ff, err := truetype.Parse(fontFile)
	if err != nil {
		log.Fatal(err)
	}

	gameFont = truetype.NewFace(ff, &truetype.Options{
		Size:    24,
		DPI:     100,
		Hinting: font.HintingFull,
	})
}

func (g *Game) Update() error {
	switch g.state {
	case Menu:
		g.MenuUpdate()
	case Playing:
		g.UpdatePlaying()
	case Quit:
		os.Exit(0)
	}

	return nil
}

func (g *Game) Draw(scr *ebiten.Image) {
	switch g.state {
	case Menu:
		g.DrawMenu(scr)
	case Playing:
		g.DrawPlaying(scr)
	case Quit:
		os.Exit(0)
	}

}

func (g *Game) Layout(outWidth, outHeight int) (int, int) {
	return scrWidth, scrHeight
}

func (g *Game) DrawMenu(dst *ebiten.Image) {
	g.bg.Fill(color.White)

	menu := "Pong\n\n(P)lay\n(Q)uit"

	menuBnds := text.BoundString(gameFont, menu)
	dx, dy := scrWidth/2-menuBnds.Dx()/2, scrHeight/2-menuBnds.Dy()/2
	text.Draw(g.bg, menu, gameFont, dx, dy, color.Black)

	dst.DrawImage(g.bg, nil)
}

func (g *Game) MenuUpdate() {
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		os.Exit(0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyP) {
		g.win = 0
		g.loss = 0
		g.ball.velocity = &image.Point{X: rand.Intn(7-3+1) + 3, Y: rand.Intn(6-3+1) + 3}
		g.state = Playing
	}
}

func (g *Game) DrawPlaying(scr *ebiten.Image) {
	g.bg.Fill(color.White)

	g.paddle.img.Fill(color.Black)
	paddleOpt := &ebiten.DrawImageOptions{}
	paddleOpt.GeoM.Translate(-float64(g.paddle.img.Bounds().Dx()/2), -float64(g.paddle.img.Bounds().Dy())/2)
	paddleOpt.GeoM.Translate(float64(g.paddle.pos.X), float64(g.paddle.pos.Y))

	g.ai.img.Fill(color.Black)
	aiOpt := &ebiten.DrawImageOptions{}
	aiOpt.GeoM.Translate(-float64(g.ai.img.Bounds().Dx()/2), -float64(g.ai.img.Bounds().Dy()/2))
	aiOpt.GeoM.Translate(float64(g.ai.pos.X), float64(g.ai.pos.Y))

	g.ball.img.Fill(color.Black)
	ballOpt := &ebiten.DrawImageOptions{}
	ballOpt.GeoM.Translate(-float64(g.ball.img.Bounds().Dx()/2), -float64(g.ball.img.Bounds().Dy()/2))
	ballOpt.GeoM.Translate(float64(g.ball.pos.X), float64(g.ball.pos.Y))

	g.bg.DrawImage(g.paddle.img, paddleOpt)
	g.bg.DrawImage(g.ai.img, aiOpt)
	g.bg.DrawImage(g.ball.img, ballOpt)

	ebitenutil.DrawLine(g.bg, 0, scrHeight/2, scrWidth, scrHeight/2, color.Black)
	ebitenutil.DrawLine(g.bg, scrWidth/2, 0, scrWidth/2, scrHeight, color.Black)

	winStr := fmt.Sprintf("Wins: %d", g.win)
	winBnds := text.BoundString(gameFont, winStr)
	dx, dy := 20, winBnds.Dy()/2+gameFont.Metrics().Ascent.Ceil()
	text.Draw(g.bg, winStr, gameFont, dx, dy, color.Black)

	lossStr := fmt.Sprintf("Losses: %d", g.loss)
	lossBnds := text.BoundString(gameFont, lossStr)
	dx, dy = scrWidth-lossBnds.Dx()-20, lossBnds.Dy()/2+gameFont.Metrics().Ascent.Ceil()
	text.Draw(g.bg, lossStr, gameFont, dx, dy, color.Black)

	scr.DrawImage(g.bg, nil)
}

func (g *Game) UpdatePlaying() {
	g.Move(g.paddle)
	g.checkPaddleBounds(g.paddle)
	g.ballMove()
	g.checkBallCollision()
	g.aiMove()
	g.checkPaddleBounds(g.ai)

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		g.state = Menu
	}
}

func (g *Game) Move(p *Paddle) {
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyA) {
		p.pos.Y -= p.speedY
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyD) {
		p.pos.Y += p.speedY
	}
}

func (g *Game) checkPaddleBounds(p *Paddle) {
	if p.pos.Y-p.img.Bounds().Dy()/2 <= 0 {
		p.pos.Y = p.img.Bounds().Dy() / 2
	}
	if p.pos.Y+p.img.Bounds().Dy()/2 >= scrHeight {
		p.pos.Y = scrHeight - p.img.Bounds().Dy()/2
	}
}

func (g *Game) ballMove() {
	g.ball.pos.X += g.ball.velocity.X
	g.ball.pos.Y += g.ball.velocity.Y

	maxSpeed := 20
	if g.ball.velocity.X >= maxSpeed {
		g.ball.velocity.X = maxSpeed
	}
	if g.ball.velocity.X <= -maxSpeed {
		g.ball.velocity.X = -maxSpeed
	}
	if g.ball.velocity.Y >= maxSpeed {
		g.ball.velocity.Y = maxSpeed
	}
	if g.ball.velocity.Y <= -maxSpeed {
		g.ball.velocity.Y = -maxSpeed
	}
}

func (g *Game) checkBallCollision() {
	// Top/Bottom collision.
	if g.ball.pos.Y < 0 || g.ball.pos.Y > scrHeight {
		g.ball.velocity.Y *= -1
	}
	// Edge collision.
	if g.ball.pos.X < 0 || g.ball.pos.X > scrWidth {
		if g.ball.pos.X < g.paddle.pos.X {
			g.loss++
			g.playAudio(Loss)
		}
		if g.ball.pos.X > g.ai.pos.X {
			g.win++
			g.playAudio(Win)
		}

		g.ball.pos.X, g.ball.pos.Y = scrWidth/2, scrHeight/2
		g.ball.velocity = &image.Point{X: rand.Intn(7-3+1) + 3, Y: rand.Intn(5-3+1) + 3}
	}

	// Player Collision
	if g.ball.pos.X <= g.paddle.pos.X+g.paddle.img.Bounds().Dx()/2 &&
		g.ball.pos.Y >= g.paddle.pos.Y-g.paddle.img.Bounds().Dy()/2 &&
		g.ball.pos.Y <= g.paddle.pos.Y {
		g.playAudio(HitSound)
		g.ball.velocity.X *= -1
		g.ball.velocity.X += 1
		g.ball.velocity.Y = -int(math.Abs(float64(g.ball.velocity.Y)))
	}
	if g.ball.pos.X <= g.paddle.pos.X+g.paddle.img.Bounds().Dx()/2 &&
		g.ball.pos.Y <= g.paddle.pos.Y+g.paddle.img.Bounds().Dy()/2 &&
		g.ball.pos.Y >= g.paddle.pos.Y {
		g.playAudio(HitSound)
		g.ball.velocity.X *= -1
		g.ball.velocity.X += 1
		g.ball.velocity.Y = int(math.Abs(float64(g.ball.velocity.Y)))
	}

	// AI Collision
	if g.ball.pos.X >= g.ai.pos.X-g.ai.img.Bounds().Dx()/2 &&
		g.ball.pos.Y >= g.ai.pos.Y-g.ai.img.Bounds().Dy()/2 &&
		g.ball.pos.Y <= g.ai.pos.Y {
		g.playAudio(HitSound)
		g.ball.velocity.X *= -1
		g.ball.velocity.X -= 1
		g.ball.velocity.Y = -int(math.Abs(float64(g.ball.velocity.Y)))
	}
	if g.ball.pos.X >= g.ai.pos.X-g.ai.img.Bounds().Dx()/2 &&
		g.ball.pos.Y <= g.ai.pos.Y+g.ai.img.Bounds().Dy()/2 &&
		g.ball.pos.Y >= g.ai.pos.Y {
		g.playAudio(HitSound)
		g.ball.velocity.X *= -1
		g.ball.velocity.X -= 1
		g.ball.velocity.Y = int(math.Abs(float64(g.ball.velocity.Y)))
	}
}

func (g *Game) playAudio(s Stream) {
	g.audioPlayer = g.audioCtx.NewPlayerFromBytes(g.audioStream[s])

	if err := g.audioPlayer.Rewind(); err != nil {
		log.Fatal(err)
	}
	g.audioPlayer.Play()
}

func (g *Game) aiMove() {
	if g.ball.pos.Y > g.ai.pos.Y {
		g.ai.pos.Y += g.ai.speedY
	}
	if g.ball.pos.Y < g.ai.pos.Y {
		g.ai.pos.Y += -g.ai.speedY
	}
}

func main() {
	rand.Seed(time.Now().Unix())

	g := &Game{
		state: Menu,
		bg:    ebiten.NewImage(scrWidth, scrHeight),
		paddle: &Paddle{
			img:    ebiten.NewImage(10, 80),
			pos:    &image.Point{X: 20, Y: (scrHeight / 2)},
			speedY: 7,
		},
		ai: &Paddle{
			img:    ebiten.NewImage(10, 80),
			pos:    &image.Point{X: scrWidth - 20, Y: (scrHeight / 2)},
			speedY: 7,
		},
		ball: &Ball{
			img:      ebiten.NewImage(10, 10),
			pos:      &image.Point{X: scrWidth / 2, Y: scrHeight / 2},
			velocity: &image.Point{X: rand.Intn(7-3+1) + 3, Y: rand.Intn(6-3+1) + 3},
		},
		audioCtx: audio.NewContext(48000),
	}

	g.audioStream = append(g.audioStream, hitFile)
	g.audioStream = append(g.audioStream, winFile)
	g.audioStream = append(g.audioStream, lossFile)

	ebiten.SetWindowTitle("Pong")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
