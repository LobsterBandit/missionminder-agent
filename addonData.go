package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"golang.org/x/exp/maps"
)

// https://wowpedia.fandom.com/wiki/Enum.GarrisonFollowerType
type GarrisonFollowerType int

const (
	FollowerType_6_0 GarrisonFollowerType = 1   // Garrison
	FollowerType_6_2 GarrisonFollowerType = 2   // Shipyard
	FollowerType_7_0 GarrisonFollowerType = 4   // Legion
	FollowerType_8_0 GarrisonFollowerType = 22  // Battle for Azeroth
	FollowerType_9_0 GarrisonFollowerType = 123 // Shadowlands

	// max number of missions to show in list of next to complete
	MAX_SHOW_NEXT_COMPLETE int = 3
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

type CharacterDetail struct {
	Key         string
	Class       string
	Gender      string
	LastSeen    int
	Level       int
	Name        string
	PlayedLevel int
	PlayedTotal int
	Race        string
	Realm       string
}

type EncounterIconInfo struct {
	IsElite            bool
	IsRare             bool
	MissionScalar      int
	PortraitFileDataID int
}

type FollowerInfo struct {
	IsAutoTroop    bool
	FollowerTypeID GarrisonFollowerType
	Xp             int
	Role           int
	Health         int
	LevelXP        int
	Name           string
	Level          int
	MaxHealth      int
	IsSoulbind     bool
}

type MissionDetail struct {
	Cost                int
	EncounterIconInfo   *EncounterIconInfo
	Followers           []string
	InProgress          bool
	Xp                  int
	MissionEndTime      int64
	MissionID           int
	FollowerTypeID      GarrisonFollowerType // Enum.GarrisonFollowerType
	Name                string
	MissionScalar       int
	DurationSeconds     int
	CostCurrencyTypesID int
	CharText            string
	Rewards             []*json.RawMessage // one of a few reward types -- currency, item, xp
	Type                string
	FollowerInfo        map[string]*FollowerInfo
	BaseCost            int
}

func (m *MissionDetail) IsComplete() bool {
	return (m.MissionEndTime - time.Now().Unix()) < 0
}

func (m *MissionDetail) TimeRemaining() (s string) {
	t := time.Until(time.Unix(m.MissionEndTime, 0)).Truncate(time.Second)
	s = time.Unix(0, 0).UTC().Add(t).Format("15h:04m:05s")

	if t.Minutes() < 30.0 {
		s = fmt.Sprintf("\033[32m%s\033[0m", s)
	} else if t.Hours() < 1.0 {
		s = fmt.Sprintf("\033[33m%s\033[0m", s)
	}

	return s
}

func (m *MissionDetail) Companions() []*FollowerInfo {
	companions := make([]*FollowerInfo, 0, len(m.FollowerInfo))
	for _, info := range m.FollowerInfo {
		if !info.IsAutoTroop {
			companions = append(companions, info)
		}
	}
	return companions
}

type AddonData struct {
	Characters map[string]*CharacterDetail
	Missions   map[string][]*MissionDetail
}

func (ad *AddonData) characterKeys() []string {
	chars := maps.Keys(ad.Characters)
	sort.Strings(chars)
	return chars
}

func (ad *AddonData) getMissions(char *CharacterDetail) []*MissionDetail {
	return ad.Missions[missionKey(char)]
}

func (ad *AddonData) getMissionsOfType(char *CharacterDetail, missionType GarrisonFollowerType) []*MissionDetail {
	missions := make([]*MissionDetail, 0)
	for _, m := range ad.Missions[missionKey(char)] {
		if m.FollowerTypeID == missionType {
			missions = append(missions, m)
		}
	}
	return missions
}

func (ad *AddonData) missionsActive(char *CharacterDetail) []*MissionDetail {
	return ad.getMissionsOfType(char, FollowerType_9_0)
}

func (ad *AddonData) missionsComplete(char *CharacterDetail) []*MissionDetail {
	complete := make([]*MissionDetail, 0)
	for _, m := range ad.getMissionsOfType(char, FollowerType_9_0) {
		if m.IsComplete() {
			complete = append(complete, m)
		}
	}
	return complete
}

func (ad *AddonData) totalMissionsActive() int {
	var active int
	for _, char := range ad.Characters {
		active += len(ad.missionsActive(char))
	}
	return active
}

func (ad *AddonData) totalMissionsComplete() int {
	var complete int
	for _, char := range ad.Characters {
		complete += len(ad.missionsComplete(char))
	}
	return complete
}

func (ad *AddonData) print() {
	log.Printf("%d characters, %d complete / %d active\n",
		len(ad.Characters),
		ad.totalMissionsComplete(),
		ad.totalMissionsActive())

	for _, key := range ad.characterKeys() {
		char := ad.Characters[key]
		if char.Level != 60 {
			continue
		}
		missions := ad.missionsActive(char)
		log.Printf("\t%-40s (%2d / %2d)\n",
			key,
			len(ad.missionsComplete(char)),
			len(missions))

		sort.Slice(missions, func(i, j int) bool { return missions[i].MissionEndTime < missions[j].MissionEndTime })
		var shown int
		for _, m := range missions {
			// skip completed missions
			if m.IsComplete() {
				continue
			}

			log.Printf("\t\t- %11s    (%d) [%2d] %-20s\n",
				m.TimeRemaining(),
				len(m.Companions()),
				m.MissionScalar,
				m.Name)

			shown++
			// break loop if we're at max list length
			if shown >= MAX_SHOW_NEXT_COMPLETE {
				break
			}
		}
	}
}

// account.realm.character
func characterKey(char *CharacterDetail) string {
	return fmt.Sprintf("%s.%s.%s", "Default", char.Realm, char.Name)
}

// character-realm
func missionKey(char *CharacterDetail) string {
	return fmt.Sprintf("%s-%s", char.Name, char.Realm)
}
