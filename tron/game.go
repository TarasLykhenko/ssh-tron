package tron

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/golang/freetype/truetype"
	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
)

type ID uint16

type Game struct {
	Config
	w, h, bw, bh int         // total score+board size
	db           *Database   // database
	score        *scoreboard // state
	board        Board
	allPlayers   map[string]*Player
	currPlayers  map[ID]*Player
	playerColors []colorful.Color
	gameWindow   *pixelgl.Window
	txtRefresher *text.Text

	logf func(format string, args ...interface{})
}

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

// NewGame returns an initialized Game according to the input arguments.
// The main() function should call the Play() method on this Game.
func NewGame(c Config) (*Game, error) {
	if c.Height < 32 || c.Height > 255 {
		return nil, errors.New("height must be between 32-256")
	}
	if c.Width < 32 || c.Width > 255 {
		return nil, errors.New("width must be between 32-256")
	}
	db, err := NewDatabase(c.DBLocation, c.DBReset)
	if err != nil {
		return nil, err
	}
	board, err := NewBoard(uint8(c.Width), uint8(c.Height))
	if err != nil {
		return nil, err
	}

	g := &Game{
		Config:       c,
		w:            c.Width + sidebarWidth,
		h:            c.Height / 2,
		bw:           c.Height,
		bh:           c.Width,
		db:           db,
		board:        board,
		playerColors: colorful.FastHappyPalette(10),
		allPlayers:   make(map[string]*Player),
		currPlayers:  make(map[ID]*Player),
		gameWindow:   c.GameWindow,
		logf:         log.New(os.Stdout, "tron: ", 0).Printf,
	}
	g.score = &scoreboard{g: g}
	//load initial player list
	prevPlayers, err := g.db.loadAll()
	if err != nil {
		return nil, errors.New("Failed to restore player list")
	}
	for _, p := range prevPlayers {
		g.allPlayers[p.hash] = p
	}

	fontFace, _ := loadTTF("./font/MesloLGSNFRegular.ttf", 20)

	atlas := text.NewAtlas(fontFace, text.ASCII, []rune{filled, top, bottom, empty})

	txt := text.New(pixel.V(float64(g.w), 670), atlas)

	txt.Color = colornames.Lightgrey

	g.txtRefresher = txt
	//compute initial score
	g.score.compute()
	//game ready
	return g, nil
}

func (g *Game) Play() {
	// build walls
	for w := 0; w < g.bw; w++ {
		g.board[w][0] = wall
		g.board[w][g.bh-1] = wall
	}
	for h := 0; h < g.bh; h++ {
		g.board[0][h] = wall
		g.board[g.bw-1][h] = wall
	}

	// start the game ticker!
	g.tick()
}

func (g *Game) death(p *Player) {
	p.Deaths++
	g.score.compute()
	go g.db.save(p) //save new death count
	p.tdeath = time.Now()
	g.remove(p)
}

//time to keep players trail around after death
var deathTrail = 1 * time.Second

func (g *Game) remove(p *Player) {
	p.waiting = true
	//respawn/deathtrail time
	if g.RespawnDelay > deathTrail {
		time.Sleep(deathTrail)
	} else {
		time.Sleep(g.RespawnDelay)
	}
	// clear this player off the board!
	for w := 0; w < g.bw; w++ {
		for h := 0; h < g.bh; h++ {
			if g.board[w][h] == p.id {
				g.board[w][h] = blank
			}
		}
	}
	//respawn extra
	if g.RespawnDelay > deathTrail {
		time.Sleep(g.RespawnDelay - deathTrail)
	}
	p.waiting = false
}

func (g *Game) AddPlayer(p *Player) {
	// attempt to load previous scores
	if err := g.db.load(p); err != nil {
		//otherwise new player
		g.db.save(p)
	}
	// connected with a valid id
	p.g = g
	p.respawn()
	g.allPlayers[p.hash] = p
	g.currPlayers[p.id] = p
	g.score.compute()
	// connected
}

func (g *Game) tick() {
	fps := time.Tick(time.Second / 16)

	// loop forever
	for !g.gameWindow.Closed() {
		g.gameWindow.Clear(colornames.Black)
		g.txtRefresher.Clear()

		// move each player 1 square
		for _, p := range g.currPlayers {
			// skip this player
			if p.dead {
				continue
			}

			if g.gameWindow.Pressed(pixelgl.KeyLeft) {
				p.d = dleft
			} else if g.gameWindow.Pressed(pixelgl.KeyRight) {
				p.d = dright
			} else if g.gameWindow.Pressed(pixelgl.KeyDown) {
				p.d = ddown
			} else if g.gameWindow.Pressed(pixelgl.KeyUp) {
				p.d = dup
			}
			// move player in [d]irection
			switch p.d {
			case dup:
				p.y--
			case ddown:
				p.y++
			case dleft:
				p.x--
			case dright:
				p.x++
			}
			// player is in a wall
			if g.board[p.x][p.y] != blank {
				// is it another player's wall? kills++
				id := g.board[p.x][p.y]
				if other, ok := g.currPlayers[id]; ok && other != p {
					other.Kills++
					g.score.compute()
					go g.db.save(other) //save new kill count
					other.logf("killed %s", p.Name)
				}
				// this player dies...
				p.dead = true
				go g.death(p)
				continue
			}
			// place a player square
			g.board[p.x][p.y] = p.id
		}
		g.refreshScreen()
		// mark score as used
		g.score.changed = false
		// game sleep! (attempt to stablize game speed)
		<-fps
	}
}

func (g *Game) refreshScreen() {

	gb := g.board

	// score state
	totalPlayers := len(g.score.allPlayersSorted)
	startIndex := 0

	// store the last rendered for optimisation
	var r rune
	var c ID
	// screen loop
	for h := 0; h < g.h; h++ {
		for tw := 0; tw < g.w; tw++ {
			// each iteration draws rune (r) and color (c)
			// at terminal location: w x h
			r = empty
			c = blank
			// choose a rune to draw, either from
			// sidebar or from game board
			if tw < sidebarWidth {
				// pick rune from sidebar
				if tw == 0 {
					r = filled
				} else if h == 0 {
					r = top
				} else if h == g.h-1 {
					r = bottom
				} else {
					bh := h - 1 //borderless height
					playerSlot := bh / slotHeight
					playerIndex := startIndex + playerSlot
					if playerIndex < totalPlayers {
						sp := g.score.allPlayersSorted[playerIndex]
						line := bh % slotHeight
						if tw == 1 {
							switch line {
							case 0:
								sp.score[0] = fmt.Sprintf("%s            ", sp.Name)
							case 1:
								sp.score[1] = fmt.Sprintf("  rank  #%03d  ", sp.rank)
							case 2:
								sp.score[2] = fmt.Sprintf("  %s           ", sp.status())
							case 3:
								sp.score[3] = fmt.Sprintf("  kills %4d   ", sp.Kills)
							}
						}
						if tw-1 < len(sp.score[line]) {
							r = rune(sp.score[line][tw-1])
							c = sp.id
						}
					}
				}
			} else {
				// pick rune from game board, one rune is two game tiles
				gw := tw - sidebarWidth
				h1 := h * 2
				h2 := h1 + 1
				// choose rune
				if gb[gw][h1] != blank && gb[gw][h2] != blank {
					r = filled
				} else if gb[gw][h1] != blank {
					r = top
				} else if gb[gw][h2] != blank {
					r = bottom
				}
				// choose color (use color of h1, otherwise h2)
				if gb[gw][h2] == blank {
					c = gb[gw][h1]
				} else {
					c = gb[gw][h2]
				}

			}
			if c != wall {
				g.txtRefresher.Color = g.playerColors[c]
			} else {
				g.txtRefresher.Color = g.playerColors[9]
			}
			g.txtRefresher.WriteRune(r)
		}
		g.txtRefresher.WriteRune('\n')

	}

	g.txtRefresher.Draw(g.gameWindow, pixel.IM)
	g.gameWindow.Update()

}
