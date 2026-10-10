package main

import (
	"encoding/binary"
)

const (
	ActiveSlotsStatusOffset = 0x1901D04
	SummaryCardStartOffset  = 0x1901D0E
	SummaryCardLength       = 0x24C
	SummaryLevelOffset      = 0x22

	PlayerGameDataPadding = 8
	PlayerLevelOffset     = 0x2C
	PlayerStatsOffset     = 0x30
	PlayerRunesOffset     = 0x5C
	PlayerClassOffset     = 0xAF
)

type Character struct {
	Slot            int
	Name            string
	Level           uint32
	PlayerDataStart int
	StatsOffset     int
	CurrentStats    []uint32
	ClassByte       byte
	ClassName       string
	RealRunes       uint32
}

var classIDs = map[byte]string{
	0x00: "Vagabond", 0x01: "Warrior", 0x02: "Hero", 0x03: "Bandit",
	0x04: "Astrologer", 0x05: "Prophet", 0x06: "Confessor", 0x07: "Samurai",
	0x08: "Prisoner", 0x09: "Wretch",
}

var classBaseStats = map[string][]uint32{
	"Vagabond":   {15, 10, 11, 14, 13, 9, 9, 7},
	"Warrior":    {11, 12, 11, 10, 16, 10, 8, 9},
	"Hero":       {14, 9, 12, 16, 9, 7, 8, 11},
	"Bandit":     {10, 11, 10, 9, 13, 9, 8, 14},
	"Astrologer": {9, 15, 9, 8, 12, 16, 7, 9},
	"Prophet":    {10, 14, 8, 11, 10, 7, 16, 10},
	"Confessor":  {10, 13, 10, 12, 12, 9, 14, 9},
	"Samurai":    {12, 11, 13, 12, 15, 9, 8, 8},
	"Prisoner":   {11, 12, 11, 11, 14, 14, 6, 9},
	"Wretch":     {10, 10, 10, 10, 10, 10, 10, 10},
}

var statNames = []string{
	"Vigor",
	"Mind",
	"Endurance",
	"Strength",
	"Dexterity",
	"Intelligence",
	"Faith",
	"Arcane",
}

func (s *SaveFile) ScanCharacters() []Character {
	var activeCharacters []Character

	for slotIndex := 0; slotIndex < 10; slotIndex++ {
		if s.Raw[ActiveSlotsStatusOffset+slotIndex] != 1 {
			continue
		}

		cardOffset := SummaryCardStartOffset + (slotIndex * SummaryCardLength)
		rawNameBuffer := s.Raw[cardOffset : cardOffset+34]
		characterName := decodeUTF16LE(rawNameBuffer)
		menuLevel := binary.LittleEndian.Uint32(s.Raw[cardOffset+SummaryLevelOffset : cardOffset+SummaryLevelOffset+4])

		slotData, err := s.GetSlotData(slotIndex)
		if err != nil {
			continue
		}

		_, _, gaItemEndOffset, err := parseGaItems(slotData)
		if err != nil {
			continue
		}

		playerDataStart := gaItemEndOffset + PlayerGameDataPadding
		statsAbsoluteOffset := playerDataStart + PlayerStatsOffset

		if statsAbsoluteOffset+32 > len(slotData) {
			continue
		}

		currentStats := make([]uint32, 8)
		for i := 0; i < 7; i++ {
			currentStats[i] = binary.LittleEndian.Uint32(slotData[statsAbsoluteOffset+(i*4) : statsAbsoluteOffset+(i*4)+4])
		}
		currentStats[7] = binary.LittleEndian.Uint32(slotData[playerDataStart+0x58 : playerDataStart+0x5C])

		classByte := slotData[playerDataStart+PlayerClassOffset]
		className, ok := classIDs[classByte]
		if !ok {
			className = "Wretch"
			classByte = 0x09
		}

		heldRunes := binary.LittleEndian.Uint32(slotData[playerDataStart+PlayerRunesOffset : playerDataStart+PlayerRunesOffset+4])

		activeCharacters = append(activeCharacters, Character{
			Slot:            slotIndex,
			Name:            characterName,
			Level:           menuLevel,
			PlayerDataStart: playerDataStart,
			StatsOffset:     statsAbsoluteOffset,
			CurrentStats:    currentStats,
			ClassByte:       classByte,
			ClassName:       className,
			RealRunes:       heldRunes,
		})
	}

	return activeCharacters
}

func (s *SaveFile) UpdateCharacterRunes(char *Character, newRunes uint32) {
	slotData, err := s.GetSlotData(char.Slot)
	if err != nil {
		return
	}

	realRunesOffset := char.PlayerDataStart + PlayerRunesOffset
	binary.LittleEndian.PutUint32(slotData[realRunesOffset:realRunesOffset+4], newRunes)

	char.RealRunes = newRunes
}

func (s *SaveFile) UpdateCharacterStats(char *Character, newStats []uint32) {
	slotData, err := s.GetSlotData(char.Slot)
	if err != nil {
		return
	}

	var statsSum uint32 = 0
	for i := 0; i < 7; i++ {
		statsSum += newStats[i]
		binary.LittleEndian.PutUint32(slotData[char.StatsOffset+(i*4):char.StatsOffset+(i*4)+4], newStats[i])
	}
	statsSum += newStats[7]
	binary.LittleEndian.PutUint32(slotData[char.PlayerDataStart+0x58:char.PlayerDataStart+0x5C], newStats[7])
	char.CurrentStats = newStats

	var calculatedLevel uint32 = 1
	if statsSum > 79 {
		calculatedLevel = statsSum - 79
	}
	char.Level = calculatedLevel

	binary.LittleEndian.PutUint32(slotData[char.PlayerDataStart+PlayerLevelOffset:char.PlayerDataStart+PlayerLevelOffset+4], calculatedLevel)

	cardLevelOffset := SummaryCardStartOffset + (char.Slot * SummaryCardLength) + SummaryLevelOffset
	binary.LittleEndian.PutUint32(s.Raw[cardLevelOffset:cardLevelOffset+4], calculatedLevel)
}
