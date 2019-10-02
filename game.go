/**
 * Copyright (c) 2018, Andy Janata
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without modification, are permitted
 * provided that the following conditions are met:
 *
 * * Redistributions of source code must retain the above copyright notice, this list of conditions
 *   and the following disclaimer.
 * * Redistributions in binary form must reproduce the above copyright notice, this list of
 *   conditions and the following disclaimer in the documentation and/or other materials provided
 *   with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR
 * IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND
 * FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR
 * CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
 * WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY
 * WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var getGameRoundsStmt *sql.Stmt

type (
	// RoundMeta holds meta information about the round
	RoundMeta struct {
		RoundID   string
		Timestamp int64
		BlackCard Card
	}

	// GameMeta holds meta information about the game
	GameMeta struct {
		GameID    string
		Timestamp int64
	}

	gameHandler struct{}
)

// FormattedTimestamp will format the unix timestamp to a RFC1123 formatted string
func (round *RoundMeta) FormattedTimestamp() string {
	return time.Unix(round.Timestamp, 0).UTC().Format(time.RFC1123)
}

// FormattedTimestamp will format the unix timestamp to a RFC1123 formatted string
func (game *GameMeta) FormattedTimestamp() string {
	return time.Unix(game.Timestamp, 0).UTC().Format(time.RFC1123)
}

func init() {
	log.Debug("Registering game handler")
	registerHandler(gameHandler{})
}

func (h gameHandler) registerEndpoints(r *gin.Engine) {
	log.Debug("Registering endpoint for game handler")
	r.GET("/game/:id", getGame)
}

func (h gameHandler) prepareStatements(db *sql.DB) error {
	log.Debug("Preparing statements for game handler")
	var err error
	getGameRoundsStmt, err = db.Prepare("SELECT bc.text, bc.watermark, bc.pick, bc.draw, rc.round_id, ((rc.meta).timestamp AT TIME ZONE 'UTC') " +
		"FROM round_complete rc " +
		"JOIN black_card bc ON bc.uid = rc.black_card_uid " +
		"WHERE rc.game_id = $1 " +
		"ORDER BY ((rc.meta).timestamp) DESC")
	return err
}

func getGame(c *gin.Context) {
	q, err := getGameRoundsStmt.Query(c.Param("id"))
	if err != nil {
		returnError(c, 500, fmt.Sprintf("Unable to query for game id %s: %v", c.Param("id"), err))
		return
	}
	defer q.Close()
	rounds := []RoundMeta{}
	for q.Next() {
		var (
			text      string
			watermark string
			pick      int16
			draw      int16
			roundID   string
			timestamp time.Time
		)
		q.Scan(&text, &watermark, &pick, &draw, &roundID, &timestamp)
		rounds = append(rounds, RoundMeta{
			BlackCard: Card{
				Text:      text,
				Watermark: watermark,
				Meta: CardMeta{
					Color: "black",
					Draw:  draw,
					Pick:  pick,
				},
			},
			RoundID:   roundID,
			Timestamp: timestamp.Unix(),
		})
	}
	if q.Err() != nil {
		log.Errorf("Error while iterating over rounds for game %s: %+v", c.Param("id"), q.Err())
	}
	if strings.Contains(c.Request.Header.Get("Accept"), "text/html") {
		c.HTML(200, "game", &rounds)
	} else {
		c.JSON(200, rounds)
	}
}
