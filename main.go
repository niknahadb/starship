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

//go:embed assets
var assets embed.FS

var (
    PlayerSprite  *ebiten.Image
    MeteorSprites []*ebiten.Image
    BulletSprite  *ebiten.Image
    ScoreFont     font.Face
    GameOverFont  font.Face  // new: larger font for game over text
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
    // Load game over font with larger size
    data, err := assets.ReadFile("assets/font.ttf")
    if err != nil {
        panic(err)
    }
    tt, err := opentype.Parse(data)
    if err != nil {
        panic(err)
    }
    GameOverFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
        Size:    72,
        DPI:     72,
        Hinting: font.HintingVertical,
    })
    if err != nil {
        panic(err)
    }
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
    difficultyLevel  int
    isGameOver       bool    // new: track game over state
    rainbowHue       float64 // new: for rainbow animation
}

func NewGame() *Game {
    g := &Game{
        meteors: make([]*Meteor, 0),
        bullets: make([]*Bullet, 0),
    }
    g.player = NewPlayer(g, 0)
    g.meteorSpawnTimer = NewTimer(1 * time.Second)
    return g
}

func (g *Game) Update() error {
    if g.isGameOver {
        // Update rainbow hue
        g.rainbowHue += 0.01
        if g.rainbowHue > 1.0 {
            g.rainbowHue = 0
        }
        
        // Check for space key to restart
        if ebiten.IsKeyPressed(ebiten.KeySpace) {
            g.Reset()
        }
        return nil
    }

    g.difficultyLevel = g.score / 10
    g.player.Update(g.difficultyLevel)

    g.meteorSpawnTimer.Update()
    if g.meteorSpawnTimer.IsReady() {
        g.meteorSpawnTimer.Reset()
        g.meteors = append(g.meteors, NewMeteor(g.difficultyLevel))
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
            g.isGameOver = true
            return nil
        }
        for j := len(g.bullets) - 1; j >= 0; j-- {
            b := g.bullets[j]
            if m.Collider().Intersects(b.Collider()) {
                g.meteors = append(g.meteors[:i], g.meteors[i+1:]...)
                g.bullets = append(g.bullets[:j], g.bullets[j+1:]...)
                g.score++
                break
            }
        }
    }

    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    if g.isGameOver {
        // Draw game over screen
        // Game Over text
        text.Draw(screen, "GAME OVER", GameOverFont,
            ScreenWidth/2-200, ScreenHeight/2-50, color.White)
        
        // Score text
        text.Draw(screen, fmt.Sprintf("Score: %06d", g.score), ScoreFont,
            ScreenWidth/2-100, ScreenHeight/2, color.White)
        
        // Rainbow "Try Again" text
        r, g, b := hsvToRGB(g.rainbowHue, 1.0, 1.0)
        rainbowColor := color.RGBA{r, g, b, 255}
        
        // Draw Try Again text with rainbow color
        text.Draw(screen, "Try Again", ScoreFont,
            ScreenWidth/2-80, ScreenHeight*3/4, rainbowColor)
        return
    }

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
    g.player = NewPlayer(g, 0)
    g.meteors = g.meteors[:0]
    g.bullets = g.bullets[:0]
    g.meteorSpawnTimer.Reset()
    g.score = 0
    g.difficultyLevel = 0
    g.isGameOver = false
    g.rainbowHue = 0
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
    baseRotSpeed  float64
    baseCooldown  time.Duration
}

func NewPlayer(g *Game, difficulty int) *Player {
    sprite := PlayerSprite
    b := sprite.Bounds()
    halfW := float64(b.Dx()) / 2
    halfH := float64(b.Dy()) / 2
    pos := Vector{
        X: ScreenWidth/2 - halfW,
        Y: ScreenHeight/2 - halfH,
    }
    baseRotSpeed := math.Pi / float64(ebiten.TPS())
    baseCooldown := 300 * time.Millisecond
    cooldown := baseCooldown
    if difficulty > 0 {
        cooldown = time.Duration(float64(baseCooldown) * math.Pow(0.92, float64(difficulty)))
    }
    return &Player{
        position:      pos,
        sprite:        sprite,
        game:          g,
        shootCooldown: NewTimer(cooldown),
        baseRotSpeed:  baseRotSpeed,
        baseCooldown:  baseCooldown,
    }
}

func (p *Player) Update(difficulty int) {
    rotSpeed := p.baseRotSpeed * (1 + 0.08*float64(difficulty))
    if ebiten.IsKeyPressed(ebiten.KeyLeft) {
        p.rotation -= rotSpeed
    }
    if ebiten.IsKeyPressed(ebiten.KeyRight) {
        p.rotation += rotSpeed
    }
    // thrust
    baseSpeed := 2.0
    speed := baseSpeed * (1 + 0.05*float64(difficulty))
    if ebiten.IsKeyPressed(ebiten.KeyUp) {
        p.position.X += math.Sin(p.rotation) * speed
        p.position.Y -= math.Cos(p.rotation) * speed
    }
    if ebiten.IsKeyPressed(ebiten.KeyDown) {
        p.position.X -= math.Sin(p.rotation) * speed
        p.position.Y += math.Cos(p.rotation) * speed
    }
    // shoot
    newCooldown := p.baseCooldown
    if difficulty > 0 {
        newCooldown = time.Duration(float64(p.baseCooldown) * math.Pow(0.92, float64(difficulty)))
    }
    p.shootCooldown.targetTicks = int(newCooldown.Seconds() * float64(ebiten.TPS()))
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

func NewMeteor(difficulty int) *Meteor {
    sprite := MeteorSprites[rand.Intn(len(MeteorSprites))]
    // spawn on circle around center
    angle := rand.Float64() * 2 * math.Pi
    r := float64(ScreenWidth) / 2
    x := ScreenWidth/2 + math.Cos(angle)*r
    y := ScreenHeight/2 + math.Sin(angle)*r
    dir := Vector{X: ScreenWidth/2 - x, Y: ScreenHeight/2 - y}.Normalize()
    baseVel := 0.3 + rand.Float64()*0.7
    vel := baseVel * (1 + 0.12*float64(difficulty))
    return &Meteor{
        position:      Vector{X: x, Y: y},
        movement:      Vector{X: dir.X * vel, Y: dir.Y * vel},
        rotationSpeed: (-0.02 + rand.Float64()*0.04) * (1 + 0.10*float64(difficulty)),
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
    op.GeoM.Scale(0.5, 0.5)
    op.GeoM.Translate(b.position.X, b.position.Y)
    screen.DrawImage(b.sprite, op)
}

func (b *Bullet) Collider() Rect {
    bb := b.sprite.Bounds()
    return NewRect(b.position.X, b.position.Y,
        float64(bb.Dx()), float64(bb.Dy()))
}

// Helper function to convert HSV to RGB
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
    var r, g, b float64
    
    i := math.Floor(h * 6)
    f := h*6 - i
    p := v * (1 - s)
    q := v * (1 - f*s)
    t := v * (1 - (1-f)*s)
    
    switch int(i) % 6 {
    case 0:
        r, g, b = v, t, p
    case 1:
        r, g, b = q, v, p
    case 2:
        r, g, b = p, v, t
    case 3:
        r, g, b = p, q, v
    case 4:
        r, g, b = t, p, v
    case 5:
        r, g, b = v, p, q
    }
    
    return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

func main() {
    ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
    ebiten.SetWindowTitle("Starship")
    if err := ebiten.RunGame(NewGame()); err != nil {
        panic(err)
    }
}

