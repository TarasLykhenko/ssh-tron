package tron

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jpillora/ansi"
)

const slotHeight = 4

var (
	filled = '⣿'
	top    = '⠛'
	bottom = '⣤'
	empty  = ' '
)

type Direction byte

const (
	dup Direction = iota + 65
	ddown
	dright
	dleft
)

func (d Direction) String() string {
	switch d {
	case dup:
		return "up"
	case ddown:
		return "down"
	case dleft:
		return "left"
	case dright:
		return "right"
	default:
		return fmt.Sprintf("%d", d)
	}
}

var colours = map[ID][]byte{
	blank: ansi.Set(ansi.White),
	wall:  ansi.Set(ansi.White),
	ID(1): ansi.Set(ansi.Blue),
	ID(2): ansi.Set(ansi.Green),
	ID(3): ansi.Set(ansi.Magenta),
	ID(4): ansi.Set(ansi.Cyan),
	ID(5): ansi.Set(ansi.Yellow),
	ID(6): ansi.Set(ansi.Red),
}

// A Player represents a live TCP connection from a client
type Player struct {
	id                   ID // identification
	Name                 string
	hash                 string //hash of public key
	rank, index          int
	x, y                 uint8     // position
	d                    Direction // curr direction
	nextd                Direction // next direction
	score                [slotHeight]string
	dead, ready, waiting bool
	tdeath               time.Time // time of death
	Kills, Deaths        int       // score
	playing              chan bool // is playing signal
	g                    *Game
	logf                 func(format string, args ...interface{})
}

// NewPlayer returns an initialized Player.
func NewPlayer(id ID, name string) *Player {
	colouredName := fmt.Sprintf("%s%s%s", colours[id], name, ansi.Set(ansi.Reset))
	p := &Player{
		id:      id,
		Name:    name,
		hash:    name,
		d:       dup,
		dead:    true,
		ready:   false,
		playing: make(chan bool, 1),
		logf:    log.New(os.Stdout, colouredName+" ", 0).Printf,
	}
	return p
}

const (
	respawnAttempts  = 100
	respawnLookahead = 15
)

func (p *Player) respawn() {
	// if !p.dead || !p.ready || p.waiting {
	// 	return
	// }

	if !p.dead {
		return
	}
	for i := 0; i < respawnAttempts; i++ {
		// randomly spawn player
		p.x = uint8(rand.Intn(int(p.g.bw-2))) + 1
		p.y = uint8(rand.Intn(int(p.g.bh-2))) + 1
		p.d = Direction(uint8(rand.Intn(4) + 65))
		p.nextd = p.d
		// look ahead
		clear := true
		x, y := p.x, p.y
		for j := 0; j < respawnLookahead; j++ {
			switch p.d {
			case dup:
				y--
			case ddown:
				y++
			case dleft:
				x--
			case dright:
				x++
			}
			if p.g.board[x][y] != blank {
				clear = false
				break
			}
		}
		// when clear, mark player as alive
		if clear {
			p.dead = false
			break
		}
	}
}

func (p *Player) status() string {
	if !p.ready {
		return "not ready"
	} else if p.dead && p.waiting {
		return fmt.Sprintf("dead %1.1f", (p.g.RespawnDelay - time.Since(p.tdeath)).Seconds())
	} else if p.dead {
		return "ready"
	}
	return "playing"
}

func (p *Player) recieveActions() {

}
