package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed data/items.json
var embeddedItemsJSON []byte

type ItemDatabase struct {
	Weapons     map[uint32]string `json:"weapons"`
	Armors      map[uint32]string `json:"armors"`
	Accessories map[uint32]string `json:"accessories"`
	Goods       map[uint32]string `json:"goods"`
	Gems        map[uint32]string `json:"gems"`
}

var DB ItemDatabase

func initItemDB() error {
	dataBytes, err := os.ReadFile("data/items.json")
	if err != nil {
		dataBytes = embeddedItemsJSON
	}

	if len(dataBytes) == 0 {
		return fmt.Errorf("item database is empty (no data/items.json file)")
	}

	if err := json.Unmarshal(dataBytes, &DB); err != nil {
		return fmt.Errorf("error parsing data/items.json: %w", err)
	}

	return nil
}

func getWeaponName(baseID uint32) string {
	if itemName, exists := DB.Weapons[baseID]; exists {
		return itemName
	}
	return fmt.Sprintf("Unknown Weapon (ID: %d)", baseID)
}
