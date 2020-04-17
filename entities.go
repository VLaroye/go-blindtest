package main

import (
	"math/rand"
	"sort"
	"time"

	"github.com/satori/go.uuid"
)

type Playlist struct {
	Songs []Song `json:"data"`
	Length int `json:"total"`
	// TODO: Think about other struct, 'next' is not meaningful here
	Next string `json:"next"`
}

func (p *Playlist) getRandomSong() Song {
	// Generate non-negative random number, between 0 and playlist's length + 1
	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(p.Length - 1)

	// Use random generated index to access a song in playlist
	return p.Songs[idx]
}

type Song struct {
	Id int `json:"id"`
	Preview string `json:"preview"`
	Artist Artist `json:"artist"`
	Title string `json:"title_short"`
}

type Artist struct {
	Name string `json:"name"`
	Picture string `json:"picture"`
}

type Game struct {
	Players []Player
	CurrentRound int
}

func newGame(players []Player) Game {
	return Game{ Players: players, CurrentRound: 0}
}

func (g *Game) restart() {
	g.CurrentRound = 1
	for _, v := range g.Players {
		v.resetScore()
	}
}

func (g *Game) join(player *Player) {
	g.Players = append(g.Players, *player)
}

func (g *Game) leave(player *Player) {
	tempPlayers := make([]Player, 0)

	for _, v := range g.Players {
		if v.ID != player.ID {
			tempPlayers = append(tempPlayers, v)
		}
	}

	g.Players = tempPlayers
}

func (g *Game) getLeaderBoard() *[]Player {
	leaderBoard := make([]Player, len(g.Players))
	copy(leaderBoard, g.Players)

	sort.SliceStable(leaderBoard, func(i, j int) bool {
		return leaderBoard[i].Score < leaderBoard[j].Score
	})

	return &leaderBoard
}

func (g *Game) getPlayerByID(id string) *Player {
	var foundPlayer Player
	for _, player := range g.Players {
		if player.ID.String() == id {
			foundPlayer = player
		}
	}
	return &foundPlayer
}

type Player struct {
	ID uuid.UUID
	Name string
	Score int
}

func newPlayer(name string) *Player {
	return &Player{
		ID: uuid.Must(uuid.NewV4(), nil),
		Name: name,
		Score: 0,
	}
}

func (p *Player) resetScore() {
	p.Score = 0
}

func (p *Player) increaseScore(points int) {
	p.Score += points
}

