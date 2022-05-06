package addon

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// https://wowpedia.fandom.com/wiki/Enum.GarrisonFollowerType
type GarrisonFollowerType int

const (
	// Garrison follower type.
	FollowerType_6_0 GarrisonFollowerType = 1 //nolint:golint,revive
	// Shipyard follower type.
	FollowerType_6_2 GarrisonFollowerType = 2 //nolint:golint,revive
	// Legion follower type.
	FollowerType_7_0 GarrisonFollowerType = 4 //nolint:golint,revive
	// Battle for Azeroth follower type.
	FollowerType_8_0 GarrisonFollowerType = 22 //nolint:golint,revive
	// Shadowlands follower type.
	FollowerType_9_0 GarrisonFollowerType = 123 //nolint:golint,revive

	MaxShowNextComplete int = 3 // max number of missions to show in list of next to complete
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

type EncounterIconInfo struct {
	IsElite            bool
	IsRare             bool
	MissionScalar      int
	PortraitFileDataID int
}

type FollowerInfo struct {
	FollowerTypeID GarrisonFollowerType
	Health         int
	IsAutoTroop    bool
	IsSoulbind     bool
	Level          int
	LevelXP        int
	MaxHealth      int
	Name           string
	Role           int
	Xp             int
}

type MissionDetail struct {
	BaseCost            int
	CharText            string
	Cost                int
	CostCurrencyTypesID int
	DurationSeconds     int
	EncounterIconInfo   *EncounterIconInfo
	Followers           []string
	FollowerInfo        map[string]*FollowerInfo
	FollowerTypeID      GarrisonFollowerType // Enum.GarrisonFollowerType
	InProgress          bool
	MissionEndTime      int64
	MissionID           int
	MissionScalar       int
	Name                string
	Rewards             []*json.RawMessage // one of a few reward types -- currency, item, xp
	Type                string
	Xp                  int
}

func (m *MissionDetail) IsComplete() bool {
	return (m.MissionEndTime - time.Now().Unix()) < 0
}

const (
	AlertTime = time.Minute * 30
	WarnTime  = time.Hour * 1
)

func (m *MissionDetail) TimeRemaining() (s string) {
	t := time.Until(time.Unix(m.MissionEndTime, 0)).Truncate(time.Second)
	s = formatDuration(t)

	if t < AlertTime {
		s = color.RedString("%11s", s)
	} else if t < WarnTime {
		s = color.YellowString("%11s", s)
	}

	return s
}

type AdventureTable struct {
	Type      GarrisonFollowerType
	Followers map[string]*FollowerInfo `json:",omitempty"` //nolint:tagliatelle
	Missions  []*MissionDetail
}

func (t *AdventureTable) CompanionsOnMission(m *MissionDetail) []*FollowerInfo {
	companions := make([]*FollowerInfo, 0, len(m.Followers))

	for _, f := range m.Followers {
		if c, ok := t.Followers[f]; ok && !c.IsAutoTroop {
			companions = append(companions, c)
		}
	}

	return companions
}

func (t *AdventureTable) Companions() []*FollowerInfo {
	companions := make([]*FollowerInfo, 0, len(t.Followers))

	for _, f := range t.Followers {
		if !f.IsAutoTroop {
			companions = append(companions, f)
		}
	}

	return companions
}

func (t *AdventureTable) NumCompanions() int {
	var c int

	for _, info := range t.Followers {
		if !info.IsAutoTroop {
			c++
		}
	}

	return c
}

func (t *AdventureTable) ActiveCompanions() []*FollowerInfo {
	companions := make([]*FollowerInfo, 0, len(t.Followers))
	for _, m := range t.Missions {
		companions = append(companions, t.CompanionsOnMission(m)...)
	}

	return companions
}

func (t *AdventureTable) IdleCompanions() []*FollowerInfo {
	idle := make([]*FollowerInfo, 0, len(t.Followers))
	active := t.ActiveCompanions()
	for _, info := range t.Followers {
		if !info.IsAutoTroop && !slices.Contains(active, info) {
			idle = append(idle, info)
		}
	}

	return idle
}

func (t *AdventureTable) MissionsActive() []*MissionDetail {
	return t.Missions
}

func (t *AdventureTable) MissionsComplete() []*MissionDetail {
	completed := make([]*MissionDetail, 0, len(t.Missions))

	for _, m := range t.Missions {
		if m.IsComplete() {
			completed = append(completed, m)
		}
	}

	return completed
}

type AdventureTables map[string]*AdventureTable

type Character struct {
	Key             string
	Class           string
	Gender          string
	LastSeen        int
	Level           int
	Money           int
	MoneyText       string
	Name            string
	PlayedLevel     int
	PlayedTotal     int
	Race            string
	Realm           string
	ReservoirAnima  int
	AdventureTables AdventureTables
}

func (char *Character) Table(followerType GarrisonFollowerType) *AdventureTable {
	table, hasTable := char.AdventureTables[strconv.Itoa(int(followerType))]
	if !hasTable {
		return nil
	}

	return table
}

func (char *Character) String() string {
	return fmt.Sprintf("%s-%s", char.Name, char.Realm)
}

type Data struct {
	Characters map[string]*Character
}

func (d *Data) CharacterKeys() []string {
	chars := maps.Keys(d.Characters)
	sort.Strings(chars)

	return chars
}

func (d *Data) NumMissionsActive(followerType GarrisonFollowerType) int {
	var active int

	for _, char := range d.Characters {
		table := char.Table(followerType)
		if table == nil {
			continue
		}

		active += len(table.MissionsActive())
	}

	return active
}

func (d *Data) NumMissionsComplete(followerType GarrisonFollowerType) int {
	var complete int

	for _, char := range d.Characters {
		table := char.Table(followerType)
		if table == nil {
			continue
		}

		complete += len(table.MissionsComplete())
	}

	return complete
}

func (d *Data) Print() {
	log.Printf("%d characters, %d complete / %d active\n",
		len(d.Characters),
		d.NumMissionsComplete(FollowerType_9_0),
		d.NumMissionsActive(FollowerType_9_0))

	for _, key := range d.CharacterKeys() {
		char := d.Characters[key]
		table := char.Table(FollowerType_9_0)
		if table == nil || len(table.Followers) == 0 {
			continue
		}

		missions := table.MissionsActive()
		log.Printf("\t%-30s M(%-2s / %2d) F(%-2s / %2d)\n",
			color.CyanString("%-30s", char.String()),
			color.GreenString("%2d", len(table.MissionsComplete())),
			len(missions),
			color.YellowString("%2d", len(table.IdleCompanions())),
			table.NumCompanions())

		sort.Slice(missions, func(i, j int) bool { return missions[i].MissionEndTime < missions[j].MissionEndTime })
		var shown int
		for _, m := range missions {
			// skip completed missions
			if m.IsComplete() {
				continue
			}

			log.Printf("\t\t- %11s    (%d) [%2d] %-20s\n",
				m.TimeRemaining(),
				len(table.CompanionsOnMission(m)),
				m.MissionScalar,
				m.Name)

			shown++
			// break loop if we're at max list length
			if shown >= MaxShowNextComplete {
				break
			}
		}
	}
}

// formatDuration returns a string in HHh:MMm:SSs format.
func formatDuration(d time.Duration) string {
	// use default duration format if only seconds
	if d < time.Minute {
		return d.String()
	}

	hasHours := false
	var b strings.Builder
	// hours
	if d >= time.Hour {
		hasHours = true
		hours := d / time.Hour
		fmt.Fprintf(&b, "%2dh:", int(hours))
		d -= hours * time.Hour
	}

	// minutes
	if d >= time.Minute || hasHours {
		// 0 pad minutes only if preceded by hours
		var fs string
		if hasHours {
			fs = "%02dm:"
		} else {
			fs = "%2dm:"
		}

		mins := d / time.Minute
		fmt.Fprintf(&b, fs, int(mins))
		d -= mins * time.Minute
	}

	// seconds
	fmt.Fprintf(&b, "%03s", d)

	return b.String()
}
