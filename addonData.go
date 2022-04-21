package main

import (
	"encoding/json"

	"golang.org/x/exp/maps"
)

type XPReward struct {
	FollowerXP string
	Icon       string
	Name       string
	Title      string
	Tooltip    string
}

type ItemReward struct {
	ItemID   int
	ItemLink string
	Quantity int
}

type CurrencyReward struct {
	CurrencyID int
	Icon       string
	Quantity   int
	Title      string
}

type Mission struct {
	Cost            int
	DurationSeconds int
	ILevel          int
	Level           int
	Rewards         []*json.RawMessage
	SuccessChance   int
	Type            string
	TypeAtlas       string
}

type FollowerID int

type ActiveMission struct {
	Mission
	Followers     []FollowerID
	RemainingTime int
	StartTime     int
}

type MissionID string

type CharacterDetail struct {
	Key               string
	Class             string
	Gender            string
	LastSeen          int
	Level             int
	MissionsActive    map[MissionID]ActiveMission
	MissionsAvailable map[MissionID]Mission
	Name              string
	PlayedLevel       int
	PlayedTotal       int
	Race              string
	Realm             string
}

type CharacterKey string

type AddonData struct {
	Characters map[CharacterKey]CharacterDetail
}

func (ad *AddonData) characters() []CharacterKey {
	return maps.Keys(ad.Characters)
}

func (ad *AddonData) numMissionsActive() int {
	var active int
	for _, char := range ad.Characters {
		active += len(char.MissionsActive)
	}
	return active
}

func (ad *AddonData) numMissionsAvailable() int {
	var avail int
	for _, char := range ad.Characters {
		avail += len(char.MissionsAvailable)
	}
	return avail
}
