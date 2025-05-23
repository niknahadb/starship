package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed assets/* assets/meteors/*.png
var assets embed.FS

var (
    PlayerSprite  *ebiten.Image
    MeteorSprites []*ebiten.Image
    BulletSprite  *ebiten.Image
    ScoreFont     font.Face
)

const (
    ScreenWidth  = 800
    ScreenHeight = 600
)

func mustLoadImage(path string) *ebiten.Image {
    f, err := assets.Open(path)
    if err != nil {
        panic(err)
    }
    defer f.Close()
    img, _, err := image.Decode(f)
    if err != nil {
        panic(err)
    }
    return ebiten.NewImageFromImage(img)
}

func mustLoadImages(dir string) []*ebiten.Image {
    entries, err := assets.ReadDir(dir)
    if err != nil {
        panic(err)
    }
    var imgs []*ebiten.Image
    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        imgs = append(imgs, mustLoadImage(dir+"/"+e.Name()))
    }
    return imgs
}

func mustLoadFont(path string) font.Face {
    data, err := assets.ReadFile(path)
    if err != nil {
        panic(err)
    }
    tt, err := opentype.Parse(data)
    if err != nil {
        panic(err)
    }
    face, err := opentype.NewFace(tt, &opentype.FaceOptions{
        Size:    48,
        DPI:     72,
        Hinting: font.HintingVertical,
    })
    if err != nil {
        panic(err)
    }
    return face
}

func init() {
    rand.Seed(time.Now().UnixNano())
    PlayerSprite = mustLoadImage("assets/player.png")
    BulletSprite = mustLoadImage("assets/bullet.png")
    MeteorSprites = mustLoadImages("assets/meteors")
    ScoreFont = mustLoadFont("assets/font.ttf")
}

// Vector is a 2D point or direction.
// :contentReference[oaicite:1]{index=1}
type Vector struct {
    X, Y float64
}

func (v Vector) Len() float64 {
    return math.Hypot(v.X, v.Y)
}

func (v Vector) Normalize() Vector {
    l := v.Len()
    if l == 0 {
        return Vector{}
    }
    return Vector{X: v.X / l, Y: v.Y / l}
}

// Timer counts ticks until a given duration.
// :contentReference[oaicite:2]{index=2}
type Timer struct {
    currentTicks, targetTicks int
}

func NewTimer(d time.Duration) *Timer {
    ticks := int(d.Seconds() * float64(ebiten.TPS()))
    return &Timer{targetTicks: ticks}
}

func (t *Timer) Update() {
    if t.currentTicks < t.targetTicks {
        t.currentTicks++
    }
}

func (t *Timer) IsReady() bool {
    return t.currentTicks >= t.targetTicks
}

func (t *Timer) Reset() {
    t.currentTicks = 0
}

// Rect for simple AABB collision.
// :contentReference[oaicite:3]{index=3}
type Rect struct {
    X, Y, Width, Height float64
}

func NewRect(x, y, w, h float64) Rect { return Rect{x, y, w, h} }
func (r Rect) MaxX() float64         { return r.X + r.Width }
func (r Rect) MaxY() float64         { return r.Y + r.Height }
func (r Rect) Intersects(o Rect) bool {
    return r.X <= o.MaxX() &&
        o.X <= r.MaxX() &&
        r.Y <= o.MaxY() &&
        o.Y <= r.MaxY()
}

// Game is our Ebiten game state.
// :contentReference[oaicite:4]{index=4}
type Game struct {
    player           *Player
    meteorSpawnTimer *Timer
    meteors          []*Meteor
    bullets          []*Bullet
    score            int
}

func NewGame() *Game {
    g := &Game{
        meteors: make([]*Meteor, 0),
        bullets: make([]*Bullet, 0),
    }
    g.player = NewPlayer(g)
    g.meteorSpawnTimer = NewTimer(1 * time.Second)
    return g
}

func (g *Game) Update() error {
    g.player.Update()

    g.meteorSpawnTimer.Update()
    if g.meteorSpawnTimer.IsReady() {
        g.meteorSpawnTimer.Reset()
        g.meteors = append(g.meteors, NewMeteor())
    }

    for _, m := range g.meteors {
        m.Update()
    }
    for _, b := range g.bullets {
        b.Update()
    }

    // handle collisions
    for i := len(g.meteors) - 1; i >= 0; i-- {
        m := g.meteors[i]
        if m.Collider().Intersects(g.player.Collider()) {
            g.Reset()
            return nil
        }
        for j := len(g.bullets) - 1; j >= 0; j-- {
            b := g.bullets[j]
            if m.Collider().Intersects(b.Collider()) {
                // remove meteor
                g.meteors = append(g.meteors[:i], g.meteors[i+1:]...)
                // remove bullet
                g.bullets = append(g.bullets[:j], g.bullets[j+1:]...)
                g.score++
                break
            }
        }
    }

    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    g.player.Draw(screen)
    for _, m := range g.meteors {
        m.Draw(screen)
    }
    for _, b := range g.bullets {
        b.Draw(screen)
    }
    // draw score UI
    text.Draw(screen, fmt.Sprintf("%06d", g.score), ScoreFont,
        ScreenWidth/2-100, 50, color.White)
}

func (g *Game) Layout(outW, outH int) (int, int) {
    return ScreenWidth, ScreenHeight
}

func (g *Game) Reset() {
    g.player = NewPlayer(g)
    g.meteors = g.meteors[:0]
    g.bullets = g.bullets[:0]
    g.meteorSpawnTimer.Reset()
    g.score = 0
}

func (g *Game) AddBullet(b *Bullet) {
    g.bullets = append(g.bullets, b)
}

// Player represents the starship.
// :contentReference[oaicite:5]{index=5}
type Player struct {
    position      Vector
    rotation      float64
    sprite        *ebiten.Image
    game          *Game
    shootCooldown *Timer
}

func NewPlayer(g *Game) *Player {
    sprite := PlayerSprite
    b := sprite.Bounds()
    halfW := float64(b.Dx()) / 2
    halfH := float64(b.Dy()) / 2
    pos := Vector{
        X: ScreenWidth/2 - halfW,
        Y: ScreenHeight/2 - halfH,
    }
    return &Player{
        position:      pos,
        sprite:        sprite,
        game:          g,
        shootCooldown: NewTimer(300 * time.Millisecond),
    }
}

func (p *Player) Update() {
    // rotate
    rotSpeed := math.Pi / float64(ebiten.TPS())
    if ebiten.IsKeyPressed(ebiten.KeyLeft) {
        p.rotation -= rotSpeed
    }
    if ebiten.IsKeyPressed(ebiten.KeyRight) {
        p.rotation += rotSpeed
    }
    // thrust
    speed := 2.0
    if ebiten.IsKeyPressed(ebiten.KeyUp) {
        p.position.X += math.Sin(p.rotation) * speed
        p.position.Y -= math.Cos(p.rotation) * speed
    }
    if ebiten.IsKeyPressed(ebiten.KeyDown) {
        p.position.X -= math.Sin(p.rotation) * speed
        p.position.Y += math.Cos(p.rotation) * speed
    }
    // shoot
    p.shootCooldown.Update()
    if p.shootCooldown.IsReady() && ebiten.IsKeyPressed(ebiten.KeySpace) {
        p.shootCooldown.Reset()
        b := p.sprite.Bounds()
        halfW := float64(b.Dx()) / 2
        halfH := float64(b.Dy()) / 2
        offset := 30.0
        spawn := Vector{
            X: p.position.X + halfW + math.Sin(p.rotation)*offset,
            Y: p.position.Y + halfH - math.Cos(p.rotation)*offset,
        }
        p.game.AddBullet(NewBullet(spawn, p.rotation))
    }
}

func (p *Player) Draw(screen *ebiten.Image) {
    b := p.sprite.Bounds()
    halfW := float64(b.Dx()) / 2
    halfH := float64(b.Dy()) / 2
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(-halfW, -halfH)
    op.GeoM.Rotate(p.rotation)
    op.GeoM.Translate(halfW, halfH)
    op.GeoM.Translate(p.position.X, p.position.Y)
    screen.DrawImage(p.sprite, op)
}

func (p *Player) Collider() Rect {
    b := p.sprite.Bounds()
    return NewRect(p.position.X, p.position.Y,
        float64(b.Dx()), float64(b.Dy()))
}

// Meteor storms in toward the center.
// :contentReference[oaicite:6]{index=6}
type Meteor struct {
    position      Vector
    movement      Vector
    rotation      float64
    rotationSpeed float64
    sprite        *ebiten.Image
}

func NewMeteor() *Meteor {
    sprite := MeteorSprites[rand.Intn(len(MeteorSprites))]
    // spawn on circle around center
    angle := rand.Float64() * 2 * math.Pi
    r := float64(ScreenWidth) / 2
    x := ScreenWidth/2 + math.Cos(angle)*r
    y := ScreenHeight/2 + math.Sin(angle)*r
    dir := Vector{X: ScreenWidth/2 - x, Y: ScreenHeight/2 - y}.Normalize()
    vel := 0.5 + rand.Float64()*1.5
    return &Meteor{
        position:      Vector{X: x, Y: y},
        movement:      Vector{X: dir.X * vel, Y: dir.Y * vel},
        rotationSpeed: -0.02 + rand.Float64()*0.04,
        sprite:        sprite,
    }
}

func (m *Meteor) Update() {
    m.position.X += m.movement.X
    m.position.Y += m.movement.Y
    m.rotation += m.rotationSpeed
}

func (m *Meteor) Draw(screen *ebiten.Image) {
    b := m.sprite.Bounds()
    halfW := float64(b.Dx()) / 2
    halfH := float64(b.Dy()) / 2
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(-halfW, -halfH)
    op.GeoM.Rotate(m.rotation)
    op.GeoM.Translate(halfW, halfH)
    op.GeoM.Translate(m.position.X, m.position.Y)
    screen.DrawImage(m.sprite, op)
}

func (m *Meteor) Collider() Rect {
    b := m.sprite.Bounds()
    return NewRect(m.position.X, m.position.Y,
        float64(b.Dx()), float64(b.Dy()))
}

// Bullet flies straight out.
// :contentReference[oaicite:7]{index=7}
type Bullet struct {
    position Vector
    movement Vector
    sprite   *ebiten.Image
}

func NewBullet(pos Vector, rot float64) *Bullet {
    vel := 5.0
    return &Bullet{
        position: pos,
        movement: Vector{X: math.Sin(rot) * vel, Y: -math.Cos(rot) * vel},
        sprite:   BulletSprite,
    }
}

func (b *Bullet) Update() {
    b.position.X += b.movement.X
    b.position.Y += b.movement.Y
}

func (b *Bullet) Draw(screen *ebiten.Image) {
    op := &ebiten.DrawImageOptions{}
    op.GeoM.Translate(b.position.X, b.position.Y)
    screen.DrawImage(b.sprite, op)
}

func (b *Bullet) Collider() Rect {
    bb := b.sprite.Bounds()
    return NewRect(b.position.X, b.position.Y,
        float64(bb.Dx()), float64(bb.Dy()))
}

func main() {
    ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
    ebiten.SetWindowTitle("Starship")
    if err := ebiten.RunGame(NewGame()); err != nil {
        panic(err)
    }
}
